// Package drive provides grw's Drive command surface.
package drive

import (
	"context"
	"fmt"

	driveapi "github.com/open-cli-collective/google-cli/internal/api/drive"
	drivecmd "github.com/open-cli-collective/google-cli/internal/cmd/drive"
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

func printFile(file *driveapi.File) {
	fmt.Printf("ID: %s\n", file.ID)
	fmt.Printf("Name: %s\n", file.Name)
	fmt.Printf("Type: %s\n", file.MimeType)
	if len(file.Parents) > 0 {
		fmt.Printf("Parent: %s\n", file.Parents[0])
	}
}
