package mail

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-cli-common/bulk"
)

func newDeleteCommand() *cobra.Command {
	var dryRun, stdin, permanent, yes bool
	var query string

	cmd := &cobra.Command{
		Use:   "delete [message-ids...]",
		Short: "Move messages to Trash (or permanently delete with --permanent)",
		Long: `Delete messages.

By default this moves messages to Trash, where Gmail keeps them for ~30 days and
you can restore them. With --permanent the messages are erased immediately and
cannot be recovered; that path requires a typed confirmation (or --yes) and the
broader mail.google.com scope granted at 'grw init'.

Message IDs come from positional args, --stdin, or --query (exactly one source).`,
		Example: `  grw mail delete 18c... 18d...            # move two messages to Trash
  grw mail delete --query "older_than:1y from:news@example.com"
  grw mail delete --query "in:trash" --permanent   # empty matching trash, irreversibly`,
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

			result := &bulk.Result{IDs: ids, Count: len(ids), DryRun: dryRun}
			if permanent {
				result.Action = "permanently deleted"
				if dryRun {
					result.Action = "permanently delete"
					return result.Print()
				}
				if err := confirmPermanent(cmd, len(ids), stdin, yes); err != nil {
					return err
				}
				if err := client.DeleteMessagesPermanently(ctx, ids); err != nil {
					return fmt.Errorf("permanently deleting messages: %w", err)
				}
				return result.Print()
			}

			result.Action = "trashed"
			if dryRun {
				result.Action = "trash"
				return result.Print()
			}
			if err := client.TrashMessages(ctx, ids); err != nil {
				return fmt.Errorf("trashing messages: %w", err)
			}
			return result.Print()
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read message IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve message IDs")
	cmd.Flags().BoolVar(&permanent, "permanent", false, "Permanently erase instead of moving to Trash (irreversible; needs the mail.google.com scope)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the typed confirmation required by --permanent")
	return cmd
}

// confirmPermanent enforces the guard on irreversible deletion. When IDs were
// read from stdin, stdin is already consumed, so a typed prompt is impossible
// and --yes is required. Otherwise the user must type "delete" to proceed.
func confirmPermanent(cmd *cobra.Command, n int, stdin, yes bool) error {
	if yes {
		return nil
	}
	if stdin {
		return errors.New("--permanent with --stdin requires --yes (stdin is consumed by the message IDs, so it can't be used for confirmation)")
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(),
		"This will PERMANENTLY delete %d message(s). This cannot be undone.\nType 'delete' to confirm: ", n)
	reader := bufio.NewReader(cmd.InOrStdin())
	line, _ := reader.ReadString('\n')
	if strings.TrimSpace(line) != "delete" {
		return errors.New("aborted: confirmation did not match")
	}
	return nil
}
