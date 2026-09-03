package appidentity

import (
	"strings"
	"testing"

	"google.golang.org/api/gmail/v1"
)

// TestScopes locks grw's OAuth scope set. Unlike gro (whose architecture test
// asserts every scope is non-destructive), grw is deliberately read-write: this
// test asserts the exact three Gmail scopes and that each is documented.
func TestScopes(t *testing.T) {
	want := []string{
		gmail.GmailModifyScope,
		gmail.GmailSettingsBasicScope,
		gmail.MailGoogleComScope,
	}
	if len(Scopes) != len(want) {
		t.Fatalf("Scopes = %v, want exactly %v", Scopes, want)
	}
	have := map[string]bool{}
	for _, s := range Scopes {
		have[s] = true
		if ScopeDescriptions[s] == "" {
			t.Errorf("scope %q has no description", s)
		}
	}
	for _, s := range want {
		if !have[s] {
			t.Errorf("missing required scope %q", s)
		}
	}
}

// TestReadWriteScopesPresent guards grw's whole reason to exist: the scopes gro
// never holds. gmail.settings.basic powers filters; mail.google.com powers
// permanent deletion.
func TestReadWriteScopesPresent(t *testing.T) {
	joined := strings.Join(Scopes, " ")
	if !strings.Contains(joined, "gmail.settings.basic") {
		t.Error("grw must request gmail.settings.basic (for filters)")
	}
	if !strings.Contains(joined, "mail.google.com") {
		t.Error("grw must request mail.google.com (for permanent delete)")
	}
}

func TestIdentity(t *testing.T) {
	id := Identity()
	if id.DirName != "google-readwrite" {
		t.Errorf("DirName = %q, want google-readwrite", id.DirName)
	}
	if id.DefaultRef != "google-readwrite/default" {
		t.Errorf("DefaultRef = %q, want google-readwrite/default", id.DefaultRef)
	}
	if id.ProductName != "grw" {
		t.Errorf("ProductName = %q, want grw", id.ProductName)
	}
}
