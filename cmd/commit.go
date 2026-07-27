package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	commitpkg "github.com/toba/jig/internal/commit"
	"github.com/toba/jig/internal/config"
	"github.com/toba/jig/internal/nope"
)

var commitCmd = &cobra.Command{
	Use:           "commit",
	Short:         "Two-phase commit workflow: gather context, then apply",
	SilenceUsage:  true,
	SilenceErrors: true,
}

var gatherCmd = &cobra.Command{
	Use:   "gather",
	Short: "Stage changes and output context for commit message authoring",
	Long: `Stages all changes (git add -A), checks for gitignore candidates,
then outputs staged files, diff, latest version tag, and recent commits.

Exit codes:
  0  Success — context printed
  2  Gitignore candidates found — review before committing`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Check for gitignore candidates.
		candidates, err := commitpkg.GitignoreCandidates()
		if err != nil {
			return err
		}
		if len(candidates) > 0 {
			fmt.Println("GITIGNORE_CANDIDATES:")
			for _, c := range candidates {
				fmt.Println(c)
			}
			fmt.Println()
			fmt.Println("These untracked files may belong in .gitignore.")
			return nope.ExitError{Code: 2}
		}

		// 2. Todo sync (if configured) — off by default; opt in with --sync.
		//    The full external sync (e.g. ClickUp) is slow and does not belong on
		//    the commit hot path; run `jig todo sync` on its own cadence instead.
		if commitSync {
			syncTodoIfConfigured(cmd)
		}

		// 3. Stage all changes.
		status, err := commitpkg.StageAll()
		if err != nil {
			return err
		}
		fmt.Println("STAGED:")
		if status != "" {
			fmt.Println(status)
		}

		// 4. Diff.
		diff, err := commitpkg.Diff()
		if err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("DIFF:")
		if diff != "" {
			fmt.Println(diff)
		}

		// 5. Latest version tag.
		tag, err := commitpkg.LatestTag()
		if err != nil {
			return err
		}
		fmt.Println()
		if tag == "" {
			fmt.Println("LATEST_VERSION: (none)")
		} else {
			fmt.Println("LATEST_VERSION:", tag)
		}

		// 6. Recent commits (for commit message style reference).
		log, err := commitpkg.RecentCommits(tag)
		if err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("RECENT_COMMITS:")
		if log != "" {
			fmt.Println(log)
		}

		return nil
	},
}

var (
	applyMessage string
	applyVersion string
	applyPush    bool
	commitSync   bool
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Commit staged changes with optional tag and push",
	Long: `Creates a git commit from staged changes. Optionally tags a version
and pushes to the remote.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Sync todo before commit — off by default; opt in with --sync.
		if commitSync {
			syncTodoIfConfigured(cmd)
		}

		// Re-stage .issues/ in case sync modified files after gather staged them.
		if err := commitpkg.RestageIssues(); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: restage issues: %v\n", err) //nolint:errcheck // warning output
		}

		// 2. Commit (skip if nothing staged and push was requested).
		staged, err := commitpkg.HasStagedChanges()
		if err != nil {
			return err
		}
		if staged {
			if err := commitpkg.Commit(applyMessage); err != nil {
				return err
			}
			fmt.Println("Committed.")
		} else if !applyPush {
			// Nothing staged and no push — fail like git commit would.
			return errors.New("nothing to commit (use --push to push existing commits)")
		} else {
			fmt.Println("Nothing to commit.")
		}

		// 3. Tag if version provided.
		if applyVersion != "" {
			if err := commitpkg.Tag(applyVersion); err != nil {
				return err
			}
			fmt.Printf("Tagged %s.\n", applyVersion)
		}

		// 4. Push if requested.
		if applyPush {
			if err := commitpkg.Push(); err != nil {
				return err
			}
			fmt.Println("Pushed.")
		}

		// 5. Final status.
		status, err := commitpkg.Status()
		if err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("STATUS:")
		if status != "" {
			fmt.Println(status)
		} else {
			fmt.Println("(clean)")
		}

		return nil
	},
}

func init() {
	applyCmd.Flags().StringVarP(&applyMessage, "message", "m", "", "commit message (required)")
	_ = applyCmd.MarkFlagRequired("message")
	applyCmd.Flags().StringVarP(&applyVersion, "version", "v", "", "version tag to create")
	applyCmd.Flags().BoolVar(&applyPush, "push", false, "push commits and tags after committing")

	// Off by default: the external issue sync (ClickUp/GitHub) is slow and does
	// not belong on the commit hot path. Opt in per-invocation with --sync.
	gatherCmd.Flags().BoolVar(&commitSync, "sync", false, "sync issues to external integrations before staging")
	applyCmd.Flags().BoolVar(&commitSync, "sync", false, "sync issues to external integrations before committing")

	commitCmd.AddCommand(gatherCmd)
	commitCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(commitCmd)
}

// syncTodoIfConfigured runs todo sync if .jig.yaml has a sync section configured.
// Errors are logged to stderr but not propagated — sync is best-effort during commits.
func syncTodoIfConfigured(cmd *cobra.Command) {
	if !hasTodoSync(configPath()) {
		return
	}
	if err := initTodoCore(cmd); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: todo sync init: %v\n", err) //nolint:errcheck // warning output
		return
	}
	if err := runSync(cmd, nil); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: todo sync: %v\n", err) //nolint:errcheck // warning output
	}
}

// hasTodoSync checks whether .jig.yaml has a todo.sync section.
func hasTodoSync(path string) bool {
	doc, err := config.LoadDocument(path)
	if err != nil {
		return false
	}
	todoNode := config.FindKey(doc.Root, "todo")
	if todoNode == nil {
		return false
	}
	return config.FindKey(todoNode, "sync") != nil
}
