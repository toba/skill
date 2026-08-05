package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/cite"
	"github.com/toba/jig-go/v4/internal/config"
	"github.com/toba/jig-go/v4/internal/github"
)

var addWrite bool
var addReleases bool

var addCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Inspect a repository and suggest citation config",
	Long:  "Inspect a repository via GitHub API or git clone, detect its language, and suggest path classification globs for the citations config.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func init() {
	addCmd.Flags().BoolVarP(&addWrite, "write", "w", false, "append the suggested source to .jig.yaml")
	addCmd.Flags().BoolVar(&addReleases, "releases", false, "track releases instead of branch commits")
	citeCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	arg := cite.ParseRepoArg(args[0])
	client := github.NewClient()

	src, err := cite.Inspect(client, arg)
	if err != nil {
		return err
	}

	if addReleases {
		src.Track = "releases"
	}

	yamlStr, err := cite.FormatSourceYAML(src)
	if err != nil {
		return err
	}

	if addWrite {
		path := configPath()
		doc, loadErr := config.LoadDocument(path)
		if loadErr != nil {
			// File doesn't exist — create a minimal document.
			if os.IsNotExist(loadErr) {
				if err := os.WriteFile(path, []byte("citations: []\n"), 0o644); err != nil {
					return fmt.Errorf("creating %s: %w", path, err)
				}
				doc, loadErr = config.LoadDocument(path)
			}
			if loadErr != nil {
				return fmt.Errorf("loading config: %w", loadErr)
			}
		}
		if config.HasSource(doc, src.Repo) {
			fmt.Fprintf(os.Stderr, "%s is already cited in %s\n", src.Repo, path)
			return nil
		}
		if err := config.AppendSource(doc, *src); err != nil {
			return fmt.Errorf("appending source: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Added %s to %s\n", src.Repo, path)
		return nil
	}

	fmt.Print(yamlStr)
	return nil
}
