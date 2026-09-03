package contacts

import (
	"context"

	contactsapi "github.com/open-cli-collective/google-cli/internal/api/contacts"
	contactscmd "github.com/open-cli-collective/google-cli/internal/cmd/contacts"
)

type mockWriteClient struct {
	contactscmd.ContactsClient
	CreateContactFunc    func(context.Context, *contactsapi.Contact) (*contactsapi.Contact, error)
	UpdateContactFunc    func(context.Context, *contactsapi.Contact) (*contactsapi.Contact, error)
	DeleteContactFunc    func(context.Context, string) error
	CreateGroupFunc      func(context.Context, string) (*contactsapi.ContactGroup, error)
	RenameGroupFunc      func(context.Context, string, string) (*contactsapi.ContactGroup, error)
	DeleteGroupFunc      func(context.Context, string) error
	ResolveGroupNameFunc func(context.Context, string) (string, error)
}

var _ WriteClient = (*mockWriteClient)(nil)

func (m *mockWriteClient) CreateContact(ctx context.Context, contact *contactsapi.Contact) (*contactsapi.Contact, error) {
	return m.CreateContactFunc(ctx, contact)
}
func (m *mockWriteClient) UpdateContact(ctx context.Context, contact *contactsapi.Contact) (*contactsapi.Contact, error) {
	return m.UpdateContactFunc(ctx, contact)
}
func (m *mockWriteClient) DeleteContact(ctx context.Context, resourceName string) error {
	return m.DeleteContactFunc(ctx, resourceName)
}
func (m *mockWriteClient) CreateGroup(ctx context.Context, name string) (*contactsapi.ContactGroup, error) {
	return m.CreateGroupFunc(ctx, name)
}
func (m *mockWriteClient) RenameGroup(ctx context.Context, resourceName, name string) (*contactsapi.ContactGroup, error) {
	return m.RenameGroupFunc(ctx, resourceName, name)
}
func (m *mockWriteClient) DeleteGroup(ctx context.Context, resourceName string) error {
	return m.DeleteGroupFunc(ctx, resourceName)
}
func (m *mockWriteClient) ResolveGroupName(ctx context.Context, name string) (string, error) {
	return m.ResolveGroupNameFunc(ctx, name)
}
