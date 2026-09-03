package mail

import (
	"fmt"

	"github.com/spf13/cobra"

	gmailapi "github.com/open-cli-collective/google-cli/internal/api/gmail"
	mailcmd "github.com/open-cli-collective/google-cli/internal/cmd/mail"
)

func newSendCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "send <draft-id>",
		Short: "Send an existing Gmail draft",
		Long: `Preview and send an existing Gmail draft by its ID.

The draft is fetched and its headers are displayed before sending. Dry-run also
constructs a Gmail client because fetching the draft is required for preview.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			draft, err := client.GetDraft(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("getting draft: %w", err)
			}
			printDraftSummary(draft)
			if dryRun {
				return nil
			}
			if draft.To == "" && draft.Cc == "" && draft.Bcc == "" {
				return fmt.Errorf("draft has no recipients")
			}
			sent, err := client.SendDraft(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("sending draft: %w", err)
			}
			fmt.Printf("Sent message %s in thread %s\n", sent.ID, sent.ThreadID)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without sending the draft")
	return cmd
}

func printDraftSummary(draft *gmailapi.DraftSummary) {
	fmt.Printf("From: %s\n", mailcmd.SanitizeOutput(draft.From))
	fmt.Printf("To: %s\n", mailcmd.SanitizeOutput(draft.To))
	fmt.Printf("Cc: %s\n", mailcmd.SanitizeOutput(draft.Cc))
	fmt.Printf("Bcc: %s\n", mailcmd.SanitizeOutput(draft.Bcc))
	fmt.Printf("Subject: %s\n", mailcmd.SanitizeOutput(draft.Subject))
	fmt.Printf("Attachments: %d\n", draft.AttachmentCount)
}
