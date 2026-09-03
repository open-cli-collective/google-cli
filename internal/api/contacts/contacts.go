package contacts

import (
	"strings"
	"time"

	"google.golang.org/api/people/v1"
)

// Contact represents a simplified contact for output
type Contact struct {
	ResourceName  string         `json:"resourceName"`
	DisplayName   string         `json:"displayName,omitempty"`
	Names         []Name         `json:"names,omitempty"`
	Emails        []Email        `json:"emails,omitempty"`
	Phones        []Phone        `json:"phones,omitempty"`
	Organizations []Organization `json:"organizations,omitempty"`
	Addresses     []Address      `json:"addresses,omitempty"`
	URLs          []URL          `json:"urls,omitempty"`
	Biography     string         `json:"biography,omitempty"`
	Birthday      string         `json:"birthday,omitempty"`
	PhotoURL      string         `json:"photoUrl,omitempty"`
}

// Name represents a contact name
type Name struct {
	DisplayName      string `json:"displayName,omitempty"`
	GivenName        string `json:"givenName,omitempty"`
	FamilyName       string `json:"familyName,omitempty"`
	MiddleName       string `json:"middleName,omitempty"`
	HonorificPrefix  string `json:"honorificPrefix,omitempty"`
	HonorificSuffix  string `json:"honorificSuffix,omitempty"`
	PhoneticFullName string `json:"phoneticFullName,omitempty"`
}

// Email represents an email address
type Email struct {
	Value       string `json:"value"`
	Type        string `json:"type,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Primary     bool   `json:"primary,omitempty"`
}

// Phone represents a phone number
type Phone struct {
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// Organization represents a company/organization
type Organization struct {
	Name       string `json:"name,omitempty"`
	Title      string `json:"title,omitempty"`
	Department string `json:"department,omitempty"`
	Type       string `json:"type,omitempty"`
}

// Address represents a physical address
type Address struct {
	FormattedValue string `json:"formattedValue,omitempty"`
	Type           string `json:"type,omitempty"`
	City           string `json:"city,omitempty"`
	Region         string `json:"region,omitempty"`
	PostalCode     string `json:"postalCode,omitempty"`
	Country        string `json:"country,omitempty"`
}

// URL represents a website or link
type URL struct {
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// ContactGroup represents a contact group/label
type ContactGroup struct {
	ResourceName string `json:"resourceName"`
	Name         string `json:"name"`
	GroupType    string `json:"groupType,omitempty"`
	MemberCount  int64  `json:"memberCount"`
}

// ParseContact converts a People API Person to our Contact type
func ParseContact(p *people.Person) *Contact {
	if p == nil {
		return nil
	}

	contact := &Contact{
		ResourceName: p.ResourceName,
	}

	// Parse names
	if len(p.Names) > 0 {
		contact.DisplayName = p.Names[0].DisplayName
		contact.Names = make([]Name, len(p.Names))
		for i, n := range p.Names {
			contact.Names[i] = Name{
				DisplayName:      n.DisplayName,
				GivenName:        n.GivenName,
				FamilyName:       n.FamilyName,
				MiddleName:       n.MiddleName,
				HonorificPrefix:  n.HonorificPrefix,
				HonorificSuffix:  n.HonorificSuffix,
				PhoneticFullName: n.PhoneticFullName,
			}
		}
	}

	// Parse emails
	if len(p.EmailAddresses) > 0 {
		contact.Emails = make([]Email, len(p.EmailAddresses))
		for i, e := range p.EmailAddresses {
			contact.Emails[i] = Email{
				Value:       e.Value,
				Type:        e.Type,
				DisplayName: e.DisplayName,
			}
			if e.Metadata != nil && e.Metadata.Primary {
				contact.Emails[i].Primary = true
			}
		}
	}

	// Parse phones
	if len(p.PhoneNumbers) > 0 {
		contact.Phones = make([]Phone, len(p.PhoneNumbers))
		for i, ph := range p.PhoneNumbers {
			contact.Phones[i] = Phone{
				Value: ph.Value,
				Type:  ph.Type,
			}
		}
	}

	// Parse organizations
	if len(p.Organizations) > 0 {
		contact.Organizations = make([]Organization, len(p.Organizations))
		for i, o := range p.Organizations {
			contact.Organizations[i] = Organization{
				Name:       o.Name,
				Title:      o.Title,
				Department: o.Department,
				Type:       o.Type,
			}
		}
	}

	// Parse addresses
	if len(p.Addresses) > 0 {
		contact.Addresses = make([]Address, len(p.Addresses))
		for i, a := range p.Addresses {
			contact.Addresses[i] = Address{
				FormattedValue: a.FormattedValue,
				Type:           a.Type,
				City:           a.City,
				Region:         a.Region,
				PostalCode:     a.PostalCode,
				Country:        a.Country,
			}
		}
	}

	// Parse URLs
	if len(p.Urls) > 0 {
		contact.URLs = make([]URL, len(p.Urls))
		for i, u := range p.Urls {
			contact.URLs[i] = URL{
				Value: u.Value,
				Type:  u.Type,
			}
		}
	}

	// Parse biography
	if len(p.Biographies) > 0 {
		contact.Biography = p.Biographies[0].Value
	}

	// Parse birthday
	if len(p.Birthdays) > 0 && p.Birthdays[0].Date != nil {
		d := p.Birthdays[0].Date
		if d.Year > 0 {
			contact.Birthday = formatDate(d.Year, d.Month, d.Day)
		} else if d.Month > 0 {
			contact.Birthday = formatMonthDay(d.Month, d.Day)
		}
	}

	// Parse photo
	if len(p.Photos) > 0 {
		contact.PhotoURL = p.Photos[0].Url
	}

	return contact
}

// ToPerson converts a Contact to the writable People API fields.
func ToPerson(c *Contact) *people.Person {
	if c == nil {
		return nil
	}
	p := &people.Person{}
	for _, name := range c.Names {
		p.Names = append(p.Names, &people.Name{
			DisplayName: name.DisplayName, GivenName: name.GivenName, FamilyName: name.FamilyName,
			MiddleName: name.MiddleName, HonorificPrefix: name.HonorificPrefix,
			HonorificSuffix: name.HonorificSuffix, PhoneticFullName: name.PhoneticFullName,
		})
	}
	for _, email := range c.Emails {
		p.EmailAddresses = append(p.EmailAddresses, &people.EmailAddress{
			Value: email.Value, Type: email.Type, DisplayName: email.DisplayName,
			Metadata: &people.FieldMetadata{Primary: email.Primary},
		})
	}
	for _, phone := range c.Phones {
		p.PhoneNumbers = append(p.PhoneNumbers, &people.PhoneNumber{Value: phone.Value, Type: phone.Type})
	}
	for _, organization := range c.Organizations {
		p.Organizations = append(p.Organizations, &people.Organization{
			Name: organization.Name, Title: organization.Title,
			Department: organization.Department, Type: organization.Type,
		})
	}
	for _, address := range c.Addresses {
		p.Addresses = append(p.Addresses, &people.Address{
			FormattedValue: address.FormattedValue, Type: address.Type, City: address.City,
			Region: address.Region, PostalCode: address.PostalCode, Country: address.Country,
		})
	}
	for _, url := range c.URLs {
		p.Urls = append(p.Urls, &people.Url{Value: url.Value, Type: url.Type})
	}
	if c.Biography != "" {
		p.Biographies = []*people.Biography{{Value: c.Biography}}
	}
	if date := parseBirthday(c.Birthday); date != nil {
		p.Birthdays = []*people.Birthday{{Date: date}}
	}
	return p
}

// PersonUpdateMask returns the People API field groups present in c.
func PersonUpdateMask(c *Contact) string {
	if c == nil {
		return ""
	}
	groups := []struct {
		field   string
		present bool
	}{
		{"names", len(c.Names) > 0},
		{"emailAddresses", len(c.Emails) > 0},
		{"phoneNumbers", len(c.Phones) > 0},
		{"organizations", len(c.Organizations) > 0},
		{"addresses", len(c.Addresses) > 0},
		{"urls", len(c.URLs) > 0},
		{"biographies", c.Biography != ""},
		{"birthdays", c.Birthday != ""},
	}
	fields := make([]string, 0, len(groups))
	for _, group := range groups {
		if group.present {
			fields = append(fields, group.field)
		}
	}
	return strings.Join(fields, ",")
}

func parseBirthday(value string) *people.Date {
	if value == "" {
		return nil
	}
	value = strings.TrimPrefix(value, "--")
	if len(value) == len("01-02") {
		date, err := time.Parse("2006-01-02", "2000-"+value)
		if err != nil {
			return nil
		}
		return &people.Date{Month: int64(date.Month()), Day: int64(date.Day())}
	}
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return nil
	}
	return &people.Date{Year: int64(date.Year()), Month: int64(date.Month()), Day: int64(date.Day())}
}

// ParseContactGroup converts a People API ContactGroup to our ContactGroup type
func ParseContactGroup(g *people.ContactGroup) *ContactGroup {
	if g == nil {
		return nil
	}

	return &ContactGroup{
		ResourceName: g.ResourceName,
		Name:         g.Name,
		GroupType:    g.GroupType,
		MemberCount:  g.MemberCount,
	}
}

func formatDate(year int64, month int64, day int64) string {
	if year > 0 && month > 0 && day > 0 {
		return formatInt(year, 4) + "-" + formatInt(month, 2) + "-" + formatInt(day, 2)
	}
	return ""
}

func formatMonthDay(month int64, day int64) string {
	if month > 0 && day > 0 {
		return formatInt(month, 2) + "-" + formatInt(day, 2)
	}
	return ""
}

func formatInt(n int64, width int) string {
	s := ""
	for i := 0; i < width; i++ {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// GetDisplayName returns the best display name for a contact
func (c *Contact) GetDisplayName() string {
	if c.DisplayName != "" {
		return c.DisplayName
	}
	if len(c.Names) > 0 && c.Names[0].DisplayName != "" {
		return c.Names[0].DisplayName
	}
	if len(c.Emails) > 0 {
		return c.Emails[0].Value
	}
	return c.ResourceName
}

// GetPrimaryEmail returns the primary email or first email
func (c *Contact) GetPrimaryEmail() string {
	for _, e := range c.Emails {
		if e.Primary {
			return e.Value
		}
	}
	if len(c.Emails) > 0 {
		return c.Emails[0].Value
	}
	return ""
}

// GetPrimaryPhone returns the first phone number
func (c *Contact) GetPrimaryPhone() string {
	if len(c.Phones) > 0 {
		return c.Phones[0].Value
	}
	return ""
}

// GetOrganization returns the first organization name
func (c *Contact) GetOrganization() string {
	if len(c.Organizations) > 0 {
		if c.Organizations[0].Name != "" {
			return c.Organizations[0].Name
		}
		return c.Organizations[0].Title
	}
	return ""
}
