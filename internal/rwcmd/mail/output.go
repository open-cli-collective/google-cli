// Package mail is grw's Gmail command surface. It composes the non-destructive
// leaves from internal/cmd/mail (search, read, thread,
// archive, label, move/categorize, star, mark read/unread, attachments, draft)
// and adds grw's read-WRITE leaves: send, delete (trash by default, permanent
// behind a guard), folder (label lifecycle), and filter management.
package mail

import (
	"context"

	gmailv1 "google.golang.org/api/gmail/v1"

	gmailapi "github.com/open-cli-collective/google-cli/internal/api/gmail"
	mailcmd "github.com/open-cli-collective/google-cli/internal/cmd/mail"
	gmailrw "github.com/open-cli-collective/google-cli/internal/rw/gmail"
)

// WriteClient is the interface grw's destructive/settings leaves depend on. It
// is satisfied by *gmailrw.Client (which embeds the shared read/organize
// client, so the message-search and label-resolution reads come for free).
type WriteClient interface {
	mailcmd.MailClient
	GetDraft(ctx context.Context, draftID string) (*gmailapi.DraftSummary, error)
	SendDraft(ctx context.Context, draftID string) (*gmailapi.SentResult, error)

	TrashMessages(ctx context.Context, ids []string) error
	UntrashMessages(ctx context.Context, ids []string) error
	DeleteMessagesPermanently(ctx context.Context, ids []string) error

	CreateLabel(ctx context.Context, name string) (*gmailv1.Label, error)
	RenameLabel(ctx context.Context, id, newName string) (*gmailv1.Label, error)
	DeleteLabel(ctx context.Context, id string) error

	ListFilters(ctx context.Context) ([]*gmailv1.Filter, error)
	CreateFilter(ctx context.Context, filter *gmailv1.Filter) (*gmailv1.Filter, error)
	DeleteFilter(ctx context.Context, id string) error
}

// ClientFactory constructs grw's write client. Override in tests to inject a
// mock.
var ClientFactory = func(ctx context.Context) (WriteClient, error) {
	return gmailrw.NewClient(ctx)
}

func newWriteClient(ctx context.Context) (WriteClient, error) {
	return ClientFactory(ctx)
}
