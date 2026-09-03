package mail

import (
	"context"
	"errors"
	"strings"
	"testing"

	gmailapi "github.com/open-cli-collective/google-cli/internal/api/gmail"
	"github.com/open-cli-collective/google-cli/internal/testutil"
)

func draftWithRecipients() *gmailapi.DraftSummary {
	return &gmailapi.DraftSummary{
		From: "sender@example.com", To: "to@example.com", Cc: "cc@example.com",
		Bcc: "bcc@example.com", Subject: "Hello", AttachmentCount: 2,
	}
}

func TestSend_Success(t *testing.T) {
	mock := &mockWriteClient{
		GetDraftFunc: func(_ context.Context, id string) (*gmailapi.DraftSummary, error) {
			if id != "draft-1" {
				t.Errorf("draft ID = %q", id)
			}
			return draftWithRecipients(), nil
		},
		SendDraftFunc: func(_ context.Context, id string) (*gmailapi.SentResult, error) {
			if id != "draft-1" {
				t.Errorf("draft ID = %q", id)
			}
			return &gmailapi.SentResult{ID: "message-1", ThreadID: "thread-1"}, nil
		},
	}
	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			_, err := runCmd(newSendCommand(), "draft-1")
			testutil.NoError(t, err)
		})
		for _, want := range []string{
			"From: sender@example.com", "To: to@example.com", "Cc: cc@example.com",
			"Bcc: bcc@example.com", "Subject: Hello", "Attachments: 2",
			"Sent message message-1 in thread thread-1",
		} {
			testutil.Contains(t, output, want)
		}
	})
}

func TestSend_DryRunDoesNotSend(t *testing.T) {
	mock := &mockWriteClient{
		GetDraftFunc: func(context.Context, string) (*gmailapi.DraftSummary, error) { return draftWithRecipients(), nil },
		SendDraftFunc: func(context.Context, string) (*gmailapi.SentResult, error) {
			t.Fatal("dry-run must not send")
			return nil, nil
		},
	}
	withMockClient(mock, func() {
		output := testutil.CaptureStdout(t, func() {
			_, err := runCmd(newSendCommand(), "draft-1", "--dry-run")
			testutil.NoError(t, err)
		})
		testutil.Contains(t, output, "Subject: Hello")
	})
}

func TestSend_BccOnlyRecipientSends(t *testing.T) {
	sent := false
	mock := &mockWriteClient{
		GetDraftFunc: func(context.Context, string) (*gmailapi.DraftSummary, error) {
			return &gmailapi.DraftSummary{Bcc: "bcc@example.com", Subject: "Hello"}, nil
		},
		SendDraftFunc: func(context.Context, string) (*gmailapi.SentResult, error) {
			sent = true
			return &gmailapi.SentResult{ID: "message-1", ThreadID: "thread-1"}, nil
		},
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			_, err := runCmd(newSendCommand(), "draft-1")
			testutil.NoError(t, err)
		})
	})
	if !sent {
		t.Fatal("a Bcc-only draft must send")
	}
}

func TestSend_DryRunReportsMissingRecipients(t *testing.T) {
	mock := &mockWriteClient{
		GetDraftFunc: func(context.Context, string) (*gmailapi.DraftSummary, error) { return &gmailapi.DraftSummary{}, nil },
		SendDraftFunc: func(context.Context, string) (*gmailapi.SentResult, error) {
			t.Fatal("dry-run must not send")
			return nil, nil
		},
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			_, err := runCmd(newSendCommand(), "draft-1", "--dry-run")
			if err == nil || !strings.Contains(err.Error(), "no recipients") {
				t.Fatalf("error = %v", err)
			}
		})
	})
}

func TestSend_MissingRecipients(t *testing.T) {
	mock := &mockWriteClient{
		GetDraftFunc: func(context.Context, string) (*gmailapi.DraftSummary, error) { return &gmailapi.DraftSummary{}, nil },
		SendDraftFunc: func(context.Context, string) (*gmailapi.SentResult, error) {
			t.Fatal("draft without recipients must not send")
			return nil, nil
		},
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			_, err := runCmd(newSendCommand(), "draft-1")
			if err == nil || !strings.Contains(err.Error(), "no recipients") {
				t.Fatalf("error = %v", err)
			}
		})
	})
}

func TestSend_GetDraftError(t *testing.T) {
	mock := &mockWriteClient{GetDraftFunc: func(context.Context, string) (*gmailapi.DraftSummary, error) {
		return nil, errors.New("API error")
	}}
	withMockClient(mock, func() {
		_, err := runCmd(newSendCommand(), "draft-1")
		if err == nil || !strings.Contains(err.Error(), "getting draft") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestSend_SendDraftError(t *testing.T) {
	mock := &mockWriteClient{
		GetDraftFunc:  func(context.Context, string) (*gmailapi.DraftSummary, error) { return draftWithRecipients(), nil },
		SendDraftFunc: func(context.Context, string) (*gmailapi.SentResult, error) { return nil, errors.New("API error") },
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			_, err := runCmd(newSendCommand(), "draft-1")
			if err == nil || !strings.Contains(err.Error(), "sending draft") {
				t.Fatalf("error = %v", err)
			}
		})
	})
}

func TestSend_ClientCreationError(t *testing.T) {
	testutil.WithFactory(&ClientFactory, func(context.Context) (WriteClient, error) {
		return nil, errors.New("no client")
	}, func() {
		_, err := runCmd(newSendCommand(), "draft-1")
		if err == nil || !strings.Contains(err.Error(), "creating Gmail client") {
			t.Fatalf("error = %v", err)
		}
	})
}
