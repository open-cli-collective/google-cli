// Package contacts provides grw's Contacts command surface.
package contacts

import (
	"github.com/spf13/cobra"

	contactscmd "github.com/open-cli-collective/google-cli/internal/cmd/contacts"
)

// NewCommand extends the shared Contacts command with contact mutations.
func NewCommand() *cobra.Command {
	cmd := contactscmd.NewCommand()
	cmd.AddCommand(newCreateCommand(), newUpdateCommand(), newDeleteCommand(), newGroupCommand())
	return cmd
}
