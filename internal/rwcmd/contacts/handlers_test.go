package contacts

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	contactsapi "github.com/open-cli-collective/google-cli/internal/api/contacts"
	"github.com/open-cli-collective/google-cli/internal/testutil"
)

func withFactory(factory func(context.Context) (WriteClient, error), f func()) {
	testutil.WithFactory(&ClientFactory, factory, f)
}

func runCommand(cmd *cobra.Command, args ...string) (string, error) {
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return output.String(), err
}

func TestCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockWriteClient{CreateContactFunc: func(_ context.Context, contact *contactsapi.Contact) (*contactsapi.Contact, error) {
			if contact.Emails[0].Type != "work" || !contact.Emails[0].Primary {
				t.Fatalf("email = %#v", contact.Emails[0])
			}
			contact.ResourceName = "people/c1"
			return contact, nil
		}}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			output := testutil.CaptureStdout(t, func() {
				_, err := runCommand(newCreateCommand(), "--given-name", "Ada", "--email", "ada@example.com:work")
				testutil.NoError(t, err)
			})
			testutil.Contains(t, output, "ID: people/c1")
		})
	})
	t.Run("API error", func(t *testing.T) {
		mock := &mockWriteClient{CreateContactFunc: func(context.Context, *contactsapi.Contact) (*contactsapi.Contact, error) {
			return nil, errors.New("API error")
		}}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			_, err := runCommand(newCreateCommand(), "--email", "a@example.com")
			if err == nil || !strings.Contains(err.Error(), "creating contact") {
				t.Fatalf("error = %v", err)
			}
		})
	})
	t.Run("client error", func(t *testing.T) {
		withFactory(func(context.Context) (WriteClient, error) { return nil, errors.New("no client") }, func() {
			_, err := runCommand(newCreateCommand(), "--email", "a@example.com")
			if err == nil || !strings.Contains(err.Error(), "creating Contacts client") {
				t.Fatalf("error = %v", err)
			}
		})
	})
}

func TestUpdate(t *testing.T) {
	t.Run("success replaces selected groups", func(t *testing.T) {
		mock := &mockWriteClient{UpdateContactFunc: func(_ context.Context, contact *contactsapi.Contact) (*contactsapi.Contact, error) {
			testutil.Equal(t, contact.ResourceName, "people/c1")
			testutil.Len(t, contact.Phones, 2)
			if len(contact.Names) != 0 {
				t.Fatalf("unexpected names: %#v", contact.Names)
			}
			return contact, nil
		}}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			_, err := runCommand(newUpdateCommand(), "people/c1", "--phone", "1:home", "--phone", "2:work")
			testutil.NoError(t, err)
		})
	})
	t.Run("API error", func(t *testing.T) {
		mock := &mockWriteClient{UpdateContactFunc: func(context.Context, *contactsapi.Contact) (*contactsapi.Contact, error) {
			return nil, errors.New("API error")
		}}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			_, err := runCommand(newUpdateCommand(), "people/c1", "--given-name", "Ada")
			if err == nil || !strings.Contains(err.Error(), "updating contact") {
				t.Fatalf("error = %v", err)
			}
		})
	})
	t.Run("client error", func(t *testing.T) {
		withFactory(func(context.Context) (WriteClient, error) { return nil, errors.New("no client") }, func() {
			_, err := runCommand(newUpdateCommand(), "people/c1", "--given-name", "Ada")
			if err == nil || !strings.Contains(err.Error(), "creating Contacts client") {
				t.Fatalf("error = %v", err)
			}
		})
	})
	t.Run("no fields", func(t *testing.T) {
		_, err := runCommand(newUpdateCommand(), "people/c1")
		if err == nil || !strings.Contains(err.Error(), "at least one") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestDryRunsDoNotConstructClient(t *testing.T) {
	withFactory(func(context.Context) (WriteClient, error) { t.Fatal("dry-run constructed client"); return nil, nil }, func() {
		tests := []struct {
			name string
			cmd  *cobra.Command
			args []string
		}{
			{"create", newCreateCommand(), []string{"--email", "a@example.com", "--dry-run"}},
			{"update", newUpdateCommand(), []string{"people/c1", "--given-name", "Ada", "--dry-run"}},
			{"delete", newDeleteCommand(), []string{"people/c1", "--dry-run"}},
			{"group create", newGroupCreateCommand(), []string{"Friends", "--dry-run"}},
			{"group rename", newGroupRenameCommand(), []string{"Friends", "Work", "--dry-run"}},
			{"group rm", newGroupDeleteCommand(), []string{"contactGroups/123", "--dry-run"}},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				testutil.CaptureStdout(t, func() { _, err := runCommand(test.cmd, test.args...); testutil.NoError(t, err) })
			})
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("requires confirmation", func(t *testing.T) {
		mock := &mockWriteClient{DeleteContactFunc: func(context.Context, string) error { t.Fatal("delete called"); return nil }}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			cmd := newDeleteCommand()
			cmd.SetIn(strings.NewReader("no\n"))
			_, err := runCommand(cmd, "people/c1")
			if err == nil || !strings.Contains(err.Error(), "aborted") {
				t.Fatalf("error = %v", err)
			}
		})
	})
	t.Run("yes deletes", func(t *testing.T) {
		var deleted []string
		mock := &mockWriteClient{DeleteContactFunc: func(_ context.Context, id string) error { deleted = append(deleted, id); return nil }}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			output := testutil.CaptureStdout(t, func() {
				_, err := runCommand(newDeleteCommand(), "people/c1", "people/c2", "--yes")
				testutil.NoError(t, err)
			})
			testutil.Equal(t, strings.Join(deleted, ","), "people/c1,people/c2")
			testutil.Contains(t, output, "Deleted 2 contact(s).")
		})
	})
	t.Run("API error", func(t *testing.T) {
		mock := &mockWriteClient{DeleteContactFunc: func(context.Context, string) error { return errors.New("API error") }}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			_, err := runCommand(newDeleteCommand(), "people/c1", "--yes")
			if err == nil || !strings.Contains(err.Error(), "deleting contact people/c1") {
				t.Fatalf("error = %v", err)
			}
		})
	})
	t.Run("client error", func(t *testing.T) {
		withFactory(func(context.Context) (WriteClient, error) { return nil, errors.New("no client") }, func() {
			_, err := runCommand(newDeleteCommand(), "people/c1", "--yes")
			if err == nil || !strings.Contains(err.Error(), "creating Contacts client") {
				t.Fatalf("error = %v", err)
			}
		})
	})
}

func TestGroups(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockWriteClient{ResolveGroupNameFunc: func(context.Context, string) (string, error) { return "contactGroups/123", nil }, CreateGroupFunc: func(_ context.Context, name string) (*contactsapi.ContactGroup, error) {
			return &contactsapi.ContactGroup{ResourceName: "contactGroups/123", Name: name}, nil
		}, RenameGroupFunc: func(_ context.Context, resourceName, name string) (*contactsapi.ContactGroup, error) {
			return &contactsapi.ContactGroup{ResourceName: resourceName, Name: name}, nil
		}, DeleteGroupFunc: func(context.Context, string) error { return nil }}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			tests := []struct {
				cmd  *cobra.Command
				args []string
			}{
				{newGroupCreateCommand(), []string{"Friends"}},
				{newGroupRenameCommand(), []string{"Friends", "Work"}},
				{newGroupDeleteCommand(), []string{"Friends", "--yes"}},
			}
			for _, test := range tests {
				testutil.CaptureStdout(t, func() { _, err := runCommand(test.cmd, test.args...); testutil.NoError(t, err) })
			}
		})
	})
	t.Run("API error", func(t *testing.T) {
		mock := &mockWriteClient{CreateGroupFunc: func(context.Context, string) (*contactsapi.ContactGroup, error) { return nil, errors.New("API error") }}
		withFactory(func(context.Context) (WriteClient, error) { return mock, nil }, func() {
			_, err := runCommand(newGroupCreateCommand(), "Friends")
			if err == nil || !strings.Contains(err.Error(), "creating contact group") {
				t.Fatalf("error = %v", err)
			}
		})
	})
	t.Run("client error", func(t *testing.T) {
		withFactory(func(context.Context) (WriteClient, error) { return nil, errors.New("no client") }, func() {
			_, err := runCommand(newGroupCreateCommand(), "Friends")
			if err == nil || !strings.Contains(err.Error(), "creating Contacts client") {
				t.Fatalf("error = %v", err)
			}
		})
	})
	t.Run("rm refuses system group", func(t *testing.T) {
		withFactory(func(context.Context) (WriteClient, error) { t.Fatal("client constructed"); return nil, nil }, func() {
			_, err := runCommand(newGroupDeleteCommand(), "contactGroups/starred", "--yes")
			if err == nil || !strings.Contains(err.Error(), "system") {
				t.Fatalf("error = %v", err)
			}
		})
	})
}
