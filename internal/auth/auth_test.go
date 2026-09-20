package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/credtest"
	"github.com/open-cli-collective/google-cli/internal/keychain"
)

func TestGetOAuthConfigForRefSelectsProfileClient(t *testing.T) {
	credtest.Setup(t)
	defaultPath := filepath.Join(credtest.ConfigDir(t), "default.json")
	profilePath := filepath.Join(credtest.ConfigDir(t), "profile.json")
	if err := os.WriteFile(defaultPath, []byte(testOAuthClientJSON("default-client", "https://oauth2.googleapis.com/token")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profilePath, []byte(testOAuthClientJSON("profile-client", "https://oauth2.googleapis.com/token")), 0600); err != nil {
		t.Fatal(err)
	}
	const ref = "google-readonly/personal"
	if err := config.SaveConfig(&config.Config{
		CredentialRef:   config.DefaultCredentialRef,
		OAuthClientPath: defaultPath,
		ProfileOAuth: map[string]config.ProfileOAuthConfig{
			ref: {OAuthClientPath: profilePath},
		},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := GetOAuthConfigForRef(ref)
	if err != nil {
		t.Fatalf("GetOAuthConfigForRef: %v", err)
	}
	if got.ClientID != "profile-client" {
		t.Fatalf("profile ClientID = %q, want profile-client", got.ClientID)
	}
	legacy, err := GetOAuthConfigForRef(config.DefaultCredentialRef)
	if err != nil {
		t.Fatalf("GetOAuthConfigForRef(default): %v", err)
	}
	if legacy.ClientID != "default-client" {
		t.Fatalf("legacy ClientID = %q, want default-client", legacy.ClientID)
	}
}

func TestGetHTTPClientForRefRefreshesWithMatchingOAuthClient(t *testing.T) {
	credtest.Setup(t)
	keychain.SetCredentialRefOverride("", false)
	t.Cleanup(func() { keychain.SetCredentialRefOverride("", false) })
	const profileRef = "google-readonly/personal"

	refreshClientID := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad form", http.StatusBadRequest)
				return
			}
			clientID := r.Form.Get("client_id")
			if clientID == "" {
				clientID, _, _ = r.BasicAuth()
			}
			refreshClientID <- clientID
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"refreshed-profile-token","token_type":"Bearer","expires_in":3600}`)
		case "/api":
			if got := r.Header.Get("Authorization"); got != "Bearer refreshed-profile-token" {
				http.Error(w, "wrong bearer token", http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	configDir := credtest.ConfigDir(t)
	defaultPath := filepath.Join(configDir, "default.json")
	profilePath := filepath.Join(configDir, "profile.json")
	if err := os.WriteFile(defaultPath, []byte(testOAuthClientJSON("default-client", server.URL+"/token")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profilePath, []byte(testOAuthClientJSON("profile-client", server.URL+"/token")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveConfig(&config.Config{
		CredentialRef:   config.DefaultCredentialRef,
		OAuthClientPath: defaultPath,
		ProfileOAuth: map[string]config.ProfileOAuthConfig{
			profileRef: {OAuthClientPath: profilePath},
		},
	}); err != nil {
		t.Fatal(err)
	}

	seed := func(ref string, token *oauth2.Token) {
		t.Helper()
		st, err := keychain.OpenRef(ref)
		if err != nil {
			t.Fatalf("OpenRef(%s): %v", ref, err)
		}
		if err := st.SetToken(token); err != nil {
			_ = st.Close()
			t.Fatalf("SetToken(%s): %v", ref, err)
		}
		if err := st.Close(); err != nil {
			t.Fatalf("Close(%s): %v", ref, err)
		}
	}
	seed(config.DefaultCredentialRef, &oauth2.Token{AccessToken: "unchanged-default", RefreshToken: "default-refresh", TokenType: "Bearer"})
	seed(profileRef, &oauth2.Token{AccessToken: "expired-profile", RefreshToken: "profile-refresh", TokenType: "Bearer", Expiry: time.Now().Add(-time.Hour)})

	client, err := GetHTTPClientForRef(context.Background(), profileRef)
	if err != nil {
		t.Fatalf("GetHTTPClientForRef: %v", err)
	}
	resp, err := client.Get(server.URL + "/api")
	if err != nil {
		t.Fatalf("GET api: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("GET api status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	select {
	case got := <-refreshClientID:
		if got != "profile-client" {
			t.Fatalf("refresh client_id = %q, want profile-client", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("profile token was not refreshed")
	}

	assertToken := func(ref, want string) {
		t.Helper()
		st, err := keychain.OpenRef(ref)
		if err != nil {
			t.Fatal(err)
		}
		tok, err := st.Token()
		_ = st.Close()
		if err != nil || tok.AccessToken != want {
			t.Fatalf("token at %s = (%q, %v), want %q", ref, tok.AccessToken, err, want)
		}
	}
	assertToken(profileRef, "refreshed-profile-token")
	assertToken(config.DefaultCredentialRef, "unchanged-default")
}

func testOAuthClientJSON(clientID, tokenURL string) string {
	return fmt.Sprintf(`{"installed":{"client_id":%q,"project_id":"test","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":%q,"auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"test-secret","redirect_uris":["http://localhost"]}}`, clientID, tokenURL)
}

// TestDeprecatedWrappers verifies that auth package wrappers delegate to config package
func TestDeprecatedWrappers(t *testing.T) {
	t.Run("GetConfigDir delegates to config package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		authDir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configDir, err := config.GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if authDir != configDir {
			t.Errorf("got %v, want %v", authDir, configDir)
		}
	})

	t.Run("GetCredentialsPath delegates to config package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		authPath, err := GetCredentialsPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath, err := config.GetCredentialsPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if authPath != configPath {
			t.Errorf("got %v, want %v", authPath, configPath)
		}
	})

	t.Run("GetTokenPath delegates to config package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		authPath, err := GetTokenPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath, err := config.GetTokenPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if authPath != configPath {
			t.Errorf("got %v, want %v", authPath, configPath)
		}
	})

	t.Run("ShortenPath delegates to config package", func(t *testing.T) {
		t.Parallel()
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		testPath := filepath.Join(home, ".config", "test")

		authResult := ShortenPath(testPath)
		configResult := config.ShortenPath(testPath)

		if authResult != configResult {
			t.Errorf("got %v, want %v", authResult, configResult)
		}
	})

	t.Run("Constants match config package", func(t *testing.T) {
		t.Parallel()
		if CredentialsFile != config.CredentialsFile {
			t.Errorf("got %v, want %v", CredentialsFile, config.CredentialsFile)
		}
		if TokenFile != config.TokenFile {
			t.Errorf("got %v, want %v", TokenFile, config.TokenFile)
		}
	})
}

// TestRegisteredScopesNonEmpty is a light guard that the test identity's scope
// set is wired through config. Per-CLI scope-content assertions (exact count,
// specific scopes) live in each CLI's own repo, since the scope set is now the
// CLI's responsibility via config.Register.
func TestRegisteredScopesNonEmpty(t *testing.T) {
	t.Parallel()
	if len(config.Scopes()) == 0 {
		t.Fatal("expected the registered test identity to expose scopes")
	}
	if !strings.Contains(strings.Join(config.Scopes(), " "), "https://www.googleapis.com/auth/gmail.modify") {
		t.Errorf("expected registered scopes to contain gmail.modify")
	}
}

func TestCheckScopesMigration_NoGrantedScopes(t *testing.T) {
	t.Parallel()
	msg := CheckScopesMigration(nil)
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
}

func TestCheckScopesMigration_AllGranted(t *testing.T) {
	t.Parallel()
	msg := CheckScopesMigration(config.Scopes())
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
}

func TestCheckScopesMigration_MissingScope(t *testing.T) {
	t.Parallel()
	oldScopes := []string{
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/calendar.readonly",
		"https://www.googleapis.com/auth/contacts.readonly",
		"https://www.googleapis.com/auth/drive.readonly",
	}
	msg := CheckScopesMigration(oldScopes)
	if msg == "" {
		t.Fatal("expected non-empty message")
	}
	if !strings.Contains(msg, "gro init") {
		t.Errorf("expected message to mention 'gro init', got %q", msg)
	}
	if !strings.Contains(msg, "Gmail Modify") {
		t.Errorf("expected message to mention 'Gmail Modify', got %q", msg)
	}
	if !strings.Contains(msg, "Contacts") {
		t.Errorf("expected message to mention 'Contacts', got %q", msg)
	}
}

// TestTokenFromFile was removed: the plaintext token.json fallback no longer
// exists. The token now lives only in the OS keyring via credstore (§1.1 /
// §2.3); legacy token.json is handled one-time by internal/keychain's
// migration and covered by that package's tests.
