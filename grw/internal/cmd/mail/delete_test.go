package mail

import (
	"context"
	"strings"
	"testing"

	"github.com/open-cli-collective/google-cli-common/testutil"
)

func TestDelete_TrashByDefault(t *testing.T) {
	var trashed []string
	mock := &mockWriteClient{
		TrashMessagesFunc: func(_ context.Context, ids []string) error { trashed = ids; return nil },
		DeleteMessagesPermanentlyFunc: func(_ context.Context, _ []string) error {
			t.Fatal("permanent delete must not be called without --permanent")
			return nil
		},
	}
	var out string
	withMockClient(mock, func() {
		out = testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newDeleteCommand(), "id1", "id2"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if len(trashed) != 2 || trashed[0] != "id1" || trashed[1] != "id2" {
		t.Fatalf("TrashMessages got %v, want [id1 id2]", trashed)
	}
	if !strings.Contains(out, "Trashed 2 message(s).") {
		t.Errorf("output %q missing trashed count", out)
	}
}

func TestDelete_DryRunDoesNotMutate(t *testing.T) {
	mock := &mockWriteClient{
		TrashMessagesFunc: func(_ context.Context, _ []string) error {
			t.Fatal("dry-run must not trash")
			return nil
		},
	}
	var out string
	withMockClient(mock, func() {
		out = testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newDeleteCommand(), "id1", "--dry-run"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if !strings.Contains(out, "[dry-run] Would trash 1 message(s).") {
		t.Errorf("output %q missing dry-run line", out)
	}
}

func TestDelete_PermanentWithYes(t *testing.T) {
	var deleted []string
	mock := &mockWriteClient{
		DeleteMessagesPermanentlyFunc: func(_ context.Context, ids []string) error { deleted = ids; return nil },
		TrashMessagesFunc: func(_ context.Context, _ []string) error {
			t.Fatal("--permanent must not trash")
			return nil
		},
	}
	var out string
	withMockClient(mock, func() {
		out = testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newDeleteCommand(), "id1", "--permanent", "--yes"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if len(deleted) != 1 || deleted[0] != "id1" {
		t.Fatalf("DeleteMessagesPermanently got %v, want [id1]", deleted)
	}
	if !strings.Contains(out, "Permanently deleted 1 message(s).") {
		t.Errorf("output %q missing permanent-delete count", out)
	}
}

func TestDelete_PermanentTypedConfirmAborts(t *testing.T) {
	mock := &mockWriteClient{
		DeleteMessagesPermanentlyFunc: func(_ context.Context, _ []string) error {
			t.Fatal("must not delete when confirmation does not match")
			return nil
		},
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			cmd := newDeleteCommand()
			cmd.SetIn(strings.NewReader("nope\n"))
			_, err := runCmd(cmd, "id1", "--permanent")
			if err == nil || !strings.Contains(err.Error(), "aborted") {
				t.Fatalf("expected abort error, got %v", err)
			}
		})
	})
}

func TestDelete_PermanentTypedConfirmProceeds(t *testing.T) {
	var deleted []string
	mock := &mockWriteClient{
		DeleteMessagesPermanentlyFunc: func(_ context.Context, ids []string) error { deleted = ids; return nil },
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			cmd := newDeleteCommand()
			cmd.SetIn(strings.NewReader("delete\n"))
			if _, err := runCmd(cmd, "id1", "--permanent"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if len(deleted) != 1 {
		t.Fatalf("expected permanent delete of 1, got %v", deleted)
	}
}

func TestDelete_ResolvesQuery(t *testing.T) {
	var trashed []string
	mock := &mockWriteClient{
		SearchMessageIDsFunc: func(_ context.Context, q string, _ int64) ([]string, error) {
			if q != "older_than:1y" {
				t.Errorf("query = %q", q)
			}
			return []string{"a", "b", "c"}, nil
		},
		TrashMessagesFunc: func(_ context.Context, ids []string) error { trashed = ids; return nil },
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newDeleteCommand(), "--query", "older_than:1y"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if len(trashed) != 3 {
		t.Fatalf("expected 3 trashed from query, got %v", trashed)
	}
}

func TestDelete_NoMatch(t *testing.T) {
	mock := &mockWriteClient{
		SearchMessageIDsFunc: func(_ context.Context, _ string, _ int64) ([]string, error) { return nil, nil },
		TrashMessagesFunc: func(_ context.Context, _ []string) error {
			t.Fatal("must not trash when nothing matched")
			return nil
		},
	}
	var out string
	withMockClient(mock, func() {
		out = testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newDeleteCommand(), "--query", "no:match"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if !strings.Contains(out, "No messages matched.") {
		t.Errorf("output %q missing no-match message", out)
	}
}
