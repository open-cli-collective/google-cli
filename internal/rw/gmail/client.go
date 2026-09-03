// Package gmail extends the read/organize Gmail API client in internal/api/gmail
// with the read-WRITE operations grw adds: trash and
// permanent delete, label lifecycle (create/rename/delete — the "folders" and
// "subfolders" users think in), and filter management. It embeds the shared
// *gmail.Client so all read and non-destructive-organize methods (search, read,
// labels, ModifyMessages, ...) are available unchanged; the destructive surface
// remains isolated in this package.
package gmail

import (
	"context"
	"fmt"

	gmailv1 "google.golang.org/api/gmail/v1"

	gmailapi "github.com/open-cli-collective/google-cli/internal/api/gmail"
)

// Client is grw's Gmail client. It promotes every method of the shared
// *gmail.Client and adds the write operations below.
type Client struct {
	*gmailapi.Client
}

// NewClient builds a write-capable Gmail client using the shared OAuth/token
// machinery. The token must carry grw's scopes (see internal/app/grw);
// filters and permanent delete fail with a scope error otherwise.
func NewClient(ctx context.Context) (*Client, error) {
	base, err := gmailapi.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{Client: base}, nil
}

// ---------------------------------------------------------------------------
// Trash / delete
// ---------------------------------------------------------------------------

// TrashMessages moves messages to Trash (recoverable for ~30 days). It is
// implemented as a label mutation (add TRASH) through the shared BatchModify
// path, so it needs only gmail.modify — no destructive scope, and it is
// reversible via UntrashMessages.
func (c *Client) TrashMessages(ctx context.Context, ids []string) error {
	return c.ModifyMessages(ctx, ids, []string{"TRASH"}, nil)
}

// UntrashMessages restores messages from Trash.
func (c *Client) UntrashMessages(ctx context.Context, ids []string) error {
	return c.ModifyMessages(ctx, ids, nil, []string{"TRASH"})
}

// DeleteMessagesPermanently irreversibly erases messages via users.messages
// .batchDelete. This requires the https://mail.google.com/ scope and there is
// no recovery — callers must gate it behind explicit confirmation. IDs are
// chunked to stay within the API's per-request limit.
func (c *Client) DeleteMessagesPermanently(ctx context.Context, ids []string) error {
	for _, chunk := range chunkIDs(ids, 1000) {
		req := &gmailv1.BatchDeleteMessagesRequest{Ids: chunk}
		if err := c.Service().Users.Messages.BatchDelete(c.UserID(), req).Context(ctx).Do(); err != nil {
			return fmt.Errorf("permanently deleting messages: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Label lifecycle — "folders" and "subfolders" are labels whose names use "/"
// as the hierarchy separator (e.g. "Receipts/2026"). Gmail infers nesting from
// the name; grw does not special-case it beyond passing the name through.
// ---------------------------------------------------------------------------

// CreateLabel creates a user label. A name containing "/" nests it under the
// named parents, which Gmail renders as a collapsible folder tree.
func (c *Client) CreateLabel(ctx context.Context, name string) (*gmailv1.Label, error) {
	label := &gmailv1.Label{
		Name:                  name,
		LabelListVisibility:   "labelShow",
		MessageListVisibility: "show",
	}
	created, err := c.Service().Users.Labels.Create(c.UserID(), label).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating label %q: %w", name, err)
	}
	c.InvalidateLabels()
	return created, nil
}

// RenameLabel renames (or re-nests, by changing its "/" path) a user label.
func (c *Client) RenameLabel(ctx context.Context, id, newName string) (*gmailv1.Label, error) {
	updated, err := c.Service().Users.Labels.Patch(c.UserID(), id, &gmailv1.Label{Name: newName}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("renaming label: %w", err)
	}
	c.InvalidateLabels()
	return updated, nil
}

// DeleteLabel deletes a user label. Messages are not deleted; they simply lose
// the label. System labels cannot be deleted (the API rejects it).
func (c *Client) DeleteLabel(ctx context.Context, id string) error {
	if err := c.Service().Users.Labels.Delete(c.UserID(), id).Context(ctx).Do(); err != nil {
		return fmt.Errorf("deleting label: %w", err)
	}
	c.InvalidateLabels()
	return nil
}

// ---------------------------------------------------------------------------
// Filters
// ---------------------------------------------------------------------------

// ListFilters returns the account's Gmail filters.
func (c *Client) ListFilters(ctx context.Context) ([]*gmailv1.Filter, error) {
	resp, err := c.Service().Users.Settings.Filters.List(c.UserID()).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("listing filters: %w", err)
	}
	return resp.Filter, nil
}

// CreateFilter creates a Gmail filter (criteria + action).
func (c *Client) CreateFilter(ctx context.Context, filter *gmailv1.Filter) (*gmailv1.Filter, error) {
	created, err := c.Service().Users.Settings.Filters.Create(c.UserID(), filter).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating filter: %w", err)
	}
	return created, nil
}

// DeleteFilter deletes a Gmail filter by ID.
func (c *Client) DeleteFilter(ctx context.Context, id string) error {
	if err := c.Service().Users.Settings.Filters.Delete(c.UserID(), id).Context(ctx).Do(); err != nil {
		return fmt.Errorf("deleting filter: %w", err)
	}
	return nil
}

// chunkIDs splits ids into batches of at most size, so batch operations stay
// within the Gmail API's per-request limits.
func chunkIDs(ids []string, size int) [][]string {
	if size <= 0 {
		return [][]string{ids}
	}
	var chunks [][]string
	for i := 0; i < len(ids); i += size {
		end := i + size
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[i:end])
	}
	return chunks
}
