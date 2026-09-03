package mail

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-cli/common/bulk"
)

func newRestoreCommand() *cobra.Command {
	var dryRun, stdin bool
	var query string

	cmd := &cobra.Command{
		Use:   "restore [message-ids...]",
		Short: "Restore messages from Trash back to the inbox",
		Long: `Move messages out of Trash (undo a plain delete).

This only works while the messages are still in Trash; once Gmail purges Trash
(~30 days) or you used 'delete --permanent', they are gone for good.

Message IDs come from positional args, --stdin, or --query (exactly one source).`,
		Example: `  grw mail restore 18c... 18d...
  grw mail restore --query "in:trash from:friend@example.com"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newWriteClient(ctx)
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}

			ids, err := bulk.ResolveIDs(bulk.Config{Args: args, Stdin: stdin, Query: query},
				func(q string) ([]string, error) { return client.SearchMessageIDs(ctx, q, 0) })
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Println("No messages matched.")
				return nil
			}

			result := &bulk.Result{IDs: ids, Count: len(ids), DryRun: dryRun, Action: "restored"}
			if dryRun {
				result.Action = "restore"
				return result.Print()
			}
			if err := client.UntrashMessages(ctx, ids); err != nil {
				return fmt.Errorf("restoring messages: %w", err)
			}
			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read message IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve message IDs (e.g. \"in:trash ...\")")
	return cmd
}
