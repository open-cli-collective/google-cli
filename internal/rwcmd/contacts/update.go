package contacts

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	var flags contactFlags
	cmd := &cobra.Command{
		Use: "update <resource-name>", Short: "Update a contact", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldNames := []string{"given-name", "family-name", "middle-name", "prefix", "suffix", "email", "phone", "org", "title", "department", "address", "url", "biography", "birthday"}
			anyChanged := false
			for _, name := range fieldNames {
				anyChanged = anyChanged || cmd.Flags().Changed(name)
			}
			if !anyChanged {
				return fmt.Errorf("set at least one contact field to update")
			}
			contact, err := flags.contact(cmd, true)
			if err != nil {
				return err
			}
			contact.ResourceName = args[0]
			if flags.dryRun {
				fmt.Println("[dry-run] Would update contact:")
				printContact(contact)
				return nil
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}
			updated, err := client.UpdateContact(cmd.Context(), contact)
			if err != nil {
				return fmt.Errorf("updating contact: %w", err)
			}
			printContact(updated)
			return nil
		},
	}
	flags.add(cmd)
	return cmd
}
