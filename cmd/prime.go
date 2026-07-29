package cmd

import (
	_ "embed"
	"os"
	"text/template"

	"github.com/spf13/cobra"
	todoconfig "github.com/toba/jig/internal/todo/config"
)

//go:embed todo_prompt.tmpl
var agentPromptTemplate string

// promptData holds all data needed to render the prompt template.
type promptData struct {
	Types           []todoconfig.TypeConfig
	Statuses        []todoconfig.StatusConfig
	Priorities      []todoconfig.PriorityConfig
	Tags            []todoconfig.TagConfig
	HasSync         bool
	SyncNames       []string
	HasGitHubSync   bool
	ReviewEnabled   bool
	DraftEnabled    bool
	DeferredEnabled bool
}

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output instructions for AI coding agents",
	Long:  `Outputs a prompt that primes AI coding agents on how to use the issues CLI to manage project issues.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var primeCfg *todoconfig.Config
		if todoDataPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return nil
			}
			configFile, err := todoconfig.FindConfig(cwd)
			if err != nil || configFile == "" {
				return nil
			}
			// A .jig.yaml may exist purely for cite/nope/brew/etc. If it has
			// no todo section, todo functionality is irrelevant to this
			// project, so emit nothing rather than priming agents with an
			// issue-tracking guide they can't use.
			if !todoconfig.HasTodoSection(configFile) {
				return nil
			}
			primeCfg, _ = todoconfig.Load(configFile)
		} else {
			cp := configPath()
			primeCfg, _ = todoconfig.Load(cp)
		}

		tmpl, err := template.New("prompt").Parse(agentPromptTemplate)
		if err != nil {
			return err
		}

		data := promptData{
			Types:           todoconfig.DefaultTypes,
			Priorities:      todoconfig.DefaultPriorities,
			ReviewEnabled:   true,
			DraftEnabled:    true,
			DeferredEnabled: true,
		}

		// Filter the listed statuses to those enabled for this project,
		// and surface a few flags the template uses for branching.
		if primeCfg != nil {
			for _, s := range todoconfig.DefaultStatuses {
				if primeCfg.IsStatusEnabled(s.Name) {
					data.Statuses = append(data.Statuses, s)
				}
			}
			data.ReviewEnabled = primeCfg.IsStatusEnabled(todoconfig.StatusReview)
			data.DraftEnabled = primeCfg.IsStatusEnabled(todoconfig.StatusDraft)
			data.DeferredEnabled = primeCfg.IsStatusEnabled(todoconfig.StatusDeferred)
		} else {
			data.Statuses = todoconfig.DefaultStatuses
		}

		if primeCfg != nil && len(primeCfg.Tags) > 0 {
			data.Tags = primeCfg.Tags
		}

		if primeCfg != nil && primeCfg.Sync != nil {
			data.HasSync = true
			for name := range primeCfg.Sync {
				data.SyncNames = append(data.SyncNames, name)
				if name == "github" {
					data.HasGitHubSync = true
				}
			}
		}

		return tmpl.Execute(os.Stdout, data)
	},
}

func init() {
	rootCmd.AddCommand(primeCmd)
}
