package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
	"github.com/toba/jig/internal/todo/graph"
	"github.com/toba/jig/internal/todo/issue"
	"github.com/toba/jig/internal/todo/output"
	"github.com/toba/jig/internal/todo/ui"
)

var (
	showJSON     bool
	showRaw      bool
	showBodyOnly bool
	showETagOnly bool
)

var showCmd = &cobra.Command{
	Use:   "show <id> [id...]",
	Short: "Show an issue's contents",
	Long:  `Displays the full contents of one or more issues, including front matter and body.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolver := &graph.Resolver{Core: todoStore}

		var issues []*issue.Issue
		for _, id := range args {
			b, err := resolver.Query().Issue(context.Background(), id)
			if err != nil {
				if showJSON {
					return output.Error(output.ErrNotFound, err.Error())
				}
				return fmt.Errorf("failed to find issue: %w", err)
			}
			if b == nil {
				if showJSON {
					return output.Error(output.ErrNotFound, "issue not found: "+id)
				}
				return fmt.Errorf("issue not found: %s", id)
			}
			issues = append(issues, b)
		}

		if showJSON {
			if len(issues) == 1 {
				return output.SuccessSingle(issues[0])
			}
			return output.SuccessMultiple(issues)
		}

		if showRaw {
			for i, b := range issues {
				if i > 0 {
					fmt.Print("\n---\n\n")
				}
				content, err := b.Render()
				if err != nil {
					return fmt.Errorf("failed to render issue: %w", err)
				}
				fmt.Print(string(content))
			}
			return nil
		}

		if showBodyOnly {
			for i, b := range issues {
				if i > 0 {
					fmt.Print("\n---\n\n")
				}
				fmt.Print(b.Body)
			}
			return nil
		}

		if showETagOnly {
			for i, b := range issues {
				if i > 0 {
					fmt.Println()
				}
				fmt.Print(b.ETag())
			}
			return nil
		}

		// Route styled output through a color-profile writer so it adapts to
		// the destination: full color on an interactive terminal, and ANSI
		// stripped when piped (non-TTY) or when NO_COLOR is set. This keeps
		// `jig todo show <id> | cat` readable for agents without forcing them
		// to fall back to the raw markdown file.
		out := colorprofile.NewWriter(os.Stdout, os.Environ())
		color := out.Profile > colorprofile.ASCII

		for i, b := range issues {
			if i > 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, ui.Muted.Render(strings.Repeat("═", 60)))
				fmt.Fprintln(out)
			}
			writeStyledIssue(out, b, color)
		}

		return nil
	},
}

// renderIssue renders an issue to a string. When color is false, all ANSI
// escape sequences are stripped so the output is plain text (used for piped /
// non-TTY destinations and in tests).
func renderIssue(b *issue.Issue, color bool) string {
	var buf bytes.Buffer
	var w io.Writer = &buf
	if !color {
		w = &colorprofile.Writer{Forward: &buf, Profile: colorprofile.NoTTY}
	}
	writeStyledIssue(w, b, color)
	return buf.String()
}

func writeStyledIssue(w io.Writer, b *issue.Issue, color bool) {
	statusCfg := todoCfg.GetStatus(b.Status)
	statusColor := "gray"
	if statusCfg != nil {
		statusColor = statusCfg.Color
	}
	isArchive := todoCfg.IsArchiveStatus(b.Status)

	var header strings.Builder
	header.WriteString(ui.ID.Render(b.ID))
	header.WriteString(" ")
	header.WriteString(ui.RenderStatusWithColor(b.Status, statusColor, isArchive))
	if b.Priority != "" {
		priorityCfg := todoCfg.GetPriority(b.Priority)
		priorityColor := "gray"
		if priorityCfg != nil {
			priorityColor = priorityCfg.Color
		}
		header.WriteString(" ")
		header.WriteString(ui.RenderPriorityWithColor(b.Priority, priorityColor))
	}
	if b.Due != nil {
		header.WriteString(" ")
		header.WriteString(ui.Muted.Render("due:" + b.Due.String()))
	}
	if b.ExternalID != "" {
		header.WriteString(" ")
		header.WriteString(ui.Muted.Render("ext:" + b.ExternalID))
	}
	if len(b.Tags) > 0 {
		header.WriteString("  ")
		header.WriteString(ui.Muted.Render(strings.Join(b.Tags, ", ")))
	}
	header.WriteString("\n")
	header.WriteString(ui.Title.Render(b.Title))

	if b.Parent != "" || len(b.Blocking) > 0 || len(b.BlockedBy) > 0 {
		header.WriteString("\n")
		header.WriteString(ui.Muted.Render(strings.Repeat("─", 50)))
		header.WriteString("\n")
		header.WriteString(formatRelationships(b))
	}

	header.WriteString("\n")
	header.WriteString(ui.Muted.Render(strings.Repeat("─", 50)))

	headerBox := lipgloss.NewStyle().
		MarginBottom(1).
		Render(header.String())

	fmt.Fprintln(w, headerBox)

	if b.Body != "" {
		// Use the clean "notty" style (no background padding) when color is
		// disabled, otherwise the terminal's configured style.
		styleOpt := glamour.WithEnvironmentConfig()
		if !color {
			styleOpt = glamour.WithStandardStyle("notty")
		}
		renderer, err := glamour.NewTermRenderer(
			styleOpt,
			glamour.WithWordWrap(80),
		)
		if err != nil {
			fmt.Fprintf(w, "failed to create renderer: %v\n", err)
			return
		}

		rendered, err := renderer.Render(b.Body)
		if err != nil {
			fmt.Fprintf(w, "failed to render markdown: %v\n", err)
			return
		}

		_, _ = fmt.Fprint(w, rendered)
	}
}

func formatRelationships(b *issue.Issue) string {
	var parts []string

	if b.Parent != "" {
		parts = append(parts, fmt.Sprintf("%s %s",
			ui.Muted.Render("parent:"),
			ui.ID.Render(b.Parent)))
	}
	for _, target := range b.Blocking {
		parts = append(parts, fmt.Sprintf("%s %s",
			ui.Muted.Render("blocking:"),
			ui.ID.Render(target)))
	}
	for _, blocker := range b.BlockedBy {
		parts = append(parts, fmt.Sprintf("%s %s",
			ui.Muted.Render("blocked by:"),
			ui.ID.Render(blocker)))
	}
	return strings.Join(parts, "\n")
}

func init() {
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&showRaw, "raw", false, "Output raw markdown without styling")
	showCmd.Flags().BoolVar(&showBodyOnly, "body-only", false, "Output only the body content")
	showCmd.Flags().BoolVar(&showETagOnly, "etag-only", false, "Output only the etag")
	showCmd.MarkFlagsMutuallyExclusive("json", "raw", "body-only", "etag-only")
	todoCmd.AddCommand(showCmd)
}
