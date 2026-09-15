package keychain

import (
	"errors"
	"strings"
	"testing"

	"github.com/open-cli-collective/cli-common/credstore"
	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/credtest"
)

// TestListProfilesAndHasTokenFor covers the cross-profile read surface that
// backs `profiles list`: enumeration reflects stored reality, and presence
// checks work against profiles other than the one the Store was opened for.
func TestListProfilesAndHasTokenFor(t *testing.T) {
	credtest.Setup(t)

	// Fresh service: nothing stored yet.
	st, err := openWith(testCfg(), false, false)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = st.Close() }()
	if ps, lerr := st.ListProfiles(); lerr != nil || len(ps) != 0 {
		t.Fatalf("fresh ListProfiles = (%v, %v), want empty", ps, lerr)
	}

	// Seed tokens under the active and a second profile.
	if err := st.SetToken(&oauth2.Token{AccessToken: "A", RefreshToken: "R"}); err != nil {
		t.Fatalf("seed default: %v", err)
	}
	work, err := openWith(&config.Config{CredentialRef: "google-readonly/work"}, false, false)
	if err != nil {
		t.Fatalf("open work: %v", err)
	}
	if err := work.SetToken(&oauth2.Token{AccessToken: "B", RefreshToken: "S"}); err != nil {
		t.Fatalf("seed work: %v", err)
	}
	_ = work.Close()

	ps, err := st.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(ps) != 2 || ps[0] != "default" || ps[1] != "work" {
		t.Fatalf("ListProfiles = %v, want [default work] (sorted)", ps)
	}

	// Cross-profile presence from the default-bound store handle.
	if has, herr := st.HasTokenFor("work"); herr != nil || !has {
		t.Fatalf("HasTokenFor(work) = (%v, %v), want (true, nil)", has, herr)
	}
	if has, herr := st.HasTokenFor("absent"); herr != nil || has {
		t.Fatalf("HasTokenFor(absent) = (%v, %v), want (false, nil)", has, herr)
	}
}

func TestProfileCopyDeleteMovesBundleWithoutReauth(t *testing.T) {
	credtest.Setup(t)
	st, err := openWith(testCfg(), false, false)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = st.Close() }()
	if err := st.SetToken(&oauth2.Token{AccessToken: "A", RefreshToken: "R"}); err != nil {
		t.Fatal(err)
	}

	if err := st.CopyProfile("default", "work"); err != nil {
		t.Fatalf("CopyProfile: %v", err)
	}
	if err := st.DeleteProfile("default"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
	if _, err := st.Token(); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("source token after rename = %v, want not found", err)
	}
	work, err := openWith(&config.Config{CredentialRef: "google-readonly/work"}, false, false)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer func() { _ = work.Close() }()
	tok, err := work.Token()
	if err != nil || tok.AccessToken != "A" || tok.RefreshToken != "R" {
		t.Fatalf("destination token = %+v, err=%v", tok, err)
	}
}

func TestCopyProfileCollisionRetainsBothBundles(t *testing.T) {
	credtest.Setup(t)
	st, err := openWith(testCfg(), false, false)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = st.Close() }()
	if err := st.SetToken(&oauth2.Token{AccessToken: "OLD", RefreshToken: "R"}); err != nil {
		t.Fatal(err)
	}
	work, err := openWith(&config.Config{CredentialRef: "google-readonly/work"}, false, false)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	if err := work.SetToken(&oauth2.Token{AccessToken: "NEW", RefreshToken: "R"}); err != nil {
		t.Fatal(err)
	}
	_ = work.Close()

	err = st.CopyProfile("default", "work")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CopyProfile collision = %v, want occupied-destination error", err)
	}
	old, err := st.Token()
	if err != nil || old.AccessToken != "OLD" {
		t.Fatalf("source token after collision = %+v, err=%v", old, err)
	}
	work, err = openWith(&config.Config{CredentialRef: "google-readonly/work"}, false, false)
	if err != nil {
		t.Fatalf("reopen destination: %v", err)
	}
	defer func() { _ = work.Close() }()
	newTok, err := work.Token()
	if err != nil || newTok.AccessToken != "NEW" {
		t.Fatalf("destination token after collision = %+v, err=%v", newTok, err)
	}
}

func TestProfileCopyDeleteDoesNotCrossServiceNamespace(t *testing.T) {
	credtest.Setup(t)
	readonly, err := openWith(testCfg(), false, false)
	if err != nil {
		t.Fatalf("open readonly: %v", err)
	}
	defer func() { _ = readonly.Close() }()
	if err := readonly.SetToken(&oauth2.Token{AccessToken: "READONLY", RefreshToken: "R"}); err != nil {
		t.Fatal(err)
	}

	service := "google-readwrite"
	t.Setenv(credstore.BackendEnvVar(service), "file")
	t.Setenv(strings.TrimSuffix(credstore.BackendEnvVar(service), "_KEYRING_BACKEND")+"_KEYRING_PASSPHRASE", "test-passphrase")
	readwrite, err := openWith(&config.Config{CredentialRef: service + "/default"}, false, false)
	if err != nil {
		t.Fatalf("open readwrite: %v", err)
	}
	defer func() { _ = readwrite.Close() }()
	if err := readwrite.SetToken(&oauth2.Token{AccessToken: "READWRITE", RefreshToken: "R"}); err != nil {
		t.Fatal(err)
	}

	if err := readonly.CopyProfile("default", "renamed"); err != nil {
		t.Fatalf("copy readonly: %v", err)
	}
	if err := readonly.DeleteProfile("default"); err != nil {
		t.Fatalf("delete readonly: %v", err)
	}
	got, err := readwrite.Token()
	if err != nil || got.AccessToken != "READWRITE" {
		t.Fatalf("readwrite token after readonly rename = %+v, err=%v", got, err)
	}
}
