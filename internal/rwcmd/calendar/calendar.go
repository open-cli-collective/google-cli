// Package calendar provides grw's Calendar command surface.
package calendar

import (
	"github.com/spf13/cobra"

	calendarcmd "github.com/open-cli-collective/google-cli/internal/cmd/calendar"
)

// NewCommand extends the shared Calendar command with event mutations.
func NewCommand() *cobra.Command {
	cmd := calendarcmd.NewCommand()
	cmd.AddCommand(newCreateCommand(), newUpdateCommand(), newDeleteCommand())
	return cmd
}
