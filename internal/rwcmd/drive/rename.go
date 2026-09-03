package drive

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRenameCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "rename <file-id> <new-name>",
		Short: "Rename a file or folder",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun {
				fmt.Printf("[dry-run] Would rename file %s to %q.\n", args[0], args[1])
				return nil
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Drive client: %w", err)
			}
			file, err := client.RenameFile(cmd.Context(), args[0], args[1])
			if err != nil {
				return fmt.Errorf("renaming file: %w", err)
			}
			printFile(file)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}
