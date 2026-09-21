package setcred

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/cli-common/credstore"
	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/credtest"
	"github.com/open-cli-collective/google-cli/internal/keychain"
)

const tokenJSON = `{"access_token":"SECRET-ACCESS","refresh_token":"SECRET-REFRESH","token_type":"Bearer"}`

func writeLegacyToken(t *testing.T, value string) string {
	t.Helper()
	path, err := config.GetTokenPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func saveCredentialRef(t *testing.T, ref string) {
	t.Helper()
	if err := config.SaveConfig(&config.Config{CredentialRef: ref}); err != nil {
		t.Fatal(err)
	}
}

func seedTokenAtRef(t *testing.T, ref, access string) {
	t.Helper()
	st, err := keychain.OpenRef(ref)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if err := st.SetToken(&oauth2.Token{AccessToken: access, RefreshToken: "seed-refresh"}); err != nil {
		t.Fatal(err)
	}
}

func tokenAtRef(t *testing.T, ref string) (*oauth2.Token, error) {
	t.Helper()
	st, err := keychain.OpenRef(ref)
	if err != nil {
		return nil, err
	}
	defer func() { _ = st.Close() }()
	return st.Token()
}

func TestSetCredentialStdin(t *testing.T) {
	credtest.Setup(t)
	err := run(&options{key: keychain.KeyOAuthToken, stdin: true, in: strings.NewReader(tokenJSON)})
	if err != nil {
		t.Fatalf("set-credential --stdin: %v", err)
	}
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	tok, err := st.Token()
	if err != nil || tok.AccessToken != "SECRET-ACCESS" {
		t.Fatalf("token not stored: %+v err=%v", tok, err)
	}
}

func TestSetCredentialUsesInheritedProfile(t *testing.T) {
	credtest.Setup(t)
	defaultStore, err := keychain.OpenNoMigrate()
	if err != nil {
		t.Fatal(err)
	}
	if err := defaultStore.SetToken(&oauth2.Token{AccessToken: "DEFAULT-ACCESS", RefreshToken: "DEFAULT-REFRESH"}); err != nil {
		_ = defaultStore.Close()
		t.Fatal(err)
	}
	_ = defaultStore.Close()

	keychain.SetCredentialRefOverride("google-readonly/work", true)
	t.Cleanup(func() { keychain.SetCredentialRefOverride("", false) })
	if err := run(&options{key: keychain.KeyOAuthToken, stdin: true, in: strings.NewReader(tokenJSON)}); err != nil {
		t.Fatalf("set-credential --profile work: %v", err)
	}

	keychain.SetCredentialRefOverride("", false)
	defaultStore, err = keychain.OpenNoMigrate()
	if err != nil {
		t.Fatal(err)
	}
	defaultToken, err := defaultStore.Token()
	_ = defaultStore.Close()
	if err != nil || defaultToken.AccessToken != "DEFAULT-ACCESS" || defaultToken.RefreshToken != "DEFAULT-REFRESH" {
		t.Fatalf("configured profile token changed: %+v err=%v", defaultToken, err)
	}

	keychain.SetCredentialRefOverride("google-readonly/work", true)
	workStore, err := keychain.OpenNoMigrate()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = workStore.Close() }()
	workToken, err := workStore.Token()
	if got := workStore.Ref(); got != "google-readonly/work" {
		t.Fatalf("stored credential ref = %q, want google-readonly/work", got)
	}
	if err != nil || workToken.AccessToken != "SECRET-ACCESS" || workToken.RefreshToken != "SECRET-REFRESH" {
		t.Fatalf("selected profile token missing: %+v err=%v", workToken, err)
	}
}

func TestSetCredentialKeyAllowlist(t *testing.T) {
	credtest.Setup(t)
	err := run(&options{key: "not_allowed", stdin: true, in: strings.NewReader(tokenJSON)})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("want allowlist error, got %v", err)
	}
}

func TestSetCredentialFromEnvEmptyNamesVar(t *testing.T) {
	credtest.Setup(t)
	err := run(&options{key: keychain.KeyOAuthToken, fromEnv: "GRO_TEST_UNSET_VAR"})
	if err == nil || !strings.Contains(err.Error(), "GRO_TEST_UNSET_VAR") {
		t.Fatalf("error must name the empty env var, got %v", err)
	}
}

func TestSetCredentialFromEnvSuccess(t *testing.T) {
	credtest.Setup(t)
	t.Setenv("GRO_TEST_TOKEN", tokenJSON)
	if err := run(&options{key: keychain.KeyOAuthToken, fromEnv: "GRO_TEST_TOKEN"}); err != nil {
		t.Fatalf("set-credential --from-env: %v", err)
	}
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	tok, err := st.Token()
	if err != nil || tok.RefreshToken != "SECRET-REFRESH" {
		t.Fatalf("token not stored from env: %+v err=%v", tok, err)
	}
}

func TestSetCredentialRejectsNonToken(t *testing.T) {
	credtest.Setup(t)
	err := run(&options{key: keychain.KeyOAuthToken, stdin: true, in: strings.NewReader(`{"not":"a token"}`)})
	if err == nil || !strings.Contains(err.Error(), "neither an access nor a refresh token") {
		t.Fatalf("want token-shape rejection, got %v", err)
	}
}

// TestSetCredentialMigrationConflictBlocksWrite proves a legacy/configured
// disagreement aborts before set-credential can overwrite the keyring value.
func TestSetCredentialMigrationConflictBlocksWrite(t *testing.T) {
	credtest.Setup(t)
	const ref = "google-readonly/work"
	saveCredentialRef(t, ref)
	seedTokenAtRef(t, ref, "KEYRING")
	legacyPath := writeLegacyToken(t, `{"access_token":"LEGACY","refresh_token":"LEGACY-REFRESH"}`)

	err := run(&options{key: keychain.KeyOAuthToken, stdin: true, in: strings.NewReader(tokenJSON)})
	if !errors.Is(err, credstore.ErrMigrationConflict) {
		t.Fatalf("want migration conflict, got %v", err)
	}
	if _, statErr := os.Stat(legacyPath); statErr != nil {
		t.Fatalf("legacy token must remain after conflict: %v", statErr)
	}
	tok, readErr := tokenAtRef(t, ref)
	if readErr != nil || tok.AccessToken != "KEYRING" {
		t.Fatalf("configured target changed after conflict: %+v, err=%v", tok, readErr)
	}
}

// TestSetCredentialExplicitProfileDoesNotMigrateDefaultLegacy proves an
// explicit --profile target is isolated from the configured/default migration:
// it writes only the selected profile and leaves the default legacy artifact
// for a later default-target invocation.
func TestSetCredentialExplicitProfileDoesNotMigrateDefaultLegacy(t *testing.T) {
	credtest.Setup(t)
	legacyPath := writeLegacyToken(t, `{"access_token":"LEGACY","refresh_token":"LEGACY-REFRESH"}`)
	keychain.SetCredentialRefOverride("google-readonly/work", true)
	t.Cleanup(func() { keychain.SetCredentialRefOverride("", false) })

	if err := run(&options{key: keychain.KeyOAuthToken, stdin: true, in: strings.NewReader(tokenJSON)}); err != nil {
		t.Fatalf("set-credential --profile work: %v", err)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("explicit profile must not consume default legacy token: %v", err)
	}
	selected, err := tokenAtRef(t, "google-readonly/work")
	if err != nil || selected.AccessToken != "SECRET-ACCESS" {
		t.Fatalf("selected profile token = %+v, err=%v", selected, err)
	}
	if _, err := tokenAtRef(t, config.DefaultCredentialRef); !errors.Is(err, keychain.ErrTokenNotFound) {
		t.Fatalf("default profile should remain unmigrated, got err=%v", err)
	}
}
