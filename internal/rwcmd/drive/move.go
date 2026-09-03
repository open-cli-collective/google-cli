package drive

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMoveCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "move <file-id> <folder-id>",
		Short: "Move a file or folder",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun {
				fmt.Printf("[dry-run] Would move file %s to folder %s.\n", args[0], args[1])
				return nil
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Drive client: %w", err)
			}
			file, err := client.MoveFile(cmd.Context(), args[0], args[1])
			if err != nil {
				return fmt.Errorf("moving file: %w", err)
			}
			printFile(file)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}
