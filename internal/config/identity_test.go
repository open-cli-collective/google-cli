package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/cli-common/statedir"
	"github.com/open-cli-collective/cli-common/statedirtest"
	"gopkg.in/yaml.v3"
)

func TestSiblingOAuthClientPathForProfileIsProfileScoped(t *testing.T) {
	statedirtest.Hermetic(t)
	t.Cleanup(RegisterForTest)
	Register(Identity{
		DirName:           "google-readonly",
		DefaultRef:        "google-readonly/default",
		ProductName:       "gro",
		Scopes:            Scopes(),
		ScopeDescriptions: ScopeDescriptions(),
		SiblingDirNames:   []string{"sibling-full", "sibling-default"},
	})

	fullDir, err := (statedir.Scope{Name: "sibling-full"}).ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	defaultOnlyDir, err := (statedir.Scope{Name: "sibling-default"}).ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	defaultPath := filepath.Join(fullDir, "oauth_clients", "default.json")
	workPath := filepath.Join(fullDir, "oauth_clients", "work.json")
	if err := os.MkdirAll(filepath.Dir(defaultPath), DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(defaultOnlyDir, DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(defaultPath, []byte("full-default-client"), TokenPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workPath, []byte("full-work-client"), TokenPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(defaultOnlyDir, OAuthClientFile), []byte("default-only-client"), TokenPerm); err != nil {
		t.Fatal(err)
	}
	configData, err := yaml.Marshal(struct {
		CredentialRef string                   `yaml:"credential_ref"`
		Profiles      map[string]ProfileConfig `yaml:"profiles"`
	}{
		CredentialRef: "sibling-full/default",
		Profiles: map[string]ProfileConfig{
			"default": {OAuthClientPath: defaultPath},
			"work":    {OAuthClientPath: workPath},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fullDir, ConfigFileYAML), configData, TokenPerm); err != nil {
		t.Fatal(err)
	}

	if got, sibling, ok := SiblingOAuthClientPathForProfile("default"); !ok || got != defaultPath || sibling != "sibling-full" {
		t.Fatalf("default sibling = (%q, %q, %v), want (%q, sibling-full, true)", got, sibling, ok, defaultPath)
	}
	if got, sibling, ok := SiblingOAuthClientPathForProfile("work"); !ok || got != workPath || sibling != "sibling-full" {
		t.Fatalf("work sibling = (%q, %q, %v), want (%q, sibling-full, true)", got, sibling, ok, workPath)
	}

	if err := os.Remove(workPath); err != nil {
		t.Fatal(err)
	}
	if got, sibling, ok := SiblingOAuthClientPathForProfile("work"); ok {
		t.Fatalf("work selected a fallback client after its own client disappeared: (%q, %q)", got, sibling)
	}
}
