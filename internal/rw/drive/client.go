// Package drive extends the read Drive client with file mutations.
package drive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	drivev3 "google.golang.org/api/drive/v3"

	driveapi "github.com/open-cli-collective/google-cli/internal/api/drive"
)

// Client is grw's Drive client.
type Client struct {
	*driveapi.Client
	service *drivev3.Service
}

// NewClient builds a write-capable Drive client.
func NewClient(ctx context.Context) (*Client, error) {
	base, err := driveapi.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{Client: base, service: base.Service()}, nil
}

// FileMIMEType detects a local file's MIME type.
func FileMIMEType(path string) (string, error) {
	if typ := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); typ != "" {
		return typ, nil
	}
	file, err := os.Open(path) //nolint:gosec // path is supplied by the user
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("reading file: %w", err)
	}
	return http.DetectContentType(buf[:n]), nil
}

// UploadFile uploads a local file.
func (c *Client) UploadFile(ctx context.Context, localPath, parentID, name string) (*driveapi.File, error) {
	file, err := os.Open(localPath) //nolint:gosec // path is supplied by the user
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("reading file info: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("uploading file: %s is a directory", localPath)
	}
	if name == "" {
		name = filepath.Base(localPath)
	}
	typ, err := FileMIMEType(localPath)
	if err != nil {
		return nil, err
	}
	metadata := &drivev3.File{Name: name, MimeType: typ}
	if parentID != "" {
		metadata.Parents = []string{parentID}
	}
	created, err := c.service.Files.Create(metadata).Media(file).SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("uploading file: %w", err)
	}
	return driveapi.ParseFile(created), nil
}

// CreateFolder creates a Drive folder.
func (c *Client) CreateFolder(ctx context.Context, name, parentID string) (*driveapi.File, error) {
	metadata := &drivev3.File{Name: name, MimeType: driveapi.MimeTypeFolder}
	if parentID != "" {
		metadata.Parents = []string{parentID}
	}
	created, err := c.service.Files.Create(metadata).SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating folder: %w", err)
	}
	return driveapi.ParseFile(created), nil
}

// RenameFile renames a Drive file or folder.
func (c *Client) RenameFile(ctx context.Context, fileID, name string) (*driveapi.File, error) {
	updated, err := c.service.Files.Update(fileID, &drivev3.File{Name: name}).SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("renaming file: %w", err)
	}
	return driveapi.ParseFile(updated), nil
}

// MoveFile moves a Drive file or folder to a new parent.
func (c *Client) MoveFile(ctx context.Context, fileID, newParentID string) (*driveapi.File, error) {
	current, err := c.service.Files.Get(fileID).Fields("parents").SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting file parents: %w", err)
	}
	updated, err := c.service.Files.Update(fileID, &drivev3.File{}).
		AddParents(newParentID).RemoveParents(strings.Join(current.Parents, ",")).
		SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("moving file: %w", err)
	}
	return driveapi.ParseFile(updated), nil
}

// TrashFiles moves files to Trash.
func (c *Client) TrashFiles(ctx context.Context, ids []string) error {
	return c.setTrashed(ctx, ids, true)
}

// UntrashFiles restores files from Trash.
func (c *Client) UntrashFiles(ctx context.Context, ids []string) error {
	return c.setTrashed(ctx, ids, false)
}

func (c *Client) setTrashed(ctx context.Context, ids []string, trashed bool) error {
	for _, id := range ids {
		metadata := &drivev3.File{Trashed: trashed}
		if !trashed {
			metadata.ForceSendFields = []string{"Trashed"}
		}
		if _, err := c.service.Files.Update(id, metadata).SupportsAllDrives(true).Context(ctx).Do(); err != nil {
			return fmt.Errorf("setting trashed state for file %s: %w", id, err)
		}
	}
	return nil
}

// DeleteFilesPermanently permanently deletes files.
func (c *Client) DeleteFilesPermanently(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := c.service.Files.Delete(id).SupportsAllDrives(true).Context(ctx).Do(); err != nil {
			return fmt.Errorf("permanently deleting file %s: %w", id, err)
		}
	}
	return nil
}
