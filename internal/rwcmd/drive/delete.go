package drive

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	var dryRun, stdin, permanent, yes bool
	var query string
	cmd := &cobra.Command{
		Use:   "delete [file-ids...]",
		Short: "Move files to Trash (or permanently delete with --permanent)",
		Long: `Delete files.

By default this moves files to Trash, where they can be restored. With
--permanent the files are erased immediately and cannot be recovered; that
path requires a typed confirmation (or --yes).

File IDs come from positional args, --stdin, or --query (exactly one source).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, ids, err := resolveFileIDs(cmd, args, stdin, query, dryRun)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Println("No files matched.")
				return nil
			}
			if permanent {
				if dryRun {
					return printBulkResult("permanently delete", ids, true)
				}
				if err := confirmPermanent(cmd, len(ids), stdin, yes); err != nil {
					return err
				}
				if err := client.DeleteFilesPermanently(cmd.Context(), ids); err != nil {
					return fmt.Errorf("permanently deleting files: %w", err)
				}
				return printBulkResult("permanently deleted", ids, false)
			}
			if dryRun {
				return printBulkResult("trash", ids, true)
			}
			if err := client.TrashFiles(cmd.Context(), ids); err != nil {
				return fmt.Errorf("trashing files: %w", err)
			}
			return printBulkResult("trashed", ids, false)
		},
	}
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read file IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve file IDs")
	cmd.Flags().BoolVar(&permanent, "permanent", false, "Permanently erase instead of moving to Trash (irreversible)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the typed confirmation required by --permanent")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func confirmPermanent(cmd *cobra.Command, n int, stdin, yes bool) error {
	if yes {
		return nil
	}
	if stdin {
		return errors.New("--permanent with --stdin requires --yes (stdin is consumed by the file IDs, so it can't be used for confirmation)")
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(),
		"This will PERMANENTLY delete %d file(s). This cannot be undone.\nType 'delete' to confirm: ", n)
	reader := bufio.NewReader(cmd.InOrStdin())
	line, _ := reader.ReadString('\n')
	if strings.TrimSpace(line) != "delete" {
		return errors.New("aborted: confirmation did not match")
	}
	return nil
}
