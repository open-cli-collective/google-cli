package contacts

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	contactsapi "github.com/open-cli-collective/google-cli/internal/api/contacts"
)

type contactFlags struct {
	givenName, familyName, middleName, prefix, suffix string
	emails, phones, addresses, urls                   []string
	organization, title, department, biography        string
	birthday                                          string
	dryRun                                            bool
}

func newCreateCommand() *cobra.Command {
	var flags contactFlags
	cmd := &cobra.Command{
		Use: "create", Short: "Create a contact", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			contact, err := flags.contact(cmd, false)
			if err != nil {
				return err
			}
			if len(contact.Names) == 0 && len(contact.Emails) == 0 && len(contact.Phones) == 0 {
				return fmt.Errorf("set at least one of --given-name, --family-name, --email, or --phone")
			}
			if flags.dryRun {
				fmt.Println("[dry-run] Would create contact:")
				printContact(contact)
				return nil
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Contacts client: %w", err)
			}
			created, err := client.CreateContact(cmd.Context(), contact)
			if err != nil {
				return fmt.Errorf("creating contact: %w", err)
			}
			printContact(created)
			return nil
		},
	}
	flags.add(cmd)
	return cmd
}

func (f *contactFlags) add(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.givenName, "given-name", "", "Given name")
	cmd.Flags().StringVar(&f.familyName, "family-name", "", "Family name")
	cmd.Flags().StringVar(&f.middleName, "middle-name", "", "Middle name")
	cmd.Flags().StringVar(&f.prefix, "prefix", "", "Honorific prefix")
	cmd.Flags().StringVar(&f.suffix, "suffix", "", "Honorific suffix")
	cmd.Flags().StringArrayVar(&f.emails, "email", nil, "Email as value[:type] (repeatable)")
	cmd.Flags().StringArrayVar(&f.phones, "phone", nil, "Phone as value[:type] (repeatable)")
	cmd.Flags().StringVar(&f.organization, "org", "", "Organization name")
	cmd.Flags().StringVar(&f.title, "title", "", "Organization title")
	cmd.Flags().StringVar(&f.department, "department", "", "Organization department")
	cmd.Flags().StringArrayVar(&f.addresses, "address", nil, "Formatted address (repeatable)")
	cmd.Flags().StringArrayVar(&f.urls, "url", nil, "URL (repeatable)")
	cmd.Flags().StringVar(&f.biography, "biography", "", "Biography")
	cmd.Flags().StringVar(&f.birthday, "birthday", "", "Birthday (YYYY-MM-DD or --MM-DD)")
	cmd.Flags().BoolVarP(&f.dryRun, "dry-run", "n", false, "Preview without making changes")
}

func (f contactFlags) contact(cmd *cobra.Command, changedOnly bool) (*contactsapi.Contact, error) {
	c := &contactsapi.Contact{}
	changed := func(names ...string) bool {
		if !changedOnly {
			return true
		}
		for _, name := range names {
			if cmd.Flags().Changed(name) {
				return true
			}
		}
		return false
	}
	if f.hasNameInput(changed) {
		c.Names = []contactsapi.Name{{GivenName: f.givenName, FamilyName: f.familyName, MiddleName: f.middleName, HonorificPrefix: f.prefix, HonorificSuffix: f.suffix}}
	}
	if changed("email") {
		for i, raw := range f.emails {
			value, typ := splitType(raw)
			c.Emails = append(c.Emails, contactsapi.Email{Value: value, Type: typ, Primary: i == 0})
		}
	}
	if changed("phone") {
		for _, raw := range f.phones {
			value, typ := splitType(raw)
			c.Phones = append(c.Phones, contactsapi.Phone{Value: value, Type: typ})
		}
	}
	if f.hasOrganizationInput(changed) {
		c.Organizations = []contactsapi.Organization{{Name: f.organization, Title: f.title, Department: f.department}}
	}
	if changed("address") {
		for _, value := range f.addresses {
			c.Addresses = append(c.Addresses, contactsapi.Address{FormattedValue: value})
		}
	}
	if changed("url") {
		for _, value := range f.urls {
			c.URLs = append(c.URLs, contactsapi.URL{Value: value})
		}
	}
	if changed("biography") {
		c.Biography = f.biography
	}
	if changed("birthday") {
		if err := validateBirthday(f.birthday); err != nil {
			return nil, err
		}
		c.Birthday = f.birthday
	}
	return c, nil
}

// hasNameInput reports whether any name flag was set (per changed) to a
// non-empty value, so an empty names group is never sent.
func (f contactFlags) hasNameInput(changed func(...string) bool) bool {
	return changed("given-name", "family-name", "middle-name", "prefix", "suffix") &&
		(f.givenName != "" || f.familyName != "" || f.middleName != "" || f.prefix != "" || f.suffix != "")
}

// hasOrganizationInput is the organization counterpart of hasNameInput.
func (f contactFlags) hasOrganizationInput(changed func(...string) bool) bool {
	return changed("org", "title", "department") &&
		(f.organization != "" || f.title != "" || f.department != "")
}

func splitType(value string) (string, string) {
	if i := strings.LastIndex(value, ":"); i > 0 {
		return value[:i], value[i+1:]
	}
	return value, ""
}

func validateBirthday(value string) error {
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "--") {
		if len(value) == 7 {
			_, err := time.Parse(time.DateOnly, "2000-"+value[2:])
			if err == nil {
				return nil
			}
		}
	} else if _, err := time.Parse(time.DateOnly, value); err == nil {
		return nil
	}
	return fmt.Errorf("invalid --birthday: must be YYYY-MM-DD or --MM-DD")
}
