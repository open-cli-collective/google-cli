package mail

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gmailv1 "google.golang.org/api/gmail/v1"

	"github.com/open-cli-collective/google-cli/internal/testutil"
)

func vacationMock(current *gmailv1.VacationSettings, got **gmailv1.VacationSettings) *mockWriteClient {
	return &mockWriteClient{
		GetVacationFunc: func(context.Context) (*gmailv1.VacationSettings, error) {
			return current, nil
		},
		UpdateVacationFunc: func(_ context.Context, s *gmailv1.VacationSettings) (*gmailv1.VacationSettings, error) {
			*got = s
			return s, nil
		},
	}
}

func TestVacationGet(t *testing.T) {
	end := time.Date(2026, 3, 9, 0, 0, 0, 0, time.Local)
	mock := &mockWriteClient{
		GetVacationFunc: func(context.Context) (*gmailv1.VacationSettings, error) {
			return &gmailv1.VacationSettings{
				EnableAutoReply:       true,
				EndTime:               end.UnixMilli(),
				ResponseSubject:       "Away\x1b[31m",
				ResponseBodyPlainText: "plain body",
				ResponseBodyHtml:      "<p>html body</p>\x1b[0m",
			}, nil
		},
	}
	withMockClient(mock, func() {
		out := testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationGetCommand()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		for _, want := range []string{"Enabled: true", "Starts: -", "Ends before: " + end.Format(time.RFC3339), "<p>html body</p>"} {
			if !strings.Contains(out, want) {
				t.Errorf("output %q missing %q", out, want)
			}
		}
		if strings.Contains(out, "plain body") {
			t.Errorf("output %q shows the plain body, but Gmail sends the HTML body", out)
		}
		if strings.Contains(out, "\x1b") {
			t.Errorf("output %q contains an escape sequence", out)
		}
	})
}

func TestVacationUpdate_ChangesOnlyGivenFlags(t *testing.T) {
	var got *gmailv1.VacationSettings
	current := &gmailv1.VacationSettings{EnableAutoReply: true, ResponseSubject: "Keep", ResponseBodyHtml: "<p>old</p>", StartTime: 7, RestrictToContacts: true}
	withMockClient(vacationMock(current, &got), func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationUpdateCommand(), "--body", "New", "--end", "2026-03-08", "--domain-only"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if got == nil {
		t.Fatal("UpdateVacation not called")
	}
	if !got.EnableAutoReply || got.ResponseSubject != "Keep" || got.StartTime != 7 || !got.RestrictToContacts || !got.RestrictToDomain {
		t.Errorf("settings = %+v", got)
	}
	if got.ResponseBodyPlainText != "New" || got.ResponseBodyHtml != "" {
		t.Errorf("body not replaced: %+v", got)
	}
	// The whole last day is covered: Gmail replies only before EndTime.
	if want := time.Date(2026, 3, 9, 0, 0, 0, 0, time.Local).UnixMilli(); got.EndTime != want {
		t.Errorf("EndTime = %d, want %d", got.EndTime, want)
	}
}

func TestVacationUpdate_MapsEachFlagToItsField(t *testing.T) {
	var got *gmailv1.VacationSettings
	current := &gmailv1.VacationSettings{ResponseSubject: "Old", StartTime: 7, EndTime: 8}
	withMockClient(vacationMock(current, &got), func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationUpdateCommand(), "--subject", "New", "--start", "2026-03-02", "--end", "2026-03-08", "--contacts-only"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if got == nil {
		t.Fatal("UpdateVacation not called")
	}
	if got.ResponseSubject != "New" || !got.RestrictToContacts || got.RestrictToDomain {
		t.Errorf("settings = %+v", got)
	}
	if want := time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local).UnixMilli(); got.StartTime != want {
		t.Errorf("StartTime = %d, want %d", got.StartTime, want)
	}
	if want := time.Date(2026, 3, 9, 0, 0, 0, 0, time.Local).UnixMilli(); got.EndTime != want {
		t.Errorf("EndTime = %d, want %d", got.EndTime, want)
	}
}

func TestVacationUpdate_SendsFalseRestrictions(t *testing.T) {
	var got *gmailv1.VacationSettings
	current := &gmailv1.VacationSettings{RestrictToContacts: true, RestrictToDomain: true}
	withMockClient(vacationMock(current, &got), func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationUpdateCommand(), "--contacts-only=false", "--domain-only=false"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	wire, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"restrictToContacts":false`, `"restrictToDomain":false`} {
		if !strings.Contains(string(wire), want) {
			t.Errorf("request %s missing %s", wire, want)
		}
	}
}

func TestVacationUpdate_DryRunPreviewsResolvedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reply.txt")
	if err := os.WriteFile(path, []byte("from file"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := testutil.CaptureStdout(t, func() {
		if _, err := runCmd(newVacationUpdateCommand(), "--file", path, "--end", "2026-03-08", "--dry-run"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	end := time.Date(2026, 3, 9, 0, 0, 0, 0, time.Local).Format(time.RFC3339)
	for _, want := range []string{"body: from file", "ends before: " + end} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q missing %q", out, want)
		}
	}
}

func TestVacationUpdate_KeepsDisabledAndClearsDate(t *testing.T) {
	var got *gmailv1.VacationSettings
	withMockClient(vacationMock(&gmailv1.VacationSettings{EndTime: 42}, &got), func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationUpdateCommand(), "--end", ""); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if got == nil || got.EnableAutoReply || got.EndTime != 0 {
		t.Errorf("settings = %+v", got)
	}
}

func TestVacationUpdate_ReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reply.txt")
	if err := os.WriteFile(path, []byte("from file"), 0o600); err != nil {
		t.Fatal(err)
	}
	var got *gmailv1.VacationSettings
	withMockClient(vacationMock(&gmailv1.VacationSettings{}, &got), func() {
		testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationUpdateCommand(), "--file", path); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})
	if got == nil || got.ResponseBodyPlainText != "from file" {
		t.Errorf("settings = %+v", got)
	}
}

func TestVacationUpdate_DryRunDoesNotMutate(t *testing.T) {
	withMockClient(&mockWriteClient{}, func() {
		original := ClientFactory
		ClientFactory = func(context.Context) (WriteClient, error) {
			t.Fatal("dry-run must not construct a client")
			return nil, nil
		}
		defer func() { ClientFactory = original }()
		out := testutil.CaptureStdout(t, func() {
			if _, err := runCmd(newVacationUpdateCommand(), "--body", "Back soon", "--dry-run"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "[dry-run] Would update vacation responder:") || !strings.Contains(out, "body: Back soon") {
			t.Errorf("output %q missing preview", out)
		}
	})
}

func TestVacationUpdate_RejectsInvalidInput(t *testing.T) {
	file := filepath.Join(t.TempDir(), "reply.txt")
	if err := os.WriteFile(file, []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		args []string
		want string
	}{
		"no settings":               {nil, "at least one setting"},
		"both bodies":               {[]string{"--body", "a", "--file", file}, "mutually exclusive"},
		"missing file":              {[]string{"--file", file + ".none"}, "reading --file"},
		"bad start":                 {[]string{"--start", "March 2"}, "invalid --start"},
		"bad end":                   {[]string{"--end", "March 2"}, "invalid --end"},
		"reversed dates":            {[]string{"--start", "2026-03-08", "--end", "2026-03-02"}, "must be before"},
		"reversed dates in dry-run": {[]string{"--start", "2026-03-08", "--end", "2026-03-02", "--dry-run"}, "must be before"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockWriteClient{
				UpdateVacationFunc: func(context.Context, *gmailv1.VacationSettings) (*gmailv1.VacationSettings, error) {
					t.Fatal("invalid input must not update the responder")
					return nil, nil
				},
			}
			withMockClient(mock, func() {
				_, err := runCmd(newVacationUpdateCommand(), tt.args...)
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Errorf("error = %v, want %q", err, tt.want)
				}
			})
		})
	}
}

func TestVacationToggle_KeepsSettingsAndShowsResult(t *testing.T) {
	tests := map[string]struct {
		use     string
		enabled bool
		wire    string
		heading string
	}{
		"enable":  {"enable", true, `"enableAutoReply":true`, "turned on:"},
		"disable": {"disable", false, `"enableAutoReply":false`, "turned off:"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var got *gmailv1.VacationSettings
			current := &gmailv1.VacationSettings{EnableAutoReply: !tt.enabled, ResponseSubject: "Away", EndTime: 42}
			withMockClient(vacationMock(current, &got), func() {
				out := testutil.CaptureStdout(t, func() {
					if _, err := runCmd(newVacationToggleCommand(tt.use, tt.enabled)); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
				})
				if !strings.Contains(out, tt.heading) || !strings.Contains(out, "Subject: Away") {
					t.Errorf("output %q missing the post-state", out)
				}
			})
			if got == nil || got.ResponseSubject != "Away" || got.EndTime != 42 {
				t.Fatalf("settings not preserved: %+v", got)
			}
			wire, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(wire), tt.wire) {
				t.Errorf("request %s missing %s", wire, tt.wire)
			}
		})
	}
}

func TestVacationToggle_DryRunDoesNotMutate(t *testing.T) {
	original := ClientFactory
	ClientFactory = func(context.Context) (WriteClient, error) {
		t.Fatal("dry-run must not construct a client")
		return nil, nil
	}
	defer func() { ClientFactory = original }()
	testutil.CaptureStdout(t, func() {
		if _, err := runCmd(newVacationToggleCommand("disable", false), "--dry-run"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestVacation_APIErrors(t *testing.T) {
	apiErr := errors.New("api failed")
	failing := &mockWriteClient{
		GetVacationFunc: func(context.Context) (*gmailv1.VacationSettings, error) { return nil, apiErr },
	}
	withMockClient(failing, func() {
		if _, err := runCmd(newVacationGetCommand()); !errors.Is(err, apiErr) {
			t.Errorf("get error = %v", err)
		}
		if _, err := runCmd(newVacationToggleCommand("enable", true)); !errors.Is(err, apiErr) {
			t.Errorf("enable error = %v", err)
		}
		if _, err := runCmd(newVacationUpdateCommand(), "--body", "a"); !errors.Is(err, apiErr) {
			t.Errorf("update error = %v", err)
		}
	})
	rejecting := &mockWriteClient{
		UpdateVacationFunc: func(context.Context, *gmailv1.VacationSettings) (*gmailv1.VacationSettings, error) {
			return nil, apiErr
		},
	}
	withMockClient(rejecting, func() {
		if _, err := runCmd(newVacationToggleCommand("disable", false)); !errors.Is(err, apiErr) {
			t.Errorf("disable error = %v", err)
		}
	})
}

func TestVacation_ClientCreationError(t *testing.T) {
	createErr := errors.New("no credentials")
	original := ClientFactory
	ClientFactory = func(context.Context) (WriteClient, error) { return nil, createErr }
	defer func() { ClientFactory = original }()
	for _, cmd := range []struct {
		name string
		run  func() error
	}{
		{"get", func() error { _, err := runCmd(newVacationGetCommand()); return err }},
		{"update", func() error { _, err := runCmd(newVacationUpdateCommand(), "--body", "a"); return err }},
		{"disable", func() error { _, err := runCmd(newVacationToggleCommand("disable", false)); return err }},
	} {
		if err := cmd.run(); !errors.Is(err, createErr) {
			t.Errorf("%s error = %v", cmd.name, err)
		}
	}
}

func TestParseVacationTime(t *testing.T) {
	const instant = "2026-03-08T15:30:00-07:00"
	parsed, err := time.Parse(time.RFC3339, instant)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		value    string
		endOfDay bool
		want     int64
	}{
		{"empty", "", true, 0},
		{"timestamp as start", instant, false, parsed.UnixMilli()},
		{"timestamp as end is not moved", instant, true, parsed.UnixMilli()},
		{"date as start", "2026-03-08", false, time.Date(2026, 3, 8, 0, 0, 0, 0, time.Local).UnixMilli()},
		{"date as end moves to next midnight", "2026-03-08", true, time.Date(2026, 3, 9, 0, 0, 0, 0, time.Local).UnixMilli()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVacationTime(tt.value, tt.endOfDay)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
