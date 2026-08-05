package cmd

import (
	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/cite"
	"github.com/toba/jig-go/v4/internal/config"
	"github.com/toba/jig-go/v4/internal/github"
	"github.com/toba/jig-go/v4/internal/nope"
)

var citeDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify cited repos have license attribution",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config; if no citations section, pass empty sources.
		var sources []config.Source
		if _, c, err := config.Load(configPath()); err == nil && c != nil {
			sources = []config.Source(*c)
		}

		code := cite.RunDoctor(cite.DoctorOpts{
			Sources: sources,
			Client:  github.NewClient(),
		})
		if code != 0 {
			return nope.ExitError{Code: code}
		}
		return nil
	},
}

func init() {
	citeCmd.AddCommand(citeDoctorCmd)
}
