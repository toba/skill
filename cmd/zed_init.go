package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/zed"
)

var (
	zedInitExt       string
	zedInitTag       string
	zedInitRepo      string
	zedInitDesc      string
	zedInitLSPName   string
	zedInitLanguages string
	zedInitDryRun    bool
)

var zedInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up a new Zed extension with automated releases",
	Long: `Creates a companion Zed extension repo, pushes an initial scaffold (extension.toml,
Cargo.toml, src/lib.rs, bump scripts, workflow, LICENSE, README), and injects a
sync-extension job into the source repo's release workflow.

This is a one-time setup command. After running it, extension updates happen
automatically via CI when you push a new tag.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ext := resolveExt(zedInitExt, configPath())
		if ext == "" {
			return fmt.Errorf("--ext is required (or set zed_extension in %s)", configPath())
		}

		if zedInitLanguages == "" {
			return errors.New("--languages is required")
		}

		opts := zed.InitOpts{
			Ext:       ext,
			Tag:       zedInitTag,
			Repo:      zedInitRepo,
			Desc:      zedInitDesc,
			LSPName:   zedInitLSPName,
			Languages: zedInitLanguages,
			DryRun:    zedInitDryRun,
		}

		result, err := zed.RunInit(opts)
		if err != nil {
			return err
		}

		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		if zedInitDryRun {
			header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
			dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

			fmt.Println(header.Render("=== extension.toml ==="))
			fmt.Println(result.ExtensionToml)

			fmt.Println(header.Render("=== Cargo.toml ==="))
			fmt.Println(result.CargoToml)

			fmt.Println(header.Render("=== src/lib.rs ==="))
			fmt.Println(result.LibRs)

			fmt.Println(header.Render("=== scripts/bump-version.sh ==="))
			fmt.Println(result.BumpScript)

			fmt.Println(header.Render("=== .github/workflows/bump-version.yml ==="))
			fmt.Println(result.BumpWorkflow)

			fmt.Println(header.Render("=== LICENSE ==="))
			fmt.Println(result.License)

			fmt.Println(header.Render("=== README.md ==="))
			fmt.Println(result.Readme)

			fmt.Println(header.Render("=== Workflow Job (sync-extension) ==="))
			fmt.Println(result.WorkflowJob)

			fmt.Println(dim.Render("(dry run — nothing was created)"))
			return nil
		}

		ok := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

		fmt.Println(ok.Render("Extension initialized"))
		fmt.Printf("  ext      %s\n", result.Ext)
		fmt.Printf("  repo     %s\n", result.Repo)
		fmt.Printf("  tag      %s\n", result.Tag)
		fmt.Printf("  lsp      %s\n", result.LSPName)

		if result.WorkflowMod {
			fmt.Println(ok.Render("\nWorkflow updated"))
			fmt.Println("  .github/workflows/release.yml now includes sync-extension job")
		}

		fmt.Println(dim.Render("\nReminders:"))
		fmt.Println(dim.Render("  1. Add EXTENSION_PAT secret to " + result.Repo))
		fmt.Println(dim.Render("     GitHub PAT with Contents write access to " + result.Ext))
		fmt.Println(dim.Render("  2. Run 'cargo generate-lockfile' in the extension repo to create Cargo.lock"))

		return nil
	},
}

func init() {
	zedInitCmd.Flags().StringVar(&zedInitExt, "ext", "", "extension repo (default: zed_extension from .jig.yaml)")
	zedInitCmd.Flags().StringVar(&zedInitTag, "tag", "", "release tag (default: latest release)")
	zedInitCmd.Flags().StringVar(&zedInitRepo, "repo", "", "source repo (default: current repo via gh)")
	zedInitCmd.Flags().StringVar(&zedInitDesc, "desc", "", "extension description (default: repo description)")
	zedInitCmd.Flags().StringVar(&zedInitLSPName, "lsp-name", "", "LSP binary name (default: source repo name)")
	zedInitCmd.Flags().StringVar(&zedInitLanguages, "languages", "", "comma-separated language list (required)")
	zedInitCmd.Flags().BoolVar(&zedInitDryRun, "dry-run", false, "show what would be created without doing it")
	zedCmd.AddCommand(zedInitCmd)
}
