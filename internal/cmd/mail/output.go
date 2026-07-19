// Package mail is grw's Gmail command surface. It composes the shared
// non-destructive leaves from google-cli-common/mailcmd (search, read, thread,
// archive, label, move/categorize, star, mark read/unread, attachments, draft)
// and adds grw's read-WRITE leaves: delete (trash by default, permanent behind
// a guard), folder (label lifecycle), and filter management.
package mail

import (
	"context"

	gmailv1 "google.golang.org/api/gmail/v1"

	"github.com/open-cli-collective/google-readwrite/internal/gmailrw"
)

// WriteClient is the interface grw's destructive/settings leaves depend on. It
// is satisfied by *gmailrw.Client (which embeds the shared read/organize
// client, so the message-search and label-resolution reads come for free).
type WriteClient interface {
	SearchMessageIDs(ctx context.Context, query string, maxResults int64) ([]string, error)
	FetchLabels(ctx context.Context) error
	GetLabels() []*gmailv1.Label
	GetLabelID(ctx context.Context, name string) (string, error)

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
