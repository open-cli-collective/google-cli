package drive

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	drivev3 "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	service, err := drivev3.NewService(context.Background(), option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return &Client{service: service}
}

func TestFileMIMEType(t *testing.T) {
	t.Run("extension", func(t *testing.T) {
		got, err := FileMIMEType("report.pdf")
		if err != nil || !strings.HasPrefix(got, "application/pdf") {
			t.Fatalf("FileMIMEType = %q, %v", got, err)
		}
	})
	t.Run("content fallback", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "data.unknown-extension")
		if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := FileMIMEType(path)
		if err != nil || !strings.HasPrefix(got, "text/plain") {
			t.Fatalf("FileMIMEType = %q, %v", got, err)
		}
	})
}

func TestFileMutations(t *testing.T) {
	localPath := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(localPath, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	var requests []string
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("supportsAllDrives") != "true" {
			t.Errorf("%s %s missing supportsAllDrives=true", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, r.Method+" "+r.URL.Path+" "+r.URL.RawQuery+" "+string(body))
		switch r.Method {
		case http.MethodGet:
			_, _ = fmt.Fprint(w, `{"parents":["old-parent"]}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			_, _ = fmt.Fprint(w, `{"id":"file-1","name":"result","mimeType":"text/plain","parents":["parent-1"]}`)
		}
	})
	ctx := context.Background()
	uploaded, err := client.UploadFile(ctx, localPath, "parent-1", "")
	if err != nil || uploaded.ID != "file-1" {
		t.Fatalf("upload = %#v, %v", uploaded, err)
	}
	folder, err := client.CreateFolder(ctx, "Folder", "parent-1")
	if err != nil || folder.ID != "file-1" {
		t.Fatalf("mkdir = %#v, %v", folder, err)
	}
	renamed, err := client.RenameFile(ctx, "file-1", "Renamed")
	if err != nil || renamed.ID != "file-1" {
		t.Fatalf("rename = %#v, %v", renamed, err)
	}
	moved, err := client.MoveFile(ctx, "file-1", "new-parent")
	if err != nil || moved.ID != "file-1" {
		t.Fatalf("move = %#v, %v", moved, err)
	}
	if err := client.TrashFiles(ctx, []string{"file-1"}); err != nil {
		t.Fatal(err)
	}
	if err := client.UntrashFiles(ctx, []string{"file-1"}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteFilesPermanently(ctx, []string{"file-1"}); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(requests, "\n")
	for _, want := range []string{
		"POST /upload/drive/v3/files", filepath.Base(localPath), "content",
		`"mimeType":"application/vnd.google-apps.folder"`, `"name":"Renamed"`,
		"addParents=new-parent", "removeParents=old-parent", `"trashed":true`, `"trashed":false`,
		"DELETE /files/file-1",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("requests missing %q:\n%s", want, joined)
		}
	}
}

func TestMutationErrorsWrap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "no", http.StatusInternalServerError) })
	tests := []struct {
		name, want string
		call       func() error
	}{
		{"upload", "uploading file", func() error { _, err := client.UploadFile(context.Background(), path, "", ""); return err }},
		{"mkdir", "creating folder", func() error { _, err := client.CreateFolder(context.Background(), "x", ""); return err }},
		{"rename", "renaming file", func() error { _, err := client.RenameFile(context.Background(), "x", "y"); return err }},
		{"move", "getting file parents", func() error { _, err := client.MoveFile(context.Background(), "x", "y"); return err }},
		{"trash", "setting trashed state", func() error { return client.TrashFiles(context.Background(), []string{"x"}) }},
		{"restore", "setting trashed state", func() error { return client.UntrashFiles(context.Background(), []string{"x"}) }},
		{"delete", "permanently deleting file", func() error { return client.DeleteFilesPermanently(context.Background(), []string{"x"}) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}
