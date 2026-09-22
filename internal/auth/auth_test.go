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

func clientJSON(clientID, tokenURI string) string {
	return fmt.Sprintf(`{"installed":{"client_id":%q,"project_id":"p","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":%q,"client_secret":"secret","redirect_uris":["http://localhost"]}}`, clientID, tokenURI)
}

func TestGetOAuthConfigForRefUsesExactProfileClient(t *testing.T) {
	credtest.Setup(t)
	dir := credtest.ConfigDir(t)
	defaultPath := filepath.Join(dir, "default-client.json")
	workPath := filepath.Join(dir, "work-client.json")
	if err := os.WriteFile(defaultPath, []byte(clientJSON("default-client", "https://oauth2.googleapis.com/token")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workPath, []byte(clientJSON("work-client", "https://oauth2.googleapis.com/token")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveConfig(&config.Config{
		CredentialRef: config.DefaultCredentialRef,
		Profiles: map[string]config.ProfileConfig{
			"default": {OAuthClientPath: defaultPath, GrantedScopes: []string{"default-scope"}},
			"work":    {OAuthClientPath: workPath, GrantedScopes: []string{"work-scope"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := GetOAuthConfigForRef("google-readonly/work")
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientID != "work-client" {
		t.Fatalf("work client ID = %q, want work-client", got.ClientID)
	}
	if _, err := GetOAuthConfigForRef("google-readonly/other"); err == nil || !strings.Contains(err.Error(), "no OAuth client configured") {
		t.Fatalf("unconfigured ref result = %v, want actionable miss", err)
	}
}

func TestGetHTTPClientForRefUsesMatchingOAuthClientOnRefresh(t *testing.T) {
	credtest.Setup(t)
	const ref = "google-readonly/work"
	seenClientID := make(chan string, 1)
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
			seenClientID <- clientID
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"work-access","token_type":"Bearer","expires_in":3600}`)
		case "/api":
			if r.Header.Get("Authorization") != "Bearer work-access" {
				http.Error(w, "wrong token", http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := credtest.ConfigDir(t)
	defaultPath := filepath.Join(dir, "default-client.json")
	workPath := filepath.Join(dir, "work-client.json")
	if err := os.WriteFile(defaultPath, []byte(clientJSON("default-client", server.URL+"/token")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workPath, []byte(clientJSON("work-client", server.URL+"/token")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveConfig(&config.Config{
		CredentialRef: config.DefaultCredentialRef,
		Profiles: map[string]config.ProfileConfig{
			"default": {OAuthClientPath: defaultPath},
			"work":    {OAuthClientPath: workPath},
		},
	}); err != nil {
		t.Fatal(err)
	}
	st, err := keychain.OpenRef(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetToken(&oauth2.Token{AccessToken: "expired", RefreshToken: "refresh", Expiry: time.Now().Add(-time.Hour)}); err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	httpClient, err := GetHTTPClientForRef(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/api", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("API status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
	select {
	case got := <-seenClientID:
		if got != "work-client" {
			t.Fatalf("refresh client ID = %q, want work-client", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("refresh endpoint was not called")
	}
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
