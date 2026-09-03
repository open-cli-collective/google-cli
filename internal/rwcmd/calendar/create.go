package calendar

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	calendarapi "github.com/open-cli-collective/google-cli/internal/api/calendar"
)

type eventFlags struct {
	summary, description, location, start, end, timezone, calendarID string
	attendees                                                        []string
	dryRun                                                           bool
}

func newCreateCommand() *cobra.Command {
	var flags eventFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a calendar event",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			event, err := flags.createEvent()
			if err != nil {
				return err
			}
			if flags.dryRun {
				fmt.Println("[dry-run] Would create event:")
				printEvent(event)
				return nil
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Calendar client: %w", err)
			}
			created, err := client.CreateEvent(cmd.Context(), flags.calendarID, event)
			if err != nil {
				return fmt.Errorf("creating event: %w", err)
			}
			printEvent(created)
			return nil
		},
	}
	flags.add(cmd)
	_ = cmd.MarkFlagRequired("summary")
	_ = cmd.MarkFlagRequired("start")
	return cmd
}

func (f *eventFlags) add(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.summary, "summary", "", "Event summary")
	cmd.Flags().StringVar(&f.description, "description", "", "Event description")
	cmd.Flags().StringVar(&f.location, "location", "", "Event location")
	cmd.Flags().StringVar(&f.start, "start", "", "Start time (RFC3339 or YYYY-MM-DD; a date makes an all-day event)")
	cmd.Flags().StringVar(&f.end, "end", "", "End time (RFC3339 or YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.timezone, "timezone", "", "Event timezone")
	cmd.Flags().StringSliceVar(&f.attendees, "attendee", nil, "Attendee email (repeatable)")
	cmd.Flags().StringVarP(&f.calendarID, "calendar", "c", "primary", "Calendar ID")
	cmd.Flags().BoolVarP(&f.dryRun, "dry-run", "n", false, "Preview without making changes")
}

func (f eventFlags) createEvent() (*calendarapi.Event, error) {
	event := &calendarapi.Event{Summary: f.summary, Description: f.description, Location: f.location}
	for _, email := range f.attendees {
		event.Attendees = append(event.Attendees, calendarapi.Person{Email: email})
	}
	if f.start == "" {
		return nil, fmt.Errorf("--start is required")
	}
	start, allDay, err := parseEventTime(f.start, f.timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid --start: %w", err)
	}
	event.Start, event.AllDay = start, allDay
	if f.end != "" {
		end, endAllDay, err := parseEventTime(f.end, f.timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid --end: %w", err)
		}
		if allDay != endAllDay {
			return nil, fmt.Errorf("--start and --end must both be dates or both be RFC3339 timestamps")
		}
		event.End = end
	} else if allDay {
		date, _ := time.Parse(time.DateOnly, f.start)
		event.End = &calendarapi.EventTime{Date: date.AddDate(0, 0, 1).Format(time.DateOnly), TimeZone: f.timezone}
	} else {
		dateTime, _ := time.Parse(time.RFC3339, f.start)
		event.End = &calendarapi.EventTime{DateTime: dateTime.Add(time.Hour).Format(time.RFC3339), TimeZone: f.timezone}
	}
	return event, nil
}

func parseEventTime(value, timezone string) (*calendarapi.EventTime, bool, error) {
	if _, err := time.Parse(time.DateOnly, value); err == nil {
		return &calendarapi.EventTime{Date: value, TimeZone: timezone}, true, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, false, fmt.Errorf("must be RFC3339 or YYYY-MM-DD: %w", err)
	}
	return &calendarapi.EventTime{DateTime: parsed.Format(time.RFC3339), TimeZone: timezone}, false, nil
}
