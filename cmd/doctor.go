package cmd

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/toba/jig/internal/nope"
)

var (
	passStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true) // green
	failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // red
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run all doctor checks",
	Long:  "Runs all doctor checks (nope, brew, scoop, zed, cite, cc), reporting results for each.",
	RunE: func(cmd *cobra.Command, args []string) error {
		type check struct {
			name string
			cmd  *cobra.Command
		}
		checks := []check{
			{"nope", nopeDoctorCmd},
			{"brew", brewDoctorCmd},
			{"scoop", scoopDoctorCmd},
			{"zed", zedDoctorCmd},
			{"cite", citeDoctorCmd},
			{"cc", ccDoctorCmd},
		}

		var failed int
		for _, c := range checks {
			fmt.Printf("%s ... ", c.name)
			err := c.cmd.RunE(c.cmd, nil)
			if err != nil {
				fmt.Println(failStyle.Render("FAIL"))
				fmt.Printf("  %s\n", err)
				failed++
			} else {
				fmt.Println(passStyle.Render("ok"))
			}
		}

		if failed > 0 {
			fmt.Printf("\n%s\n", failStyle.Render(fmt.Sprintf("%d of %d checks failed", failed, len(checks))))
			return nope.ExitError{Code: 1}
		}
		fmt.Printf("\n%s\n", passStyle.Render("All checks passed"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
