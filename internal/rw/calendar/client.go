// Package calendar extends the read Calendar client with event mutations.
package calendar

import (
	"context"
	"fmt"

	calendarv3 "google.golang.org/api/calendar/v3"

	calendarapi "github.com/open-cli-collective/google-cli/internal/api/calendar"
)

// Client is grw's Calendar client.
type Client struct {
	*calendarapi.Client
	service *calendarv3.Service
}

// NewClient builds a write-capable Calendar client.
func NewClient(ctx context.Context) (*Client, error) {
	base, err := calendarapi.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{Client: base, service: base.Service()}, nil
}

// CreateEvent creates an event on a calendar.
func (c *Client) CreateEvent(ctx context.Context, calendarID string, event *calendarapi.Event) (*calendarapi.Event, error) {
	created, err := c.service.Events.Insert(calendarID, calendarapi.ToAPIEvent(event)).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating event: %w", err)
	}
	return calendarapi.ParseEvent(created), nil
}

// UpdateEvent patches an event, leaving omitted fields unchanged.
func (c *Client) UpdateEvent(ctx context.Context, calendarID string, event *calendarapi.Event) (*calendarapi.Event, error) {
	updated, err := c.service.Events.Patch(calendarID, event.ID, calendarapi.ToAPIEvent(event)).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("updating event: %w", err)
	}
	return calendarapi.ParseEvent(updated), nil
}

// DeleteEvent deletes an event.
func (c *Client) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	if err := c.service.Events.Delete(calendarID, eventID).Context(ctx).Do(); err != nil {
		return fmt.Errorf("deleting event: %w", err)
	}
	return nil
}
