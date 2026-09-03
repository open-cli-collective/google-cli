package calendar

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	calendarapi "github.com/open-cli-collective/google-cli/internal/api/calendar"
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

func TestCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockWriteClient{CreateEventFunc: func(_ context.Context, calendarID string, event *calendarapi.Event) (*calendarapi.Event, error) {
			testutil.Equal(t, calendarID, "primary")
			testutil.Equal(t, event.End.DateTime, "2026-10-01T11:00:00-04:00")
			event.ID = "created"
			return event, nil
		}}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			output := testutil.CaptureStdout(t, func() {
				_, err := runCommand(newCreateCommand(), "--summary", "Meeting", "--start", "2026-10-01T10:00:00-04:00")
				testutil.NoError(t, err)
			})
			testutil.Contains(t, output, "ID: created")
		})
	})

	t.Run("API error", func(t *testing.T) {
		mock := &mockWriteClient{CreateEventFunc: func(context.Context, string, *calendarapi.Event) (*calendarapi.Event, error) {
			return nil, errors.New("API error")
		}}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			_, err := runCommand(newCreateCommand(), "--summary", "Meeting", "--start", "2026-10-01")
			if err == nil || !strings.Contains(err.Error(), "creating event") {
				t.Fatalf("error = %v", err)
			}
		})
	})

	t.Run("missing start", func(t *testing.T) {
		withFactory(func(context.Context) (WriteClient, error) { t.Fatal("client must not be constructed"); return nil, nil }, func() {
			_, err := runCommand(newCreateCommand(), "--summary", "Meeting")
			if err == nil || !strings.Contains(err.Error(), "start") {
				t.Fatalf("error = %v", err)
			}
		})
	})

	t.Run("client error", func(t *testing.T) {
		withFactory(func(context.Context) (WriteClient, error) { return nil, errors.New("no client") }, func() {
			_, err := runCommand(newCreateCommand(), "--summary", "Meeting", "--start", "2026-10-01")
			if err == nil || !strings.Contains(err.Error(), "creating Calendar client") {
				t.Fatalf("error = %v", err)
			}
		})
	})
}

func TestUpdate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockWriteClient{UpdateEventFunc: func(_ context.Context, calendarID string, event *calendarapi.Event) (*calendarapi.Event, error) {
			testutil.Equal(t, calendarID, "work")
			testutil.Equal(t, event.ID, "event-1")
			testutil.Equal(t, event.Summary, "Changed")
			return event, nil
		}}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			_, err := runCommand(newUpdateCommand(), "event-1", "--summary", "Changed", "--calendar", "work")
			testutil.NoError(t, err)
		})
	})

	t.Run("API error", func(t *testing.T) {
		mock := &mockWriteClient{UpdateEventFunc: func(context.Context, string, *calendarapi.Event) (*calendarapi.Event, error) {
			return nil, errors.New("API error")
		}}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			_, err := runCommand(newUpdateCommand(), "event-1", "--summary", "Changed")
			if err == nil || !strings.Contains(err.Error(), "updating event") {
				t.Fatalf("error = %v", err)
			}
		})
	})

	t.Run("client error", func(t *testing.T) {
		withFactory(func(context.Context) (WriteClient, error) { return nil, errors.New("no client") }, func() {
			_, err := runCommand(newUpdateCommand(), "event-1", "--summary", "Changed")
			if err == nil || !strings.Contains(err.Error(), "creating Calendar client") {
				t.Fatalf("error = %v", err)
			}
		})
	})

	t.Run("no fields", func(t *testing.T) {
		_, err := runCommand(newUpdateCommand(), "event-1")
		if err == nil || !strings.Contains(err.Error(), "at least one") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestDryRunsDoNotConstructClient(t *testing.T) {
	withFactory(func(context.Context) (WriteClient, error) { t.Fatal("dry-run constructed client"); return nil, nil }, func() {
		tests := []struct {
			name string
			cmd  *cobra.Command
			args []string
		}{
			{"create", newCreateCommand(), []string{"--summary", "x", "--start", "2026-10-01", "--dry-run"}},
			{"update", newUpdateCommand(), []string{"event-1", "--summary", "x", "--dry-run"}},
			{"delete", newDeleteCommand(), []string{"event-1", "--dry-run"}},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				testutil.CaptureStdout(t, func() { _, err := runCommand(test.cmd, test.args...); testutil.NoError(t, err) })
			})
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("requires confirmation", func(t *testing.T) {
		mock := &mockWriteClient{DeleteEventFunc: func(context.Context, string, string) error { t.Fatal("delete called"); return nil }}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			cmd := newDeleteCommand()
			cmd.SetIn(strings.NewReader("no\n"))
			_, err := runCommand(cmd, "event-1")
			if err == nil || !strings.Contains(err.Error(), "aborted") {
				t.Fatalf("error = %v", err)
			}
		})
	})

	t.Run("yes deletes", func(t *testing.T) {
		var deleted []string
		mock := &mockWriteClient{DeleteEventFunc: func(_ context.Context, calendarID, eventID string) error {
			testutil.Equal(t, calendarID, "primary")
			deleted = append(deleted, eventID)
			return nil
		}}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			output := testutil.CaptureStdout(t, func() {
				_, err := runCommand(newDeleteCommand(), "one", "two", "--yes")
				testutil.NoError(t, err)
			})
			testutil.Equal(t, strings.Join(deleted, ","), "one,two")
			testutil.Contains(t, output, "Deleted 2 event(s).")
		})
	})

	t.Run("API error", func(t *testing.T) {
		mock := &mockWriteClient{DeleteEventFunc: func(context.Context, string, string) error { return errors.New("API error") }}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			_, err := runCommand(newDeleteCommand(), "event-1", "--yes")
			if err == nil || !strings.Contains(err.Error(), "deleting event event-1") {
				t.Fatalf("error = %v", err)
			}
		})
	})

	t.Run("client error", func(t *testing.T) {
		withFactory(func(context.Context) (WriteClient, error) { return nil, errors.New("no client") }, func() {
			_, err := runCommand(newDeleteCommand(), "event-1", "--yes")
			if err == nil || !strings.Contains(err.Error(), "creating Calendar client") {
				t.Fatalf("error = %v", err)
			}
		})
	})
}
