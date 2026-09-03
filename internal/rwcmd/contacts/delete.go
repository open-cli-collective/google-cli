package contacts

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
	cmd := &cobra.Command{
		Use: "delete <resource-name>...", Aliases: []string{"rm", "remove"}, Short: "Delete contacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, err := bulk.ResolveIDs(bulk.Config{Args: args, Stdin: stdin}, nil)
			if err != nil {
				return err
			}
			if dryRun {
				fmt.Println("[dry-run] Would delete contacts:")
				for _, id := range ids {
					fmt.Println(id)
				}
				return nil
			}
			if err := confirm(cmd, fmt.Sprintf("Delete %d contact(s)?", len(ids)), stdin, yes); err != nil {
				return err
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}
			for _, id := range ids {
				if err := client.DeleteContact(cmd.Context(), id); err != nil {
					return fmt.Errorf("deleting contact %s: %w", id, err)
				}
			}
			return (&bulk.Result{Action: "deleted", IDs: ids, Count: len(ids), ItemNoun: "contact"}).Print()
		},
	}
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read contact resource names from stdin")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func confirm(cmd *cobra.Command, prompt string, stdin, yes bool) error {
	if yes {
		return nil
	}
	if stdin {
		return errors.New("--stdin requires --yes because stdin is consumed by resource names")
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s Type 'y' or 'yes' to confirm: ", prompt)
	answer, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		return errors.New("aborted: confirmation did not match")
	}
	return nil
}
