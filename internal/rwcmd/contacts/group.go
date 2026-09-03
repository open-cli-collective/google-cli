package contacts

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newGroupCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "group", Short: "Manage contact groups"}
	cmd.AddCommand(newGroupCreateCommand(), newGroupRenameCommand(), newGroupDeleteCommand())
	return cmd
}

func newGroupCreateCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{Use: "create <name>", Short: "Create a contact group", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if dryRun {
			fmt.Printf("[dry-run] Would create contact group: %s\n", args[0])
			return nil
		}
		client, err := newWriteClient(cmd.Context())
		if err != nil {
			return fmt.Errorf("creating Contacts client: %w", err)
		}
		group, err := client.CreateGroup(cmd.Context(), args[0])
		if err != nil {
			return fmt.Errorf("creating contact group: %w", err)
		}
		printGroup(group)
		return nil
	}}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func newGroupRenameCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{Use: "rename <name-or-resource> <new-name>", Short: "Rename a contact group", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		if dryRun {
			fmt.Printf("[dry-run] Would rename contact group %s to %s\n", args[0], args[1])
			return nil
		}
		client, err := newWriteClient(cmd.Context())
		if err != nil {
			return fmt.Errorf("creating Contacts client: %w", err)
		}
		resourceName, err := resolveGroup(cmd, client, args[0])
		if err != nil {
			return err
		}
		group, err := client.RenameGroup(cmd.Context(), resourceName, args[1])
		if err != nil {
			return fmt.Errorf("renaming contact group: %w", err)
		}
		printGroup(group)
		return nil
	}}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func newGroupDeleteCommand() *cobra.Command {
	var yes, dryRun bool
	cmd := &cobra.Command{Use: "rm <name-or-resource>", Aliases: []string{"delete", "remove"}, Short: "Delete a contact group", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if isSystemGroup(args[0]) {
			return fmt.Errorf("refusing to delete system contact group %s", args[0])
		}
		if dryRun {
			fmt.Printf("[dry-run] Would delete contact group: %s\n", args[0])
			return nil
		}
		if err := confirm(cmd, fmt.Sprintf("Delete contact group %s?", args[0]), false, yes); err != nil {
			return err
		}
		client, err := newWriteClient(cmd.Context())
		if err != nil {
			return fmt.Errorf("creating Contacts client: %w", err)
		}
		resourceName, err := resolveGroup(cmd, client, args[0])
		if err != nil {
			return err
		}
		if isSystemGroup(resourceName) {
			return fmt.Errorf("refusing to delete system contact group %s", resourceName)
		}
		if err := client.DeleteGroup(cmd.Context(), resourceName); err != nil {
			return fmt.Errorf("deleting contact group: %w", err)
		}
		fmt.Printf("Deleted contact group %s.\n", resourceName)
		return nil
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func resolveGroup(cmd *cobra.Command, client WriteClient, name string) (string, error) {
	if strings.HasPrefix(name, "contactGroups/") {
		return name, nil
	}
	resourceName, err := client.ResolveGroupName(cmd.Context(), name)
	if err != nil {
		return "", fmt.Errorf("resolving contact group: %w", err)
	}
	return resourceName, nil
}

func isSystemGroup(resourceName string) bool {
	id, ok := strings.CutPrefix(resourceName, "contactGroups/")
	if !ok {
		return false
	}
	_, err := strconv.ParseUint(id, 10, 64)
	return err != nil
}
