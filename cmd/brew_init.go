package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/brew"
	"github.com/toba/jig-go/v4/internal/config"
)

var (
	brewInitTap     string
	brewInitTag     string
	brewInitRepo    string
	brewInitDesc    string
	brewInitLicense string
	brewInitDryRun  bool
)

var brewInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Add a formula to a shared Homebrew tap",
	Long: `Pushes a formula to an existing shared Homebrew tap repo and injects an
update-homebrew job into the source repo's release workflow.

The tap repo (e.g. org/homebrew-tap) must already exist on GitHub.
After running this, tap updates happen automatically via CI on new tags.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tap, err := resolveTap(brewInitTap, configPath())
		if err != nil {
			return err
		}

		opts := brew.InitOpts{
			Tap:     tap,
			Tag:     brewInitTag,
			Repo:    brewInitRepo,
			Desc:    brewInitDesc,
			License: brewInitLicense,
			DryRun:  brewInitDryRun,
		}

		result, err := brew.RunInit(opts)
		if err != nil {
			return err
		}

		// Add brew to packages in .jig.yaml if not already present.
		if !brewInitDryRun {
			addPackage(configPath(), "brew")
		}

		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		if brewInitDryRun {
			header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
			dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

			fmt.Println(header.Render("=== Formula ==="))
			fmt.Println(result.Formula)

			fmt.Println(header.Render("=== Workflow Job ==="))
			fmt.Println(result.WorkflowJob)

			fmt.Println(dim.Render("(dry run — nothing was created)"))
			return nil
		}

		ok := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

		fmt.Println(ok.Render("Formula pushed to tap"))
		fmt.Printf("  tap      %s\n", result.Tap)
		fmt.Printf("  repo     %s\n", result.Repo)
		fmt.Printf("  tool     %s\n", result.Tool)
		fmt.Printf("  tag      %s\n", result.Tag)
		fmt.Printf("  sha256   %s\n", dim.Render(result.SHA256[:12]+"…"))

		if result.WorkflowMod {
			fmt.Println(ok.Render("\nWorkflow updated"))
			fmt.Println("  .github/workflows/release.yml now includes update-homebrew job")
		}

		fmt.Println(dim.Render("\nReminder: add HOMEBREW_TAP_TOKEN secret to " + result.Repo))
		fmt.Println(dim.Render("  GitHub PAT with Contents write access to " + result.Tap))

		return nil
	},
}

func init() {
	brewInitCmd.Flags().StringVar(&brewInitTap, "tap", "", "tap repo (default: owner/homebrew-tap)")
	brewInitCmd.Flags().StringVar(&brewInitTag, "tag", "", "release tag (default: latest release)")
	brewInitCmd.Flags().StringVar(&brewInitRepo, "repo", "", "source repo (default: current repo via gh)")
	brewInitCmd.Flags().StringVar(&brewInitDesc, "desc", "", "formula description (default: repo description)")
	brewInitCmd.Flags().StringVar(&brewInitLicense, "license", "", "license identifier (default: from repo)")
	brewInitCmd.Flags().BoolVar(&brewInitDryRun, "dry-run", false, "show what would be created without doing it")
	brewCmd.AddCommand(brewInitCmd)
}

// addPackage adds a package name to the packages list in .jig.yaml.
func addPackage(cfgPath, name string) {
	doc, err := config.LoadDocument(cfgPath)
	if err != nil {
		return
	}
	_ = config.AddPackage(doc, name)
}
