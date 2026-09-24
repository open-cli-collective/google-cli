package initcmd

import (
	"path/filepath"
	"testing"
)

// TestEnsureCredentials_ReusesSiblingOAuthClient proves the seamless-setup
// path: when a sibling CLI already has an OAuth client JSON, init adopts it and
// never drives the paste wizard.
func TestEnsureCredentials_ReusesSiblingOAuthClient(t *testing.T) {
	fs := newFakeFS()
	d := baseDeps(t, fs)

	credPath := filepath.Join(t.TempDir(), "oauth_client.json")
	siblingPath := filepath.Join(t.TempDir(), "sibling-oauth_client.json")
	fs.files[siblingPath] = []byte(validOAuthJSON)

	prompter := &stubPrompter{}
	d.Prompter = prompter
	d.DiscoverSiblingClientJSON = func() (string, string, bool) { return siblingPath, "google-readonly", true }

	if err := ensureCredentials(d, &initOptions{}, credPath); err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	if _, ok := fs.files[credPath]; !ok {
		t.Fatal("expected the sibling OAuth client JSON to be written to credPath")
	}
	if len(prompter.calls) != 0 {
		t.Errorf("the paste wizard must not run when a sibling client is reused; calls=%v", prompter.calls)
	}
}

func TestEnsureCredentials_ReusesSiblingOAuthClientForSelectedProfile(t *testing.T) {
	fs := newFakeFS()
	d := baseDeps(t, fs)
	credPath := filepath.Join(t.TempDir(), "oauth_clients", "work.json")
	siblingPath := filepath.Join(t.TempDir(), "sibling-work.json")
	fs.files[siblingPath] = []byte(validOAuthJSON)
	var selected string
	d.Prompter = &stubPrompter{}
	d.DiscoverSiblingClientJSONForProfile = func(profile string) (string, string, bool) {
		selected = profile
		return siblingPath, "google-readonly", true
	}

	if err := ensureCredentialsForRef(d, &initOptions{}, credPath, "google-readonly/work"); err != nil {
		t.Fatalf("ensureCredentialsForRef: %v", err)
	}
	if selected != "work" {
		t.Fatalf("sibling profile = %q, want work", selected)
	}
	if _, ok := fs.files[credPath]; !ok {
		t.Fatal("expected the selected profile's sibling OAuth client to be written")
	}
}

// TestEnsureCredentials_NoSiblingFallsThroughToWizard proves the discovery does
// not hijack the normal path: with no sibling, the wizard still runs.
func TestEnsureCredentials_NoSiblingFallsThroughToWizard(t *testing.T) {
	fs := newFakeFS()
	d := baseDeps(t, fs)

	credPath := filepath.Join(t.TempDir(), "oauth_client.json")
	prompter := &stubPrompter{credChoice: "paste", pasteJSON: validOAuthJSON}
	d.Prompter = prompter
	d.DiscoverSiblingClientJSON = func() (string, string, bool) { return "", "", false }

	if err := ensureCredentials(d, &initOptions{}, credPath); err != nil {
		t.Fatalf("ensureCredentials: %v", err)
	}
	if len(prompter.calls) == 0 {
		t.Error("expected the paste wizard to run when no sibling client JSON exists")
	}
}
