package cmd

import (
	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/nope"
)

var nopeDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate nope configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		code := nope.RunDoctor()
		if code != 0 {
			return nope.ExitError{Code: code}
		}
		return nil
	},
}

func init() {
	nopeCmd.AddCommand(nopeDoctorCmd)
}
