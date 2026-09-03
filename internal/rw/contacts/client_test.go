package contacts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"

	contactsapi "github.com/open-cli-collective/google-cli/internal/api/contacts"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	service, err := people.NewService(context.Background(), option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return &Client{service: service}
}

func TestContactAndGroupMutations(t *testing.T) {
	var requests []string
	var patched people.Person
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "people/c1"):
			_, _ = fmt.Fprint(w, `{"resourceName":"people/c1","etag":"person-etag","names":[{"givenName":"Ada","familyName":"Lovelace","displayName":"Ada Lovelace"}],"organizations":[{"name":"Analytical Engines","title":"Countess"}]}`)
		case r.Method == http.MethodPatch:
			_ = json.NewDecoder(r.Body).Decode(&patched)
			_, _ = fmt.Fprint(w, `{"resourceName":"people/c1","names":[{"givenName":"Augusta","familyName":"Lovelace"}]}`)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "contactGroups/123"):
			_, _ = fmt.Fprint(w, `{"resourceName":"contactGroups/123","etag":"group-etag"}`)
		case r.Method == http.MethodDelete:
			if r.URL.Query().Get("deleteContacts") != "" {
				t.Error("group delete must not set deleteContacts")
			}
			_, _ = fmt.Fprint(w, `{}`)
		case strings.Contains(r.URL.Path, "contactGroups"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			name := "Friends"
			if group, ok := body["contactGroup"].(map[string]any); ok && group["name"] != nil {
				name = group["name"].(string)
			}
			_, _ = fmt.Fprintf(w, `{"resourceName":"contactGroups/123","name":%q}`, name)
		default:
			_, _ = fmt.Fprint(w, `{"resourceName":"people/c1","names":[{"givenName":"Ada"}]}`)
		}
	})
	ctx := context.Background()
	created, err := client.CreateContact(ctx, &contactsapi.Contact{Names: []contactsapi.Name{{GivenName: "Ada"}}})
	if err != nil || created.ResourceName != "people/c1" {
		t.Fatalf("create = %#v, %v", created, err)
	}
	updated, err := client.UpdateContact(ctx, &contactsapi.Contact{
		ResourceName:  "people/c1",
		Names:         []contactsapi.Name{{GivenName: "Augusta"}},
		Organizations: []contactsapi.Organization{{Title: "Mathematician"}},
	})
	if err != nil || updated.Names[0].GivenName != "Augusta" {
		t.Fatalf("update = %#v, %v", updated, err)
	}
	if patched.Etag != "person-etag" {
		t.Errorf("patched etag = %q, want the fetched etag", patched.Etag)
	}
	if n := patched.Names[0]; n.GivenName != "Augusta" || n.FamilyName != "Lovelace" || n.DisplayName != "" {
		t.Errorf("patched name = %+v, want given name replaced and family name kept", n)
	}
	if o := patched.Organizations[0]; o.Name != "Analytical Engines" || o.Title != "Mathematician" {
		t.Errorf("patched organization = %+v, want title replaced and name kept", o)
	}
	if err := client.DeleteContact(ctx, "people/c1"); err != nil {
		t.Fatal(err)
	}
	group, err := client.CreateGroup(ctx, "Friends")
	if err != nil || group.Name != "Friends" {
		t.Fatalf("create group = %#v, %v", group, err)
	}
	group, err = client.RenameGroup(ctx, "contactGroups/123", "Work")
	if err != nil || group.Name != "Work" {
		t.Fatalf("rename group = %#v, %v", group, err)
	}
	if err := client.DeleteGroup(ctx, "contactGroups/123"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"POST /v1/people:createContact", "GET /v1/people/c1", "PATCH /v1/people/c1:updateContact", "DELETE /v1/people/c1:deleteContact",
		"POST /v1/contactGroups", "GET /v1/contactGroups/123", "PUT /v1/contactGroups/123", "DELETE /v1/contactGroups/123",
	}
	if strings.Join(requests, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests = %v, want %v", requests, want)
	}
}

func TestMutationErrorsWrap(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "no", http.StatusInternalServerError) })
	tests := []struct {
		name, want string
		call       func() error
	}{
		{"create contact", "creating contact", func() error { _, err := client.CreateContact(context.Background(), &contactsapi.Contact{}); return err }},
		{"update contact get", "getting contact for update", func() error {
			_, err := client.UpdateContact(context.Background(), &contactsapi.Contact{ResourceName: "people/c1", Names: []contactsapi.Name{{GivenName: "x"}}})
			return err
		}},
		{"delete contact", "deleting contact", func() error { return client.DeleteContact(context.Background(), "people/c1") }},
		{"create group", "creating contact group", func() error { _, err := client.CreateGroup(context.Background(), "x"); return err }},
		{"rename group get", "getting contact group for update", func() error { _, err := client.RenameGroup(context.Background(), "contactGroups/1", "x"); return err }},
		{"delete group", "deleting contact group", func() error { return client.DeleteGroup(context.Background(), "contactGroups/1") }},
	}
	for _, test := range tests {
		if err := test.call(); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%s error = %v", test.name, err)
		}
	}
}
