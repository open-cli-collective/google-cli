package contacts

import (
	"context"
	"fmt"
	"strings"

	contactsapi "github.com/open-cli-collective/google-cli/internal/api/contacts"
	contactscmd "github.com/open-cli-collective/google-cli/internal/cmd/contacts"
	contactsrw "github.com/open-cli-collective/google-cli/internal/rw/contacts"
	"github.com/open-cli-collective/google-cli/internal/sanitize"
)

// WriteClient is the Contacts surface used by grw commands.
type WriteClient interface {
	contactscmd.ContactsClient
	CreateContact(context.Context, *contactsapi.Contact) (*contactsapi.Contact, error)
	UpdateContact(context.Context, *contactsapi.Contact) (*contactsapi.Contact, error)
	DeleteContact(context.Context, string) error
	CreateGroup(context.Context, string) (*contactsapi.ContactGroup, error)
	RenameGroup(context.Context, string, string) (*contactsapi.ContactGroup, error)
	DeleteGroup(context.Context, string) error
}

// ClientFactory constructs grw's write Contacts client.
var ClientFactory = func(ctx context.Context) (WriteClient, error) { return contactsrw.NewClient(ctx) }

func newWriteClient(ctx context.Context) (WriteClient, error) { return ClientFactory(ctx) }

func printContact(contact *contactsapi.Contact) {
	if contact.ResourceName != "" {
		fmt.Printf("ID: %s\n", contact.ResourceName)
	}
	if len(contact.Names) > 0 {
		name := contact.Names[0]
		fmt.Printf("Name: %s\n", sanitize.Output(strings.TrimSpace(strings.Join([]string{name.HonorificPrefix, name.GivenName, name.MiddleName, name.FamilyName, name.HonorificSuffix}, " "))))
	}
	for _, email := range contact.Emails {
		fmt.Printf("Email: %s", sanitize.Output(email.Value))
		if email.Type != "" {
			fmt.Printf(" [%s]", email.Type)
		}
		if email.Primary {
			fmt.Print(" (primary)")
		}
		fmt.Println()
	}
	for _, phone := range contact.Phones {
		fmt.Printf("Phone: %s", sanitize.Output(phone.Value))
		if phone.Type != "" {
			fmt.Printf(" [%s]", phone.Type)
		}
		fmt.Println()
	}
	for _, organization := range contact.Organizations {
		fmt.Printf("Organization: %s", sanitize.Output(organization.Name))
		if organization.Title != "" {
			fmt.Printf(" (%s)", sanitize.Output(organization.Title))
		}
		if organization.Department != "" {
			fmt.Printf(" - %s", sanitize.Output(organization.Department))
		}
		fmt.Println()
	}
	for _, address := range contact.Addresses {
		fmt.Printf("Address: %s\n", sanitize.Output(address.FormattedValue))
	}
	for _, url := range contact.URLs {
		fmt.Printf("URL: %s\n", sanitize.Output(url.Value))
	}
	if contact.Biography != "" {
		fmt.Printf("Biography: %s\n", sanitize.Output(contact.Biography))
	}
	if contact.Birthday != "" {
		fmt.Printf("Birthday: %s\n", sanitize.Output(contact.Birthday))
	}
}

func printGroup(group *contactsapi.ContactGroup) {
	if group.ResourceName != "" {
		fmt.Printf("ID: %s\n", group.ResourceName)
	}
	fmt.Printf("Name: %s\n", sanitize.Output(group.Name))
}
