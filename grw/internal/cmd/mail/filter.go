package mail

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	gmailv1 "google.golang.org/api/gmail/v1"
)

func newFilterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "filter",
		Short: "List, create, and delete Gmail filters",
	}
	cmd.AddCommand(newFilterListCommand(), newFilterCreateCommand(), newFilterRemoveCommand())
	return cmd
}

func newFilterListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Gmail filters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			client, err := newWriteClient(ctx)
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			filters, err := client.ListFilters(ctx)
			if err != nil {
				return err
			}
			if len(filters) == 0 {
				fmt.Println("No filters.")
				return nil
			}
			// Resolve label IDs to names for readable action output.
			if err := client.FetchLabels(ctx); err != nil {
				return err
			}
			names := map[string]string{}
			for _, l := range client.GetLabels() {
				names[l.Id] = l.Name
			}
			for _, f := range filters {
				fmt.Printf("%s\n", f.Id)
				fmt.Printf("  when: %s\n", criteriaSummary(f.Criteria))
				fmt.Printf("  then: %s\n", actionSummary(f.Action, names))
			}
			return nil
		},
	}
}

func newFilterCreateCommand() *cobra.Command {
	var from, to, subject, query string
	var hasAttachment bool
	var addLabel string
	var archive, markRead, star, trash bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a filter: match on criteria, then apply an action",
		Long: `Create a Gmail filter.

Specify at least one criterion (--from/--to/--subject/--query/--has-attachment)
and at least one action (--add-label/--archive/--mark-read/--star/--trash).
Actions combine, e.g. --archive --mark-read --add-label Newsletters.`,
		Example: `  grw mail filter create --from news@example.com --archive --mark-read
  grw mail filter create --subject "[CI]" --add-label CI --archive`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			client, err := newWriteClient(ctx)
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			crit := &gmailv1.FilterCriteria{}
			hasCriterion := false
			if from != "" {
				crit.From = from
				hasCriterion = true
			}
			if to != "" {
				crit.To = to
				hasCriterion = true
			}
			if subject != "" {
				crit.Subject = subject
				hasCriterion = true
			}
			if query != "" {
				crit.Query = query
				hasCriterion = true
			}
			if hasAttachment {
				crit.HasAttachment = true
				hasCriterion = true
			}
			if !hasCriterion {
				return fmt.Errorf("at least one criterion is required (--from/--to/--subject/--query/--has-attachment)")
			}

			action := &gmailv1.FilterAction{}
			if addLabel != "" {
				id, err := client.GetLabelID(ctx, addLabel)
				if err != nil {
					return fmt.Errorf("resolving --add-label %q (create it first with 'grw mail folder create'): %w", addLabel, err)
				}
				action.AddLabelIds = append(action.AddLabelIds, id)
			}
			if star {
				action.AddLabelIds = append(action.AddLabelIds, "STARRED")
			}
			if trash {
				action.AddLabelIds = append(action.AddLabelIds, "TRASH")
			}
			if archive {
				action.RemoveLabelIds = append(action.RemoveLabelIds, "INBOX")
			}
			if markRead {
				action.RemoveLabelIds = append(action.RemoveLabelIds, "UNREAD")
			}
			if len(action.AddLabelIds) == 0 && len(action.RemoveLabelIds) == 0 {
				return fmt.Errorf("at least one action is required (--add-label/--archive/--mark-read/--star/--trash)")
			}

			created, err := client.CreateFilter(ctx, &gmailv1.Filter{Criteria: crit, Action: action})
			if err != nil {
				return err
			}
			fmt.Printf("Created filter %s.\n", created.Id)
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Match messages from this sender")
	cmd.Flags().StringVar(&to, "to", "", "Match messages to this recipient")
	cmd.Flags().StringVar(&subject, "subject", "", "Match messages with this subject")
	cmd.Flags().StringVar(&query, "query", "", "Match messages with this Gmail search query")
	cmd.Flags().BoolVar(&hasAttachment, "has-attachment", false, "Match messages that have an attachment")
	cmd.Flags().StringVar(&addLabel, "add-label", "", "Apply this label (must already exist)")
	cmd.Flags().BoolVar(&archive, "archive", false, "Skip the inbox (archive)")
	cmd.Flags().BoolVar(&markRead, "mark-read", false, "Mark as read")
	cmd.Flags().BoolVar(&star, "star", false, "Star the message")
	cmd.Flags().BoolVar(&trash, "trash", false, "Move to Trash")
	return cmd
}

func newFilterRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "rm <filter-id>",
		Aliases: []string{"delete", "remove"},
		Short:   "Delete a filter by ID (from 'filter list')",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newWriteClient(ctx)
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			if err := client.DeleteFilter(ctx, args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted filter %s.\n", args[0])
			return nil
		},
	}
}

func criteriaSummary(c *gmailv1.FilterCriteria) string {
	if c == nil {
		return "(any)"
	}
	var parts []string
	if c.From != "" {
		parts = append(parts, "from:"+c.From)
	}
	if c.To != "" {
		parts = append(parts, "to:"+c.To)
	}
	if c.Subject != "" {
		parts = append(parts, "subject:"+c.Subject)
	}
	if c.Query != "" {
		parts = append(parts, c.Query)
	}
	if c.HasAttachment {
		parts = append(parts, "has:attachment")
	}
	if len(parts) == 0 {
		return "(any)"
	}
	return strings.Join(parts, " ")
}

func actionSummary(a *gmailv1.FilterAction, labelNames map[string]string) string {
	if a == nil {
		return "(none)"
	}
	var parts []string
	for _, id := range a.AddLabelIds {
		switch id {
		case "STARRED":
			parts = append(parts, "star")
		case "TRASH":
			parts = append(parts, "trash")
		default:
			if n := labelNames[id]; n != "" {
				parts = append(parts, "label:"+n)
			} else {
				parts = append(parts, "label:"+id)
			}
		}
	}
	for _, id := range a.RemoveLabelIds {
		switch id {
		case "INBOX":
			parts = append(parts, "archive")
		case "UNREAD":
			parts = append(parts, "mark-read")
		default:
			if n := labelNames[id]; n != "" {
				parts = append(parts, "remove-label:"+n)
			} else {
				parts = append(parts, "remove-label:"+id)
			}
		}
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, ", ")
}
