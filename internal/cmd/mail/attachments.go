package mail

import (
	"github.com/spf13/cobra"
)

func newAttachmentsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachments",
		Short: "Manage message attachments",
		Long: `List and download attachments from Gmail messages.

This command group provides read-only access to message attachments.
Use 'list' to view attachment metadata and 'download' to save files locally.

Examples:
  mail attachments list 18abc123def456
  mail attachments download 18abc123def456 --all
  mail attachments download 18abc123def456 --filename report.pdf`,
	}

	cmd.AddCommand(newListAttachmentsCommand())
	cmd.AddCommand(newDownloadAttachmentsCommand())

	return cmd
}
