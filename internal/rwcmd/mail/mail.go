package mail

import (
	"github.com/spf13/cobra"

	mailcmd "github.com/open-cli-collective/google-cli/internal/cmd/mail"
)

// NewCommand returns grw's `mail` command: the shared non-destructive leaves
// from mailcmd plus grw's read-write leaves (send, delete, folder, filter). The
// shared leaves resolve their client through mailcmd's own ClientFactory, which
// uses the registered (grw) identity's credentials — so `grw mail search` and
// `grw mail archive` work against the same account as the write commands.
func NewCommand() *cobra.Command {
	cmd := mailcmd.NewCommand()
	cmd.AddCommand(newSendCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newRestoreCommand())
	cmd.AddCommand(newFolderCommand())
	cmd.AddCommand(newFilterCommand())
	return cmd
}
