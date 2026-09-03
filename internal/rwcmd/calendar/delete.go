package calendar

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-cli/internal/bulk"
)

func newDeleteCommand() *cobra.Command {
	var stdin, yes, dryRun bool
	var calendarID string
	cmd := &cobra.Command{
		Use:     "delete <event-id>...",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete calendar events",
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, err := bulk.ResolveIDs(bulk.Config{Args: args, Stdin: stdin}, nil)
			if err != nil {
				return err
			}
			if dryRun {
				fmt.Println("[dry-run] Would delete event IDs:")
				for _, id := range ids {
					fmt.Println(id)
				}
				return nil
			}
			if err := confirmDelete(cmd, len(ids), stdin, yes); err != nil {
				return err
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Calendar client: %w", err)
			}
			for _, id := range ids {
				if err := client.DeleteEvent(cmd.Context(), calendarID, id); err != nil {
					return fmt.Errorf("deleting event %s: %w", id, err)
				}
			}
			return (&bulk.Result{Action: "deleted", IDs: ids, Count: len(ids), ItemNoun: "event"}).Print()
		},
	}
	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "primary", "Calendar ID")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read event IDs from stdin")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func confirmDelete(cmd *cobra.Command, count int, stdin, yes bool) error {
	if yes {
		return nil
	}
	if stdin {
		return errors.New("--stdin requires --yes because stdin is consumed by event IDs")
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Delete %d event(s)? Type 'y' or 'yes' to confirm: ", count)
	line, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != "yes" {
		return errors.New("aborted: confirmation did not match")
	}
	return nil
}
