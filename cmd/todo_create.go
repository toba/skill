package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	todoconfig "github.com/toba/jig/internal/todo/config"
	"github.com/toba/jig/internal/todo/graph"
	"github.com/toba/jig/internal/todo/graph/model"
	"github.com/toba/jig/internal/todo/output"
	"github.com/toba/jig/internal/todo/ui"
)

var (
	createStatus     string
	createType       string
	createPriority   string
	createMilestone  string
	createExternalID string
	createBody       string
	createBodyFile   string
	createTag        []string
	createDue        string
	createParent     string
	createBlocking   []string
	createBlockedBy  []string
	createJSON       bool
)

var createCmd = &cobra.Command{
	Use:     "create [title]",
	Aliases: []string{"c", "new"},
	Short:   "Create a new issue",
	Long:    `Creates a new issue (issue) with a generated ID and optional title.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		if title == "" {
			title = "Untitled"
		}

		// Validate inputs
		if createStatus != "" {
			if !todoCfg.IsValidStatus(createStatus) {
				return cmdError(createJSON, output.ErrInvalidStatus, "invalid status: %s (must be %s)", createStatus, todoCfg.StatusList())
			}
			if !todoCfg.IsStatusEnabled(createStatus) {
				return cmdError(createJSON, output.ErrInvalidStatus, "status %q is disabled in this project (enabled: %s)", createStatus, todoCfg.EnabledStatusList())
			}
		}
		if createType != "" && !todoCfg.IsValidType(createType) {
			return cmdError(createJSON, output.ErrValidation, "invalid type: %s (must be %s)", createType, todoCfg.TypeList())
		}
		if createPriority != "" && !todoCfg.IsValidPriority(createPriority) {
			return cmdError(createJSON, output.ErrValidation, "invalid priority: %s (must be %s)", createPriority, todoCfg.PriorityList())
		}

		body, err := resolveContent(createBody, createBodyFile)
		if err != nil {
			return cmdError(createJSON, output.ErrFileError, "%s", err)
		}

		// Build GraphQL input
		input := model.CreateIssueInput{Title: title}
		if createStatus != "" {
			input.Status = &createStatus
		} else {
			defaultStatus := todoCfg.GetDefaultStatus()
			input.Status = &defaultStatus
		}
		if createType != "" {
			input.Type = &createType
		} else {
			defaultType := todoCfg.GetDefaultType()
			input.Type = &defaultType
		}
		if createPriority != "" {
			input.Priority = &createPriority
		}
		if createMilestone != "" {
			input.Milestone = &createMilestone
		}
		if createExternalID != "" {
			input.ExternalID = &createExternalID
		}
		if body != "" {
			input.Body = &body
		}
		if len(createTag) > 0 {
			input.Tags = createTag
		}
		if createDue != "" {
			input.Due = &createDue
		}
		if createParent != "" {
			input.Parent = &createParent
		}
		if len(createBlocking) > 0 {
			input.Blocking = createBlocking
		}
		if len(createBlockedBy) > 0 {
			input.BlockedBy = createBlockedBy
		}

		// Create via GraphQL mutation
		resolver := &graph.Resolver{Core: todoStore}
		b, err := resolver.Mutation().CreateIssue(context.Background(), input)
		if err != nil {
			return cmdError(createJSON, output.ErrFileError, "failed to create issue: %v", err)
		}

		if createJSON {
			return output.Success(b, "Issue created")
		}

		fmt.Println(ui.Success.Render("Created ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		return nil
	},
}

func init() {
	statusNames := todoconfig.DefaultStatusNames()
	typeNames := todoconfig.DefaultTypeNames()
	priorityNames := todoconfig.DefaultPriorityNames()

	createCmd.Flags().StringVarP(&createStatus, "status", "s", "", "Initial status ("+strings.Join(statusNames, ", ")+")")
	createCmd.Flags().StringVarP(&createType, "type", "t", "", "issue type ("+strings.Join(typeNames, ", ")+")")
	createCmd.Flags().StringVarP(&createPriority, "priority", "p", "", "Priority level ("+strings.Join(priorityNames, ", ")+")")
	createCmd.Flags().StringVar(&createMilestone, "milestone", "", "Milestone ID to assign this issue to")
	createCmd.Flags().StringVar(&createExternalID, "external-id", "", "External identifier referencing this issue in another system")
	createCmd.Flags().StringVarP(&createBody, "body", "d", "", "Body content (use '-' to read from stdin)")
	createCmd.Flags().StringVar(&createBodyFile, "body-file", "", "Read body from file (use '-' to read from stdin)")
	createCmd.Flags().StringArrayVar(&createTag, "tag", nil, "Add tag (can be repeated)")
	createCmd.Flags().StringVar(&createDue, "due", "", "Due date (YYYY-MM-DD)")
	createCmd.Flags().StringVar(&createParent, "parent", "", "Parent issue ID")
	createCmd.Flags().StringArrayVar(&createBlocking, "blocking", nil, "ID of issue this blocks (can be repeated)")
	createCmd.Flags().StringArrayVar(&createBlockedBy, "blocked-by", nil, "ID of issue that blocks this one (can be repeated)")
	createCmd.Flags().BoolVar(&createJSON, "json", false, "Output as JSON")
	createCmd.MarkFlagsMutuallyExclusive("body", "body-file")
	todoCmd.AddCommand(createCmd)
}
