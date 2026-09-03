package drive

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	driveapi "github.com/open-cli-collective/google-cli/internal/api/drive"
	"github.com/open-cli-collective/google-cli/internal/testutil"
)

func withFactory(factory func(context.Context) (WriteClient, error), f func()) {
	testutil.WithFactory(&ClientFactory, factory, f)
}

func runCommand(cmd *cobra.Command, args ...string) (string, error) {
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return output.String(), err
}

func localFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCommandsSuccess(t *testing.T) {
	file := &driveapi.File{ID: "file-1", Name: "result", MimeType: "text/plain", Parents: []string{"parent-1"}}
	mock := &mockWriteClient{
		UploadFileFunc: func(_ context.Context, _, parent, name string) (*driveapi.File, error) {
			testutil.Equal(t, parent, "parent-1")
			testutil.Equal(t, name, "remote.txt")
			return file, nil
		},
		CreateFolderFunc: func(_ context.Context, name, parent string) (*driveapi.File, error) {
			testutil.Equal(t, name, "Folder")
			testutil.Equal(t, parent, "parent-1")
			return file, nil
		},
		RenameFileFunc: func(_ context.Context, id, name string) (*driveapi.File, error) {
			testutil.Equal(t, id, "file-1")
			testutil.Equal(t, name, "Renamed")
			return file, nil
		},
		MoveFileFunc: func(_ context.Context, id, parent string) (*driveapi.File, error) {
			testutil.Equal(t, id, "file-1")
			testutil.Equal(t, parent, "folder-1")
			return file, nil
		},
		TrashFilesFunc: func(_ context.Context, ids []string) error {
			testutil.Equal(t, strings.Join(ids, ","), "one,two")
			return nil
		},
		UntrashFilesFunc: func(_ context.Context, ids []string) error {
			testutil.Equal(t, strings.Join(ids, ","), "one,two")
			return nil
		},
	}
	path := localFile(t)
	tests := []struct {
		name string
		cmd  *cobra.Command
		args []string
		want string
	}{
		{"upload", newUploadCommand(), []string{path, "--parent", "parent-1", "--name", "remote.txt"}, "ID: file-1"},
		{"mkdir", newMkdirCommand(), []string{"Folder", "--parent", "parent-1"}, "ID: file-1"},
		{"rename", newRenameCommand(), []string{"file-1", "Renamed"}, "ID: file-1"},
		{"move", newMoveCommand(), []string{"file-1", "folder-1"}, "ID: file-1"},
		{"trash", newTrashCommand(), []string{"one", "two"}, "Trashed 2 file(s)."},
		{"restore", newRestoreCommand(), []string{"one", "two"}, "Restored 2 file(s)."},
	}
	withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				output := testutil.CaptureStdout(t, func() {
					_, err := runCommand(test.cmd, test.args...)
					testutil.NoError(t, err)
				})
				testutil.Contains(t, output, test.want)
			})
		}
	})
}

func TestCommandErrors(t *testing.T) {
	boom := errors.New("API error")
	file := localFile(t)
	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
		mock *mockWriteClient
		want string
	}{
		{"upload", newUploadCommand, []string{file}, &mockWriteClient{UploadFileFunc: func(context.Context, string, string, string) (*driveapi.File, error) { return nil, boom }}, "uploading file"},
		{"mkdir", newMkdirCommand, []string{"Folder"}, &mockWriteClient{CreateFolderFunc: func(context.Context, string, string) (*driveapi.File, error) { return nil, boom }}, "creating folder"},
		{"rename", newRenameCommand, []string{"one", "name"}, &mockWriteClient{RenameFileFunc: func(context.Context, string, string) (*driveapi.File, error) { return nil, boom }}, "renaming file"},
		{"move", newMoveCommand, []string{"one", "folder"}, &mockWriteClient{MoveFileFunc: func(context.Context, string, string) (*driveapi.File, error) { return nil, boom }}, "moving file"},
		{"trash", newTrashCommand, []string{"one"}, &mockWriteClient{TrashFilesFunc: func(context.Context, []string) error { return boom }}, "trashing files"},
		{"restore", newRestoreCommand, []string{"one"}, &mockWriteClient{UntrashFilesFunc: func(context.Context, []string) error { return boom }}, "restoring files"},
		{"delete", newDeleteCommand, []string{"one"}, &mockWriteClient{TrashFilesFunc: func(context.Context, []string) error { return boom }}, "trashing files"},
	}
	for _, test := range tests {
		t.Run(test.name+" API error", func(t *testing.T) {
			withFactory(func(context.Context) (WriteClient, error) { return test.mock, nil }, func() {
				_, err := runCommand(test.cmd(), test.args...)
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("error = %v, want %q", err, test.want)
				}
			})
		})
		t.Run(test.name+" client error", func(t *testing.T) {
			withFactory(func(context.Context) (WriteClient, error) { return nil, errors.New("no client") }, func() {
				_, err := runCommand(test.cmd(), test.args...)
				if err == nil || !strings.Contains(err.Error(), "creating Drive client") {
					t.Fatalf("error = %v", err)
				}
			})
		})
	}
}

func TestDryRunsDoNotConstructClient(t *testing.T) {
	path := localFile(t)
	withFactory(func(context.Context) (WriteClient, error) { t.Fatal("dry-run constructed client"); return nil, nil }, func() {
		tests := []struct {
			name string
			cmd  *cobra.Command
			args []string
		}{
			{"upload", newUploadCommand(), []string{path, "--dry-run"}},
			{"mkdir", newMkdirCommand(), []string{"Folder", "--dry-run"}},
			{"rename", newRenameCommand(), []string{"one", "name", "--dry-run"}},
			{"move", newMoveCommand(), []string{"one", "folder", "--dry-run"}},
			{"trash", newTrashCommand(), []string{"one", "--dry-run"}},
			{"restore", newRestoreCommand(), []string{"one", "--dry-run"}},
			{"delete", newDeleteCommand(), []string{"one", "--dry-run"}},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				output := testutil.CaptureStdout(t, func() { _, err := runCommand(test.cmd, test.args...); testutil.NoError(t, err) })
				testutil.Contains(t, output, "[dry-run]")
			})
		}
	})
}

func TestTrashResolvesQuery(t *testing.T) {
	var trashed []string
	mock := &mockWriteClient{
		SearchFileIDsFunc: func(_ context.Context, query string, maxResults int64) ([]string, error) {
			testutil.Equal(t, query, "name contains 'old'")
			testutil.Equal(t, maxResults, int64(0))
			return []string{"one", "two"}, nil
		},
		TrashFilesFunc: func(_ context.Context, ids []string) error { trashed = ids; return nil },
	}
	withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
		testutil.CaptureStdout(t, func() {
			_, err := runCommand(newTrashCommand(), "--query", "name contains 'old'")
			testutil.NoError(t, err)
		})
	})
	testutil.Equal(t, strings.Join(trashed, ","), "one,two")
}

func TestDelete(t *testing.T) {
	t.Run("permanent requires confirmation", func(t *testing.T) {
		mock := &mockWriteClient{DeleteFilesPermanentlyFunc: func(context.Context, []string) error { t.Fatal("delete called"); return nil }}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			cmd := newDeleteCommand()
			cmd.SetIn(strings.NewReader("no\n"))
			_, err := runCommand(cmd, "one", "--permanent")
			if err == nil || !strings.Contains(err.Error(), "aborted") {
				t.Fatalf("error = %v", err)
			}
		})
	})
	t.Run("yes skips confirmation", func(t *testing.T) {
		var deleted []string
		mock := &mockWriteClient{DeleteFilesPermanentlyFunc: func(_ context.Context, ids []string) error { deleted = ids; return nil }}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			output := testutil.CaptureStdout(t, func() {
				_, err := runCommand(newDeleteCommand(), "one", "--permanent", "--yes")
				testutil.NoError(t, err)
			})
			testutil.Equal(t, strings.Join(deleted, ","), "one")
			testutil.Contains(t, output, "Permanently deleted 1 file(s).")
		})
	})
	t.Run("stdin and query conflict", func(t *testing.T) {
		withFactory(func(context.Context) (WriteClient, error) { t.Fatal("client constructed"); return nil, nil }, func() {
			_, err := runCommand(newDeleteCommand(), "--stdin", "--query", "trashed = true")
			if err == nil || !strings.Contains(err.Error(), "only one input source") {
				t.Fatalf("error = %v", err)
			}
		})
	})
}

func TestNewCommandComposition(t *testing.T) {
	names := map[string]bool{}
	for _, command := range NewCommand().Commands() {
		names[command.Name()] = true
	}
	for _, want := range []string{"list", "search", "get", "download", "tree", "drives", "star", "unstar", "upload", "mkdir", "rename", "move", "trash", "restore", "delete"} {
		if !names[want] {
			t.Errorf("missing %s command", want)
		}
	}
}
