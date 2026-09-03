package drive

import (
	"github.com/spf13/cobra"

	drivecmd "github.com/open-cli-collective/google-cli/internal/cmd/drive"
)

// NewCommand extends the shared Drive command with file mutations.
func NewCommand() *cobra.Command {
	cmd := drivecmd.NewCommand()
	cmd.AddCommand(newUploadCommand(), newMkdirCommand(), newRenameCommand(), newMoveCommand(), newTrashCommand(), newRestoreCommand(), newDeleteCommand())
	return cmd
}
