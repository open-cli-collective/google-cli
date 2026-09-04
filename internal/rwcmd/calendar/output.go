package calendar

import (
	"context"
	"fmt"

	calendarapi "github.com/open-cli-collective/google-cli/internal/api/calendar"
	calendarcmd "github.com/open-cli-collective/google-cli/internal/cmd/calendar"
	calendarrw "github.com/open-cli-collective/google-cli/internal/rw/calendar"
	"github.com/open-cli-collective/google-cli/internal/sanitize"
)

// WriteClient is the Calendar surface used by grw commands.
type WriteClient interface {
	calendarcmd.CalendarClient
	CreateEvent(ctx context.Context, calendarID string, event *calendarapi.Event) (*calendarapi.Event, error)
	UpdateEvent(ctx context.Context, calendarID string, event *calendarapi.Event) (*calendarapi.Event, error)
	DeleteEvent(ctx context.Context, calendarID, eventID string) error
}

// ClientFactory constructs grw's write Calendar client.
var ClientFactory = func(ctx context.Context) (WriteClient, error) {
	return calendarrw.NewClient(ctx)
}

func newWriteClient(ctx context.Context) (WriteClient, error) { return ClientFactory(ctx) }

func printEvent(event *calendarapi.Event) {
	if event.ID != "" {
		fmt.Printf("ID: %s\n", event.ID)
	}
	if event.Summary != "" {
		fmt.Printf("Summary: %s\n", sanitize.Output(event.Summary))
	}
	if event.Start != nil {
		fmt.Printf("Start: %s\n", eventTimeValue(event.Start))
	}
	if event.End != nil {
		fmt.Printf("End: %s\n", eventTimeValue(event.End))
	}
	if event.Location != "" {
		fmt.Printf("Location: %s\n", sanitize.Output(event.Location))
	}
	if event.Description != "" {
		fmt.Printf("Description: %s\n", sanitize.Output(event.Description))
	}
	for _, attendee := range event.Attendees {
		fmt.Printf("Attendee: %s\n", sanitize.Output(attendee.Email))
	}
}

func eventTimeValue(eventTime *calendarapi.EventTime) string {
	value := eventTime.DateTime
	if eventTime.Date != "" {
		value = eventTime.Date
	}
	if eventTime.TimeZone != "" {
		return fmt.Sprintf("%s (%s)", value, eventTime.TimeZone)
	}
	return value
}
