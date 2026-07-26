package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	todoconfig "github.com/toba/jig/internal/todo/config"
	"github.com/toba/jig/internal/todo/core"
	"github.com/toba/jig/internal/todo/graph"
	"github.com/toba/jig/internal/todo/graph/model"
	"github.com/toba/jig/internal/todo/output"
	"github.com/toba/jig/internal/todo/ui"
)

var (
	updateStatus          string
	updateType            string
	updatePriority        string
	updateMilestone       string
	updateExternalID      string
	updateTitle           string
	updateBody            string
	updateBodyFile        string
	updateReplaceBody     string
	updateReplaceBodyFile string
	updateAppendBody      string
	updateBodyReplaceOld  string
	updateBodyReplaceNew  string
	updateBodyAppend      string
	updateBodyCheck       []string
	updateBodyUncheck     []string
	updateDue             string
	updateParent          string
	updateRemoveParent    bool
	updateBlocking        []string
	updateRemoveBlocking  []string
	updateBlockedBy       []string
	updateRemoveBlockedBy []string
	updateTag             []string
	updateRemoveTag       []string
	updateIfMatch         string
	todoUpdateJSON        bool
)

var todoUpdateCmd = &cobra.Command{
	Use:     "update <id>",
	Aliases: []string{"u"},
	Short:   "Update an issue's properties",
	Long:    `Updates one or more properties of an existing issue.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &graph.Resolver{Core: todoStore}

		b, err := resolver.Query().Issue(ctx, args[0])
		if err != nil {
			return cmdError(todoUpdateJSON, output.ErrNotFound, "failed to find issue: %v", err)
		}

		wasArchived := false
		if b == nil {
			unarchived, unarchiveErr := todoStore.LoadAndUnarchive(args[0])
			if unarchiveErr != nil {
				return cmdError(todoUpdateJSON, output.ErrNotFound, "issue not found: %s", args[0])
			}
			b, err = resolver.Query().Issue(ctx, unarchived.ID)
			if err != nil || b == nil {
				return cmdError(todoUpdateJSON, output.ErrNotFound, "issue not found: %s", args[0])
			}
			wasArchived = true
		}

		var changes []string

		var ifMatch *string
		if updateIfMatch != "" {
			ifMatch = &updateIfMatch
		}

		input, fieldChanges, err := buildUpdateInput(cmd, b.Tags, b.Body)
		if err != nil {
			return cmdError(todoUpdateJSON, output.ErrValidation, "%s", err)
		}
		changes = append(changes, fieldChanges...)

		if ifMatch != nil {
			input.IfMatch = ifMatch
		}

		if hasFieldUpdates(input) {
			b, err = resolver.Mutation().UpdateIssue(ctx, b.ID, input)
			if err != nil {
				return mutationError(todoUpdateJSON, err)
			}
		}

		if len(changes) == 0 {
			return cmdError(todoUpdateJSON, output.ErrValidation,
				"no changes specified (use --status, --type, --priority, --title, --due, --milestone, --external-id, --append-body, --body-replace-old/--body-replace-new, --replace-body, --parent, --blocking, --blocked-by, --tag, or their --remove-* variants)")
		}

		if todoUpdateJSON {
			msg := "Issue updated"
			if wasArchived {
				msg = "Issue unarchived and updated"
			}
			return output.Success(b, msg)
		}

		if wasArchived {
			fmt.Println(ui.Success.Render("Unarchived and updated ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		} else {
			fmt.Println(ui.Success.Render("Updated ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		}
		return nil
	},
}

func buildUpdateInput(cmd *cobra.Command, _ []string, _ string) (model.UpdateIssueInput, []string, error) {
	var input model.UpdateIssueInput
	var changes []string

	if cmd.Flags().Changed("status") {
		if !todoCfg.IsValidStatus(updateStatus) {
			return input, nil, fmt.Errorf("invalid status: %s (must be %s)", updateStatus, todoCfg.StatusList())
		}
		if !todoCfg.IsStatusEnabled(updateStatus) {
			return input, nil, fmt.Errorf("status %q is disabled in this project (enabled: %s)", updateStatus, todoCfg.EnabledStatusList())
		}
		input.Status = &updateStatus
		changes = append(changes, "status")
	}

	if cmd.Flags().Changed("type") {
		if !todoCfg.IsValidType(updateType) {
			return input, nil, fmt.Errorf("invalid type: %s (must be %s)", updateType, todoCfg.TypeList())
		}
		input.Type = &updateType
		changes = append(changes, "type")
	}

	if cmd.Flags().Changed("priority") {
		if !todoCfg.IsValidPriority(updatePriority) {
			return input, nil, fmt.Errorf("invalid priority: %s (must be %s)", updatePriority, todoCfg.PriorityList())
		}
		input.Priority = &updatePriority
		changes = append(changes, "priority")
	}

	if cmd.Flags().Changed("milestone") {
		input.Milestone = &updateMilestone
		changes = append(changes, "milestone")
	}

	if cmd.Flags().Changed("external-id") {
		input.ExternalID = &updateExternalID
		changes = append(changes, "external-id")
	}

	if cmd.Flags().Changed("title") {
		input.Title = &updateTitle
		changes = append(changes, "title")
	}

	if cmd.Flags().Changed("due") {
		input.Due = &updateDue
		changes = append(changes, "due")
	}

	// The legacy --body/--body-file flags silently replaced the entire body, which
	// repeatedly caused accidental loss of existing content. They are retired on
	// update in favor of the explicit --replace-body/--append-body verbs.
	if cmd.Flags().Changed("body") || cmd.Flags().Changed("body-file") {
		return input, nil, errors.New(
			"--body/--body-file were removed from `update` because they replace the entire body. " +
				"Use --append-body to add content, --body-replace-old/--body-replace-new to edit a section, " +
				"or --replace-body/--replace-body-file to intentionally overwrite the whole body")
	}

	appendChanged := cmd.Flags().Changed("append-body") || cmd.Flags().Changed("body-append")

	if cmd.Flags().Changed("replace-body") || cmd.Flags().Changed("replace-body-file") {
		body, err := resolveContent(updateReplaceBody, updateReplaceBodyFile)
		if err != nil {
			return input, nil, err
		}
		input.Body = &body
		changes = append(changes, "body")
	} else if cmd.Flags().Changed("body-replace-old") || appendChanged || len(updateBodyCheck) > 0 || len(updateBodyUncheck) > 0 {
		bodyMod := &model.BodyModification{}

		if cmd.Flags().Changed("body-replace-old") {
			bodyMod.Replace = []*model.ReplaceOperation{
				{
					Old: updateBodyReplaceOld,
					New: updateBodyReplaceNew,
				},
			}
		}

		if len(updateBodyCheck) > 0 {
			bodyMod.Check = updateBodyCheck
		}
		if len(updateBodyUncheck) > 0 {
			bodyMod.Uncheck = updateBodyUncheck
		}

		if appendChanged {
			raw := updateAppendBody
			if cmd.Flags().Changed("body-append") {
				raw = updateBodyAppend
			}
			appendText, err := resolveAppendContent(raw)
			if err != nil {
				return input, nil, err
			}
			bodyMod.Append = &appendText
		}

		input.BodyMod = bodyMod
		changes = append(changes, "body")
	}

	if len(updateTag) > 0 {
		input.AddTags = updateTag
		changes = append(changes, "tags")
	}
	if len(updateRemoveTag) > 0 {
		input.RemoveTags = updateRemoveTag
		changes = append(changes, "tags")
	}

	if cmd.Flags().Changed("parent") {
		input.Parent = &updateParent
		changes = append(changes, "parent")
	} else if updateRemoveParent {
		emptyParent := ""
		input.Parent = &emptyParent
		changes = append(changes, "parent")
	}

	if len(updateBlocking) > 0 {
		input.AddBlocking = updateBlocking
		changes = append(changes, "blocking")
	}
	if len(updateRemoveBlocking) > 0 {
		input.RemoveBlocking = updateRemoveBlocking
		changes = append(changes, "blocking")
	}

	if len(updateBlockedBy) > 0 {
		input.AddBlockedBy = updateBlockedBy
		changes = append(changes, "blocked-by")
	}
	if len(updateRemoveBlockedBy) > 0 {
		input.RemoveBlockedBy = updateRemoveBlockedBy
		changes = append(changes, "blocked-by")
	}

	return input, changes, nil
}

func hasFieldUpdates(input model.UpdateIssueInput) bool {
	return input.Status != nil || input.Type != nil || input.Priority != nil || input.Milestone != nil ||
		input.ExternalID != nil ||
		input.Title != nil || input.Due != nil || input.Body != nil || input.BodyMod != nil || input.Tags != nil ||
		input.AddTags != nil || input.RemoveTags != nil ||
		input.Parent != nil || input.AddBlocking != nil || input.RemoveBlocking != nil ||
		input.AddBlockedBy != nil || input.RemoveBlockedBy != nil
}

func isConflictError(err error) bool {
	_, isMismatch := errors.AsType[*core.ETagMismatchError](err)
	_, isRequired := errors.AsType[*core.ETagRequiredError](err)
	return isMismatch || isRequired
}

func mutationError(jsonOutput bool, err error) error {
	if isConflictError(err) {
		return cmdError(jsonOutput, output.ErrConflict, "%s", err)
	}
	return cmdError(jsonOutput, output.ErrValidation, "%s", err)
}

// registerUpdateFlags binds all `todo update` flags to the given command. Split
// out from init() so tests can build an isolated command and exercise
// buildUpdateInput without polluting the shared todoUpdateCmd flag state.
func registerUpdateFlags(cmd *cobra.Command) {
	statusNames := todoconfig.DefaultStatusNames()
	typeNames := todoconfig.DefaultTypeNames()
	priorityNames := todoconfig.DefaultPriorityNames()

	cmd.Flags().StringVarP(&updateStatus, "status", "s", "", "New status ("+strings.Join(statusNames, ", ")+")")
	cmd.Flags().StringVarP(&updateType, "type", "t", "", "New type ("+strings.Join(typeNames, ", ")+")")
	cmd.Flags().StringVarP(&updatePriority, "priority", "p", "", "New priority ("+strings.Join(priorityNames, ", ")+", or empty to clear)")
	cmd.Flags().StringVar(&updateTitle, "title", "", "New title")
	cmd.Flags().StringVar(&updateMilestone, "milestone", "", "Milestone ID to assign (empty to clear)")
	cmd.Flags().StringVar(&updateExternalID, "external-id", "", "External identifier referencing this issue in another system (empty to clear)")
	cmd.Flags().StringVar(&updateDue, "due", "", "Due date (YYYY-MM-DD, empty to clear)")

	// Whole-body writes. --replace-body is destructive (overwrites everything);
	// --append-body is the safe additive verb. The legacy --body/--body-file are
	// retained (hidden) only so passing them yields a helpful guidance error
	// instead of an opaque "unknown flag".
	cmd.Flags().StringVar(&updateReplaceBody, "replace-body", "", "Overwrite the entire body (use '-' to read from stdin)")
	cmd.Flags().StringVar(&updateReplaceBodyFile, "replace-body-file", "", "Overwrite the entire body from a file (use '-' to read from stdin)")
	cmd.Flags().StringVar(&updateAppendBody, "append-body", "", "Append text to the body (use '-' for stdin)")
	cmd.Flags().StringVarP(&updateBody, "body", "d", "", "Removed: use --append-body or --replace-body")
	cmd.Flags().StringVar(&updateBodyFile, "body-file", "", "Removed: use --append-body or --replace-body-file")
	cmd.Flags().StringVar(&updateBodyAppend, "body-append", "", "Deprecated alias for --append-body")
	_ = cmd.Flags().MarkHidden("body")
	_ = cmd.Flags().MarkHidden("body-file")
	_ = cmd.Flags().MarkHidden("body-append")

	cmd.Flags().StringVar(&updateBodyReplaceOld, "body-replace-old", "", "Substring to find and replace (requires --body-replace-new)")
	cmd.Flags().StringVar(&updateBodyReplaceNew, "body-replace-new", "", "Replacement substring (requires --body-replace-old)")
	cmd.Flags().StringArrayVar(&updateBodyCheck, "body-check", nil, "Check a checkbox item by substring match (can be repeated)")
	cmd.Flags().StringArrayVar(&updateBodyUncheck, "body-uncheck", nil, "Uncheck a checkbox item by substring match (can be repeated)")
	cmd.Flags().StringVar(&updateParent, "parent", "", "Set parent issue ID")
	cmd.Flags().BoolVar(&updateRemoveParent, "remove-parent", false, "Remove parent")
	cmd.Flags().StringArrayVar(&updateBlocking, "blocking", nil, "ID of issue this blocks (can be repeated)")
	cmd.Flags().StringArrayVar(&updateRemoveBlocking, "remove-blocking", nil, "ID of issue to unblock (can be repeated)")
	cmd.Flags().StringArrayVar(&updateBlockedBy, "blocked-by", nil, "ID of issue that blocks this one (can be repeated)")
	cmd.Flags().StringArrayVar(&updateRemoveBlockedBy, "remove-blocked-by", nil, "ID of blocker issue to remove (can be repeated)")
	cmd.Flags().StringArrayVar(&updateTag, "tag", nil, "Add tag (can be repeated)")
	cmd.Flags().StringArrayVar(&updateRemoveTag, "remove-tag", nil, "Remove tag (can be repeated)")
	cmd.Flags().StringVar(&updateIfMatch, "if-match", "", "Only update if etag matches (optimistic locking)")
	cmd.Flags().BoolVar(&todoUpdateJSON, "json", false, "Output as JSON")

	cmd.MarkFlagsMutuallyExclusive("parent", "remove-parent")
	cmd.MarkFlagsMutuallyExclusive("replace-body", "replace-body-file", "body-replace-old")
	cmd.MarkFlagsMutuallyExclusive("replace-body", "replace-body-file", "append-body")
	cmd.MarkFlagsRequiredTogether("body-replace-old", "body-replace-new")
}

func init() {
	registerUpdateFlags(todoUpdateCmd)
	todoCmd.AddCommand(todoUpdateCmd)
}
