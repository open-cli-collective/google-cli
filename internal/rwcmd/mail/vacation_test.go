package mail

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	gmailv1 "google.golang.org/api/gmail/v1"

	"github.com/open-cli-collective/google-cli/internal/testutil"
)

func TestVacationSet_SendsSettings(t *testing.T) {
	var got *gmailv1.VacationSettings
	mock := &mockWriteClient{
		UpdateVacationFunc: func(_ context.Context, s *gmailv1.VacationSettings) (*gmailv1.VacationSettings, error) {
			got = s
			return s, nil
		},
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationSetCommand(), "--subject", "Away", "--body", "Back soon",
				"--start", "2026-03-02", "--end", "2026-03-08", "--domain-only"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if got == nil {
		t.Fatal("UpdateVacation not called")
	}
	if !got.EnableAutoReply || got.ResponseSubject != "Away" || got.ResponseBodyPlainText != "Back soon" || !got.RestrictToDomain || got.RestrictToContacts {
		t.Errorf("settings = %+v", got)
	}
	if want := time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local).UnixMilli(); got.StartTime != want {
		t.Errorf("StartTime = %d, want %d", got.StartTime, want)
	}
	// The whole last day is covered: Gmail replies only before EndTime.
	if want := time.Date(2026, 3, 9, 0, 0, 0, 0, time.Local).UnixMilli(); got.EndTime != want {
		t.Errorf("EndTime = %d, want %d", got.EndTime, want)
	}
}

func TestVacationSet_DryRunDoesNotMutate(t *testing.T) {
	mock := &mockWriteClient{
		UpdateVacationFunc: func(context.Context, *gmailv1.VacationSettings) (*gmailv1.VacationSettings, error) {
			t.Fatal("dry-run must not update the responder")
			return nil, nil
		},
	}
	withMockClient(mock, func() {
		out := testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationSetCommand(), "--body", "Back soon", "--dry-run"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "[dry-run] Would set vacation responder:") || !strings.Contains(out, "Back soon") {
			t.Errorf("output %q missing preview", out)
		}
	})
}

func TestVacationSet_RejectsInvalidInput(t *testing.T) {
	tests := map[string][]string{
		"no content":     {"--start", "2026-03-02"},
		"both bodies":    {"--body", "a", "--body-file", "b"},
		"bad date":       {"--body", "a", "--start", "March 2"},
		"reversed dates": {"--body", "a", "--start", "2026-03-08", "--end", "2026-03-02"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockWriteClient{
				UpdateVacationFunc: func(context.Context, *gmailv1.VacationSettings) (*gmailv1.VacationSettings, error) {
					t.Fatal("invalid input must not update the responder")
					return nil, nil
				},
			}
			withMockClient(mock, func() {
				if _, err := runCmd(newVacationSetCommand(), args...); err == nil {
					t.Error("expected an error")
				}
			})
		})
	}
}

func TestVacationOff_KeepsSettingsAndSendsFalse(t *testing.T) {
	var got *gmailv1.VacationSettings
	mock := &mockWriteClient{
		GetVacationFunc: func(context.Context) (*gmailv1.VacationSettings, error) {
			return &gmailv1.VacationSettings{EnableAutoReply: true, ResponseSubject: "Away", EndTime: 42}, nil
		},
		UpdateVacationFunc: func(_ context.Context, s *gmailv1.VacationSettings) (*gmailv1.VacationSettings, error) {
			got = s
			return s, nil
		},
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationOffCommand()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if got == nil {
		t.Fatal("UpdateVacation not called")
	}
	if got.ResponseSubject != "Away" || got.EndTime != 42 {
		t.Errorf("settings not preserved: %+v", got)
	}
	wire, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wire), `"enableAutoReply":false`) {
		t.Errorf("request %s must send enableAutoReply=false explicitly", wire)
	}
}

func TestVacationOff_DryRunDoesNotMutate(t *testing.T) {
	mock := &mockWriteClient{
		GetVacationFunc: func(context.Context) (*gmailv1.VacationSettings, error) {
			t.Fatal("dry-run must not read the responder")
			return nil, nil
		},
	}
	withMockClient(mock, func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationOffCommand(), "--dry-run"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
}
