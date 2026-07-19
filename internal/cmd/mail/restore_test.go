package mail

import (
	"context"
	"strings"
	"testing"

	"github.com/open-cli-collective/google-cli-common/testutil"
)

func TestRestore_Untrashes(t *testing.T) {
	var untrashed []string
	mock := &mockWriteClient{
		UntrashMessagesFunc: func(_ context.Context, ids []string) error { untrashed = ids; return nil },
	}
	var out string
	withMockClient(mock, func() {
		out = testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newRestoreCommand(), "id1", "id2"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if len(untrashed) != 2 || untrashed[0] != "id1" {
		t.Fatalf("UntrashMessages got %v, want [id1 id2]", untrashed)
	}
	if !strings.Contains(out, "Restored 2 message(s).") {
		t.Errorf("output %q missing restored count", out)
	}
}

func TestRestore_DryRunDoesNotMutate(t *testing.T) {
	mock := &mockWriteClient{
		UntrashMessagesFunc: func(_ context.Context, _ []string) error {
			t.Fatal("dry-run must not untrash")
			return nil
		},
	}
	var out string
	withMockClient(mock, func() {
		out = testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newRestoreCommand(), "id1", "--dry-run"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if !strings.Contains(out, "[dry-run] Would restore 1 message(s).") {
		t.Errorf("output %q missing dry-run line", out)
	}
}
