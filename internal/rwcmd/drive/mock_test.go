package drive

import (
	"context"

	driveapi "github.com/open-cli-collective/google-cli/internal/api/drive"
	drivecmd "github.com/open-cli-collective/google-cli/internal/cmd/drive"
)

type mockWriteClient struct {
	drivecmd.DriveClient
	SearchFileIDsFunc          func(context.Context, string, int64) ([]string, error)
	UploadFileFunc             func(context.Context, string, string, string) (*driveapi.File, error)
	CreateFolderFunc           func(context.Context, string, string) (*driveapi.File, error)
	RenameFileFunc             func(context.Context, string, string) (*driveapi.File, error)
	MoveFileFunc               func(context.Context, string, string) (*driveapi.File, error)
	TrashFilesFunc             func(context.Context, []string) error
	UntrashFilesFunc           func(context.Context, []string) error
	DeleteFilesPermanentlyFunc func(context.Context, []string) error
}

var _ WriteClient = (*mockWriteClient)(nil)

func (m *mockWriteClient) SearchFileIDs(ctx context.Context, query string, maxResults int64) ([]string, error) {
	return m.SearchFileIDsFunc(ctx, query, maxResults)
}
func (m *mockWriteClient) UploadFile(ctx context.Context, path, parent, name string) (*driveapi.File, error) {
	return m.UploadFileFunc(ctx, path, parent, name)
}
func (m *mockWriteClient) CreateFolder(ctx context.Context, name, parent string) (*driveapi.File, error) {
	return m.CreateFolderFunc(ctx, name, parent)
}
func (m *mockWriteClient) RenameFile(ctx context.Context, id, name string) (*driveapi.File, error) {
	return m.RenameFileFunc(ctx, id, name)
}
func (m *mockWriteClient) MoveFile(ctx context.Context, id, parent string) (*driveapi.File, error) {
	return m.MoveFileFunc(ctx, id, parent)
}
func (m *mockWriteClient) TrashFiles(ctx context.Context, ids []string) error {
	return m.TrashFilesFunc(ctx, ids)
}
func (m *mockWriteClient) UntrashFiles(ctx context.Context, ids []string) error {
	return m.UntrashFilesFunc(ctx, ids)
}
func (m *mockWriteClient) DeleteFilesPermanently(ctx context.Context, ids []string) error {
	return m.DeleteFilesPermanentlyFunc(ctx, ids)
}
