package mail

import (
	"context"
	"strings"
	"testing"

	gmailv1 "google.golang.org/api/gmail/v1"

	"github.com/open-cli-collective/google-cli/internal/testutil"
)

func TestFilter_Create(t *testing.T) {
	var got *gmailv1.Filter
	mock := &mockWriteClient{
		GetLabelIDFunc: func(_ context.Context, name string) (string, error) {
			if name != "Newsletters" {
				t.Errorf("add-label resolve name = %q", name)
			}
			return "Label_5", nil
		},
		CreateFilterFunc: func(_ context.Context, f *gmailv1.Filter) (*gmailv1.Filter, error) {
			got = f
			return &gmailv1.Filter{Id: "flt_1"}, nil
		},
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newFilterCreateCommand(),
				"--from", "news@example.com", "--add-label", "Newsletters", "--archive", "--mark-read"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if got == nil {
		t.Fatal("CreateFilter not called")
	}
	if got.Criteria == nil || got.Criteria.From != "news@example.com" {
		t.Errorf("criteria.From = %+v", got.Criteria)
	}
	if got.Action == nil || len(got.Action.AddLabelIds) != 1 || got.Action.AddLabelIds[0] != "Label_5" {
		t.Errorf("action.AddLabelIds = %+v, want [Label_5]", got.Action)
	}
	// --archive => remove INBOX, --mark-read => remove UNREAD
	joined := strings.Join(got.Action.RemoveLabelIds, ",")
	if !strings.Contains(joined, "INBOX") || !strings.Contains(joined, "UNREAD") {
		t.Errorf("action.RemoveLabelIds = %v, want INBOX and UNREAD", got.Action.RemoveLabelIds)
	}
}

func TestFilter_CreateDryRun(t *testing.T) {
	mock := &mockWriteClient{
		GetLabelIDFunc: func(context.Context, string) (string, error) {
			t.Fatal("dry-run must not resolve a label")
			return "", nil
		},
		CreateFilterFunc: func(context.Context, *gmailv1.Filter) (*gmailv1.Filter, error) {
			t.Fatal("dry-run must not create a filter")
			return nil, nil
		},
	}
	withMockClient(mock, func() {
		out := testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newFilterCreateCommand(), "--from", "news@example.com", "--add-label", "Newsletters", "--dry-run"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "[dry-run] Would create filter.") {
			t.Errorf("output %q missing dry-run line", out)
		}
	})
}

func TestFilter_CreateRequiresCriterion(t *testing.T) {
	withMockClient(&mockWriteClient{}, func() {
		_, err := runCmd(newFilterCreateCommand(), "--archive")
		if err == nil || !strings.Contains(err.Error(), "criterion") {
			t.Fatalf("expected criterion error, got %v", err)
		}
	})
}

func TestFilter_CreateRequiresAction(t *testing.T) {
	withMockClient(&mockWriteClient{}, func() {
		_, err := runCmd(newFilterCreateCommand(), "--from", "x@y.com")
		if err == nil || !strings.Contains(err.Error(), "action") {
			t.Fatalf("expected action error, got %v", err)
		}
	})
}

func TestFilter_List(t *testing.T) {
	mock := &mockWriteClient{
		ListFiltersFunc: func(_ context.Context) ([]*gmailv1.Filter, error) {
			return []*gmailv1.Filter{{
				Id:       "flt_1",
				Criteria: &gmailv1.FilterCriteria{From: "news@example.com"},
				Action:   &gmailv1.FilterAction{AddLabelIds: []string{"Label_5"}, RemoveLabelIds: []string{"INBOX"}},
			}}, nil
		},
		GetLabelsFunc: func() []*gmailv1.Label {
			return []*gmailv1.Label{{Id: "Label_5", Name: "Newsletters"}}
		},
	}
	var out string
	withMockClient(mock, func() {
		out = testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newFilterListCommand()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if !strings.Contains(out, "flt_1") || !strings.Contains(out, "from:news@example.com") {
		t.Errorf("list output %q missing filter/criteria", out)
	}
	if !strings.Contains(out, "label:Newsletters") || !strings.Contains(out, "archive") {
		t.Errorf("list output %q missing resolved action", out)
	}
}

func TestFilter_Remove(t *testing.T) {
	var deleted string
	mock := &mockWriteClient{
		DeleteFilterFunc: func(_ context.Context, id string) error { deleted = id; return nil },
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newFilterRemoveCommand(), "flt_9"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if deleted != "flt_9" {
		t.Fatalf("DeleteFilter(id=%q), want flt_9", deleted)
	}
}

func TestFilter_RemoveDryRun(t *testing.T) {
	mock := &mockWriteClient{DeleteFilterFunc: func(context.Context, string) error {
		t.Fatal("dry-run must not delete a filter")
		return nil
	}}
	withMockClient(mock, func() {
		out := testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newFilterRemoveCommand(), "flt_9", "-n"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "[dry-run] Would delete filter flt_9.") {
			t.Errorf("output %q missing dry-run line", out)
		}
	})
}
