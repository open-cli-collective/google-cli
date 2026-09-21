package setcred

import (
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-cli/internal/credtest"
	"github.com/open-cli-collective/google-cli/internal/keychain"
)

const tokenJSON = `{"access_token":"SECRET-ACCESS","refresh_token":"SECRET-REFRESH","token_type":"Bearer"}`

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
