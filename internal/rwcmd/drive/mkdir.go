package drive

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMkdirCommand() *cobra.Command {
	var parent string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "mkdir <name>",
		Short: "Create a folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun {
				fmt.Printf("[dry-run] Would create folder %q in parent %s.\n", args[0], displayParent(parent))
				return nil
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Drive client: %w", err)
			}
			file, err := client.CreateFolder(cmd.Context(), args[0], parent)
			if err != nil {
				return fmt.Errorf("creating folder: %w", err)
			}
			printFile(file)
			return nil
		},
	}
	cmd.Flags().StringVar(&parent, "parent", "", "Parent folder ID")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func displayParent(parent string) string {
	if parent == "" {
		return "root"
	}
	return parent
}
