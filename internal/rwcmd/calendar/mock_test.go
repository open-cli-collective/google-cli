package calendar

import (
	"context"

	calendarapi "github.com/open-cli-collective/google-cli/internal/api/calendar"
	calendarcmd "github.com/open-cli-collective/google-cli/internal/cmd/calendar"
)

type mockWriteClient struct {
	calendarcmd.CalendarClient
	CreateEventFunc func(context.Context, string, *calendarapi.Event) (*calendarapi.Event, error)
	UpdateEventFunc func(context.Context, string, *calendarapi.Event) (*calendarapi.Event, error)
	DeleteEventFunc func(context.Context, string, string) error
}

var _ WriteClient = (*mockWriteClient)(nil)

func (m *mockWriteClient) CreateEvent(ctx context.Context, calendarID string, event *calendarapi.Event) (*calendarapi.Event, error) {
	return m.CreateEventFunc(ctx, calendarID, event)
}

func (m *mockWriteClient) UpdateEvent(ctx context.Context, calendarID string, event *calendarapi.Event) (*calendarapi.Event, error) {
	return m.UpdateEventFunc(ctx, calendarID, event)
}

func (m *mockWriteClient) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	return m.DeleteEventFunc(ctx, calendarID, eventID)
}
