package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/config"
)

var (
	updateBranch    string
	updateTrack     string
	updateScope     string
	updateNotes     string
	updateRepo      string
	updatePathsHigh []string
	updatePathsMed  []string
	updatePathsLow  []string
	clearTrack      bool
	clearScope      bool
	clearNotes      bool
	clearPathsHigh  bool
	clearPathsMed   bool
	clearPathsLow   bool
)

var citeUpdateCmd = &cobra.Command{
	Use:   "update <source>",
	Short: "Update fields of an existing citation",
	Long:  "Modify the branch, track, scope, notes, or path globs of an existing citation in .jig.yaml.",
	Args:  cobra.ExactArgs(1),
	RunE:  runCiteUpdate,
}

func init() {
	citeUpdateCmd.Flags().StringVar(&updateBranch, "branch", "", "tracked branch")
	citeUpdateCmd.Flags().StringVar(&updateTrack, "track", "", `tracking mode ("releases" or empty)`)
	citeUpdateCmd.Flags().StringVar(&updateScope, "scope", "", "scope description")
	citeUpdateCmd.Flags().StringVar(&updateNotes, "notes", "", "notes text")
	citeUpdateCmd.Flags().StringVar(&updateRepo, "repo", "", "rename the repo identifier")
	citeUpdateCmd.Flags().StringSliceVar(&updatePathsHigh, "paths-high", nil, "high-priority path globs (replaces existing)")
	citeUpdateCmd.Flags().StringSliceVar(&updatePathsMed, "paths-medium", nil, "medium-priority path globs (replaces existing)")
	citeUpdateCmd.Flags().StringSliceVar(&updatePathsLow, "paths-low", nil, "low-priority path globs (replaces existing)")
	citeUpdateCmd.Flags().BoolVar(&clearTrack, "clear-track", false, "clear tracking mode (revert to branch tracking)")
	citeUpdateCmd.Flags().BoolVar(&clearScope, "clear-scope", false, "clear scope")
	citeUpdateCmd.Flags().BoolVar(&clearNotes, "clear-notes", false, "clear notes")
	citeUpdateCmd.Flags().BoolVar(&clearPathsHigh, "clear-paths-high", false, "clear high-priority paths")
	citeUpdateCmd.Flags().BoolVar(&clearPathsMed, "clear-paths-medium", false, "clear medium-priority paths")
	citeUpdateCmd.Flags().BoolVar(&clearPathsLow, "clear-paths-low", false, "clear low-priority paths")
	citeCmd.AddCommand(citeUpdateCmd)
}

func runCiteUpdate(cmd *cobra.Command, args []string) error {
	path := configPath()
	doc, c, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	src := config.FindSource(c, args[0])
	if src == nil {
		return fmt.Errorf("source %q not found in config", args[0])
	}

	changed := false

	if cmd.Flags().Changed("branch") {
		src.Branch = updateBranch
		changed = true
	}
	if cmd.Flags().Changed("track") {
		src.Track = updateTrack
		changed = true
	}
	if clearTrack {
		src.Track = ""
		changed = true
	}
	if cmd.Flags().Changed("scope") {
		src.Scope = updateScope
		changed = true
	}
	if clearScope {
		src.Scope = ""
		changed = true
	}
	if cmd.Flags().Changed("notes") {
		src.Notes = updateNotes
		changed = true
	}
	if clearNotes {
		src.Notes = ""
		changed = true
	}
	if cmd.Flags().Changed("repo") {
		src.Repo = updateRepo
		changed = true
	}
	if cmd.Flags().Changed("paths-high") {
		src.Paths.High = updatePathsHigh
		changed = true
	}
	if clearPathsHigh {
		src.Paths.High = nil
		changed = true
	}
	if cmd.Flags().Changed("paths-medium") {
		src.Paths.Medium = updatePathsMed
		changed = true
	}
	if clearPathsMed {
		src.Paths.Medium = nil
		changed = true
	}
	if cmd.Flags().Changed("paths-low") {
		src.Paths.Low = updatePathsLow
		changed = true
	}
	if clearPathsLow {
		src.Paths.Low = nil
		changed = true
	}

	if !changed {
		fmt.Fprintln(os.Stderr, "no changes specified")
		return nil
	}

	if err := config.Save(doc, c); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Updated %s in %s\n", src.Repo, path)
	return nil
}
