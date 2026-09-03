package mail

import (
	"context"

	gmailv1 "google.golang.org/api/gmail/v1"

	mailcmd "github.com/open-cli-collective/google-cli/internal/cmd/mail"
)

// mockWriteClient is a function-field mock of WriteClient. Unset fields return
// zero values, so each test wires only the methods it exercises.
type mockWriteClient struct {
	mailcmd.MailClient
	SearchMessageIDsFunc          func(ctx context.Context, query string, maxResults int64) ([]string, error)
	FetchLabelsFunc               func(ctx context.Context) error
	GetLabelsFunc                 func() []*gmailv1.Label
	GetLabelIDFunc                func(ctx context.Context, name string) (string, error)
	TrashMessagesFunc             func(ctx context.Context, ids []string) error
	UntrashMessagesFunc           func(ctx context.Context, ids []string) error
	DeleteMessagesPermanentlyFunc func(ctx context.Context, ids []string) error
	CreateLabelFunc               func(ctx context.Context, name string) (*gmailv1.Label, error)
	RenameLabelFunc               func(ctx context.Context, id, newName string) (*gmailv1.Label, error)
	DeleteLabelFunc               func(ctx context.Context, id string) error
	ListFiltersFunc               func(ctx context.Context) ([]*gmailv1.Filter, error)
	CreateFilterFunc              func(ctx context.Context, filter *gmailv1.Filter) (*gmailv1.Filter, error)
	DeleteFilterFunc              func(ctx context.Context, id string) error
}

var _ WriteClient = (*mockWriteClient)(nil)

func (m *mockWriteClient) SearchMessageIDs(ctx context.Context, query string, maxResults int64) ([]string, error) {
	if m.SearchMessageIDsFunc != nil {
		return m.SearchMessageIDsFunc(ctx, query, maxResults)
	}
	return nil, nil
}

func (m *mockWriteClient) FetchLabels(ctx context.Context) error {
	if m.FetchLabelsFunc != nil {
		return m.FetchLabelsFunc(ctx)
	}
	return nil
}

func (m *mockWriteClient) GetLabels() []*gmailv1.Label {
	if m.GetLabelsFunc != nil {
		return m.GetLabelsFunc()
	}
	return nil
}

func (m *mockWriteClient) GetLabelID(ctx context.Context, name string) (string, error) {
	if m.GetLabelIDFunc != nil {
		return m.GetLabelIDFunc(ctx, name)
	}
	return "", nil
}

func (m *mockWriteClient) TrashMessages(ctx context.Context, ids []string) error {
	if m.TrashMessagesFunc != nil {
		return m.TrashMessagesFunc(ctx, ids)
	}
	return nil
}

func (m *mockWriteClient) UntrashMessages(ctx context.Context, ids []string) error {
	if m.UntrashMessagesFunc != nil {
		return m.UntrashMessagesFunc(ctx, ids)
	}
	return nil
}

func (m *mockWriteClient) DeleteMessagesPermanently(ctx context.Context, ids []string) error {
	if m.DeleteMessagesPermanentlyFunc != nil {
		return m.DeleteMessagesPermanentlyFunc(ctx, ids)
	}
	return nil
}

func (m *mockWriteClient) CreateLabel(ctx context.Context, name string) (*gmailv1.Label, error) {
	if m.CreateLabelFunc != nil {
		return m.CreateLabelFunc(ctx, name)
	}
	return &gmailv1.Label{}, nil
}

func (m *mockWriteClient) RenameLabel(ctx context.Context, id, newName string) (*gmailv1.Label, error) {
	if m.RenameLabelFunc != nil {
		return m.RenameLabelFunc(ctx, id, newName)
	}
	return &gmailv1.Label{}, nil
}

func (m *mockWriteClient) DeleteLabel(ctx context.Context, id string) error {
	if m.DeleteLabelFunc != nil {
		return m.DeleteLabelFunc(ctx, id)
	}
	return nil
}

func (m *mockWriteClient) ListFilters(ctx context.Context) ([]*gmailv1.Filter, error) {
	if m.ListFiltersFunc != nil {
		return m.ListFiltersFunc(ctx)
	}
	return nil, nil
}

func (m *mockWriteClient) CreateFilter(ctx context.Context, filter *gmailv1.Filter) (*gmailv1.Filter, error) {
	if m.CreateFilterFunc != nil {
		return m.CreateFilterFunc(ctx, filter)
	}
	return &gmailv1.Filter{}, nil
}

func (m *mockWriteClient) DeleteFilter(ctx context.Context, id string) error {
	if m.DeleteFilterFunc != nil {
		return m.DeleteFilterFunc(ctx, id)
	}
	return nil
}
