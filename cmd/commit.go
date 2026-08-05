package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	commitpkg "github.com/toba/jig-go/v4/internal/commit"
	"github.com/toba/jig-go/v4/internal/nope"
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

		// 2. Stage all changes.
		status, err := commitpkg.StageAll()
		if err != nil {
			return err
		}
		fmt.Println("STAGED:")
		if status != "" {
			fmt.Println(status)
		}

		// 3. Diff.
		diff, err := commitpkg.Diff()
		if err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("DIFF:")
		if diff != "" {
			fmt.Println(diff)
		}

		// 4. Latest version tag.
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

		// 5. Recent commits (for commit message style reference).
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
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Commit staged changes with optional tag and push",
	Long: `Creates a git commit from staged changes. Optionally tags a version
and pushes to the remote.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Commit (skip if nothing staged and push was requested).
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

		// 2. Tag if version provided.
		if applyVersion != "" {
			if err := commitpkg.Tag(applyVersion); err != nil {
				return err
			}
			fmt.Printf("Tagged %s.\n", applyVersion)
		}

		// 3. Push if requested.
		if applyPush {
			if err := commitpkg.Push(); err != nil {
				return err
			}
			fmt.Println("Pushed.")
		}

		// 4. Final status.
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

	commitCmd.AddCommand(gatherCmd)
	commitCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(commitCmd)
}
