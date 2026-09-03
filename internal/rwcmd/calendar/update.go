package calendar

import (
	"fmt"

	"github.com/spf13/cobra"

	calendarapi "github.com/open-cli-collective/google-cli/internal/api/calendar"
)

func newUpdateCommand() *cobra.Command {
	var flags eventFlags
	cmd := &cobra.Command{
		Use:   "update <event-id>",
		Short: "Update a calendar event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			event, err := flags.updateEvent(cmd, args[0])
			if err != nil {
				return err
			}
			if flags.dryRun {
				fmt.Println("[dry-run] Would update event:")
				printEvent(event)
				return nil
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Calendar client: %w", err)
			}
			updated, err := client.UpdateEvent(cmd.Context(), flags.calendarID, event)
			if err != nil {
				return fmt.Errorf("updating event: %w", err)
			}
			printEvent(updated)
			return nil
		},
	}
	flags.add(cmd)
	return cmd
}

func (f eventFlags) updateEvent(cmd *cobra.Command, eventID string) (*calendarapi.Event, error) {
	fieldNames := []string{"summary", "description", "location", "start", "end", "timezone", "attendee"}
	changed := false
	for _, name := range fieldNames {
		changed = changed || cmd.Flags().Changed(name)
	}
	if !changed {
		return nil, fmt.Errorf("set at least one event field to update")
	}
	event := &calendarapi.Event{ID: eventID}
	if cmd.Flags().Changed("summary") {
		event.Summary = f.summary
	}
	if cmd.Flags().Changed("description") {
		event.Description = f.description
	}
	if cmd.Flags().Changed("location") {
		event.Location = f.location
	}
	if cmd.Flags().Changed("attendee") {
		for _, email := range f.attendees {
			event.Attendees = append(event.Attendees, calendarapi.Person{Email: email})
		}
	}
	if cmd.Flags().Changed("start") {
		start, allDay, err := parseEventTime(f.start, f.timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid --start: %w", err)
		}
		event.Start, event.AllDay = start, allDay
	}
	if cmd.Flags().Changed("end") {
		end, _, err := parseEventTime(f.end, f.timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid --end: %w", err)
		}
		event.End = end
	}
	if cmd.Flags().Changed("timezone") {
		if event.Start == nil && event.End == nil {
			return nil, fmt.Errorf("--timezone requires --start or --end")
		}
		if event.Start != nil {
			event.Start.TimeZone = f.timezone
		}
		if event.End != nil {
			event.End.TimeZone = f.timezone
		}
	}
	return event, nil
}
