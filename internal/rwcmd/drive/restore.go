package drive

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRestoreCommand() *cobra.Command {
	var dryRun, stdin bool
	var query string
	cmd := &cobra.Command{
		Use:   "restore [file-ids...]",
		Short: "Restore files from Trash",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, ids, err := resolveFileIDs(cmd, args, stdin, query, dryRun)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Println("No files matched.")
				return nil
			}
			if dryRun {
				return printBulkResult("restore", ids, true)
			}
			if err := client.UntrashFiles(cmd.Context(), ids); err != nil {
				return fmt.Errorf("restoring files: %w", err)
			}
			return printBulkResult("restored", ids, false)
		},
	}
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read file IDs from stdin")
	cmd.Flags().StringVar(&query, "query", "", "Search query to resolve file IDs")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}
