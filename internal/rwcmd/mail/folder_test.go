package mail

import (
	"context"
	"strings"
	"testing"

	gmailv1 "google.golang.org/api/gmail/v1"

	"github.com/open-cli-collective/google-cli/internal/testutil"
)

func TestFolder_Create(t *testing.T) {
	var created string
	mock := &mockWriteClient{
		CreateLabelFunc: func(_ context.Context, name string) (*gmailv1.Label, error) {
			created = name
			return &gmailv1.Label{Id: "Label_9", Name: name}, nil
		},
	}
	var out string
	withMockClient(mock, func() {
		out = testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newFolderCreateCommand(), "Receipts/2026"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if created != "Receipts/2026" {
		t.Fatalf("CreateLabel got %q, want Receipts/2026 (nested subfolder)", created)
	}
	if !strings.Contains(out, "Created folder") || !strings.Contains(out, "Receipts/2026") {
		t.Errorf("output %q missing created confirmation", out)
	}
}

func TestFolder_Rename(t *testing.T) {
	var gotID, gotNew string
	mock := &mockWriteClient{
		GetLabelIDFunc: func(_ context.Context, name string) (string, error) {
			if name != "Old" {
				t.Errorf("resolve name = %q", name)
			}
			return "Label_3", nil
		},
		RenameLabelFunc: func(_ context.Context, id, newName string) (*gmailv1.Label, error) {
			gotID, gotNew = id, newName
			return &gmailv1.Label{Id: id, Name: newName}, nil
		},
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newFolderRenameCommand(), "Old", "New"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if gotID != "Label_3" || gotNew != "New" {
		t.Fatalf("RenameLabel(id=%q,new=%q), want (Label_3,New)", gotID, gotNew)
	}
}

func TestFolder_Remove(t *testing.T) {
	var deletedID string
	mock := &mockWriteClient{
		GetLabelIDFunc:  func(_ context.Context, _ string) (string, error) { return "Label_7", nil },
		DeleteLabelFunc: func(_ context.Context, id string) error { deletedID = id; return nil },
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newFolderRemoveCommand(), "Junk"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if deletedID != "Label_7" {
		t.Fatalf("DeleteLabel(id=%q), want Label_7", deletedID)
	}
}
