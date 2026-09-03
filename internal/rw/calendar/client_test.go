package calendar

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	calendarv3 "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	calendarapi "github.com/open-cli-collective/google-cli/internal/api/calendar"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	service, err := calendarv3.NewService(context.Background(), option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return &Client{service: service}
}

func TestEventMutations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, method, path string
		call               func(context.Context, *Client) error
	}{
		{"create", http.MethodPost, "/calendars/primary/events", func(ctx context.Context, client *Client) error {
			got, err := client.CreateEvent(ctx, "primary", &calendarapi.Event{Summary: "Created"})
			if err == nil && got.ID != "created" {
				t.Errorf("ID = %q", got.ID)
			}
			return err
		}},
		{"update", http.MethodPatch, "/calendars/primary/events/event-1", func(ctx context.Context, client *Client) error {
			got, err := client.UpdateEvent(ctx, "primary", &calendarapi.Event{ID: "event-1", Summary: "Updated"})
			if err == nil && got.Summary != "Updated" {
				t.Errorf("Summary = %q", got.Summary)
			}
			return err
		}},
		{"delete", http.MethodDelete, "/calendars/primary/events/event-1", func(ctx context.Context, client *Client) error {
			return client.DeleteEvent(ctx, "primary", "event-1")
		}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != test.method || r.URL.Path != test.path {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if test.method != http.MethodDelete {
					_, _ = fmt.Fprint(w, `{"id":"created","summary":"Updated"}`)
				}
			})
			if err := test.call(context.Background(), client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEventMutationErrorsWrap(t *testing.T) {
	t.Parallel()
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "no", http.StatusInternalServerError) })
	tests := []struct {
		name, want string
		call       func() error
	}{
		{"create", "creating event", func() error {
			_, err := client.CreateEvent(context.Background(), "primary", &calendarapi.Event{})
			return err
		}},
		{"update", "updating event", func() error {
			_, err := client.UpdateEvent(context.Background(), "primary", &calendarapi.Event{ID: "x"})
			return err
		}},
		{"delete", "deleting event", func() error { return client.DeleteEvent(context.Background(), "primary", "x") }},
	}
	for _, test := range tests {
		if err := test.call(); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%s error = %v", test.name, err)
		}
	}
}
