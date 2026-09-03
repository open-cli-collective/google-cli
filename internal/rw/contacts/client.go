// Package contacts extends the read Contacts client with contact mutations.
package contacts

import (
	"context"
	"fmt"

	"google.golang.org/api/people/v1"

	contactsapi "github.com/open-cli-collective/google-cli/internal/api/contacts"
)

// Client is grw's Contacts client.
type Client struct {
	*contactsapi.Client
	service *people.Service
}

// NewClient builds a write-capable Contacts client.
func NewClient(ctx context.Context) (*Client, error) {
	base, err := contactsapi.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{Client: base, service: base.Service()}, nil
}

// CreateContact creates a Google contact.
func (c *Client) CreateContact(ctx context.Context, contact *contactsapi.Contact) (*contactsapi.Contact, error) {
	created, err := c.service.People.CreateContact(contactsapi.ToPerson(contact)).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating contact: %w", err)
	}
	return contactsapi.ParseContact(created), nil
}

// UpdateContact replaces the supplied field groups on a Google contact. A
// partial name or organization (for example only a new given name) is merged
// onto the contact's existing first entry so the untouched sub-fields survive.
func (c *Client) UpdateContact(ctx context.Context, contact *contactsapi.Contact) (*contactsapi.Contact, error) {
	current, err := c.service.People.Get(contact.ResourceName).PersonFields("metadata,names,organizations").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting contact for update: %w", err)
	}
	applyPersonFields(current, contactsapi.ToPerson(contact))
	updated, err := c.service.People.UpdateContact(contact.ResourceName, current).
		UpdatePersonFields(contactsapi.PersonUpdateMask(contact)).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("updating contact: %w", err)
	}
	return contactsapi.ParseContact(updated), nil
}

func applyPersonFields(dst, src *people.Person) {
	if len(src.Names) > 0 {
		if len(dst.Names) > 0 {
			mergeName(dst.Names[0], src.Names[0])
			src.Names[0] = dst.Names[0]
		}
		dst.Names = src.Names
	}
	if len(src.EmailAddresses) > 0 {
		dst.EmailAddresses = src.EmailAddresses
	}
	if len(src.PhoneNumbers) > 0 {
		dst.PhoneNumbers = src.PhoneNumbers
	}
	if len(src.Organizations) > 0 {
		if len(dst.Organizations) > 0 {
			mergeOrganization(dst.Organizations[0], src.Organizations[0])
			src.Organizations[0] = dst.Organizations[0]
		}
		dst.Organizations = src.Organizations
	}
	if len(src.Addresses) > 0 {
		dst.Addresses = src.Addresses
	}
	if len(src.Urls) > 0 {
		dst.Urls = src.Urls
	}
	if len(src.Biographies) > 0 {
		dst.Biographies = src.Biographies
	}
	if len(src.Birthdays) > 0 {
		dst.Birthdays = src.Birthdays
	}
}

func mergeName(dst, src *people.Name) {
	overlay(&dst.GivenName, src.GivenName)
	overlay(&dst.FamilyName, src.FamilyName)
	overlay(&dst.MiddleName, src.MiddleName)
	overlay(&dst.HonorificPrefix, src.HonorificPrefix)
	overlay(&dst.HonorificSuffix, src.HonorificSuffix)
	// DisplayName is derived by Google from the structured parts and a stale
	// PhoneticFullName would no longer match them; clear both so the server
	// recomputes rather than keeping the pre-edit values.
	dst.DisplayName, dst.PhoneticFullName = "", ""
}

func mergeOrganization(dst, src *people.Organization) {
	overlay(&dst.Name, src.Name)
	overlay(&dst.Title, src.Title)
	overlay(&dst.Department, src.Department)
}

func overlay(dst *string, src string) {
	if src != "" {
		*dst = src
	}
}

// DeleteContact deletes a Google contact.
func (c *Client) DeleteContact(ctx context.Context, resourceName string) error {
	if _, err := c.service.People.DeleteContact(resourceName).Context(ctx).Do(); err != nil {
		return fmt.Errorf("deleting contact: %w", err)
	}
	return nil
}

// CreateGroup creates a contact group.
func (c *Client) CreateGroup(ctx context.Context, name string) (*contactsapi.ContactGroup, error) {
	created, err := c.service.ContactGroups.Create(&people.CreateContactGroupRequest{
		ContactGroup: &people.ContactGroup{Name: name},
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating contact group: %w", err)
	}
	return contactsapi.ParseContactGroup(created), nil
}

// RenameGroup renames a contact group.
func (c *Client) RenameGroup(ctx context.Context, resourceName, name string) (*contactsapi.ContactGroup, error) {
	current, err := c.service.ContactGroups.Get(resourceName).GroupFields("metadata").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting contact group for update: %w", err)
	}
	updated, err := c.service.ContactGroups.Update(resourceName, &people.UpdateContactGroupRequest{
		ContactGroup:      &people.ContactGroup{Name: name, Etag: current.Etag},
		UpdateGroupFields: "name",
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("renaming contact group: %w", err)
	}
	return contactsapi.ParseContactGroup(updated), nil
}

// DeleteGroup deletes a contact group without deleting its members.
func (c *Client) DeleteGroup(ctx context.Context, resourceName string) error {
	if _, err := c.service.ContactGroups.Delete(resourceName).Context(ctx).Do(); err != nil {
		return fmt.Errorf("deleting contact group: %w", err)
	}
	return nil
}
