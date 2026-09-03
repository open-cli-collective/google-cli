// Package drive provides grw's Drive command surface.
package drive

import (
	"context"
	"fmt"

	driveapi "github.com/open-cli-collective/google-cli/internal/api/drive"
	drivecmd "github.com/open-cli-collective/google-cli/internal/cmd/drive"
	mailcmd "github.com/open-cli-collective/google-cli/internal/cmd/mail"
	driverw "github.com/open-cli-collective/google-cli/internal/rw/drive"
)

// WriteClient is the Drive surface used by grw commands.
type WriteClient interface {
	drivecmd.DriveClient
	UploadFile(context.Context, string, string, string) (*driveapi.File, error)
	CreateFolder(context.Context, string, string) (*driveapi.File, error)
	RenameFile(context.Context, string, string) (*driveapi.File, error)
	MoveFile(context.Context, string, string) (*driveapi.File, error)
	TrashFiles(context.Context, []string) error
	UntrashFiles(context.Context, []string) error
	DeleteFilesPermanently(context.Context, []string) error
}

// ClientFactory constructs grw's write Drive client.
var ClientFactory = func(ctx context.Context) (WriteClient, error) { return driverw.NewClient(ctx) }

func newWriteClient(ctx context.Context) (WriteClient, error) { return ClientFactory(ctx) }

// printFile prints a file's identity. Names come from Drive, where a
// collaborator may have set them, so they are sanitized before reaching the
// terminal.
func printFile(file *driveapi.File) {
	fmt.Printf("ID: %s\n", mailcmd.SanitizeOutput(file.ID))
	fmt.Printf("Name: %s\n", mailcmd.SanitizeFilename(file.Name))
	fmt.Printf("Type: %s\n", mailcmd.SanitizeOutput(file.MimeType))
	if len(file.Parents) > 0 {
		fmt.Printf("Parent: %s\n", mailcmd.SanitizeOutput(file.Parents[0]))
	}
}
