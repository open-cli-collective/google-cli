package mail

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newFolderCommand groups the label-lifecycle operations. In Gmail a "folder"
// is a user label; nesting ("subfolders") is expressed by "/" in the name,
// e.g. "Receipts/2026".
func newFolderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folder",
		Short: "Create, rename, and delete labels (folders/subfolders)",
		Long: `Manage Gmail labels as folders.

A folder is a Gmail label. Create a subfolder by nesting its name with "/",
e.g. "Receipts/2026" nests "2026" under "Receipts".`,
	}
	cmd.AddCommand(newFolderCreateCommand(), newFolderRenameCommand(), newFolderRemoveCommand())
	return cmd
}

func newFolderCreateCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a label (use \"Parent/Child\" for a subfolder)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun {
				fmt.Printf("[dry-run] Would create folder %q.\n", args[0])
				return nil
			}
			ctx := cmd.Context()
			client, err := newWriteClient(ctx)
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			label, err := client.CreateLabel(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Created folder %q (id %s).\n", label.Name, label.Id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func newFolderRenameCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "rename <current-name> <new-name>",
		Short: "Rename or re-nest a label",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun {
				fmt.Printf("[dry-run] Would rename folder %q to %q.\n", args[0], args[1])
				return nil
			}
			ctx := cmd.Context()
			client, err := newWriteClient(ctx)
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			id, err := client.GetLabelID(ctx, args[0])
			if err != nil {
				return fmt.Errorf("resolving folder %q: %w", args[0], err)
			}
			label, err := client.RenameLabel(ctx, id, args[1])
			if err != nil {
				return err
			}
			fmt.Printf("Renamed folder to %q.\n", label.Name)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func newFolderRemoveCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:     "rm <name>",
		Aliases: []string{"delete", "remove"},
		Short:   "Delete a label (messages keep their content; they just lose the label)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun {
				fmt.Printf("[dry-run] Would delete folder %q.\n", args[0])
				return nil
			}
			ctx := cmd.Context()
			client, err := newWriteClient(ctx)
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			id, err := client.GetLabelID(ctx, args[0])
			if err != nil {
				return fmt.Errorf("resolving folder %q: %w", args[0], err)
			}
			if err := client.DeleteLabel(ctx, id); err != nil {
				return err
			}
			fmt.Printf("Deleted folder %q.\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}
