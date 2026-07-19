// Package auth provides OAuth2 authentication and credential management for Google APIs.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/open-cli-collective/google-cli-common/config"
	"github.com/open-cli-collective/google-cli-common/keychain"
)

// CheckScopesMigration compares the registered CLI's currently-required scopes
// (config.Scopes(), set by the CLI via config.Register) against the scopes a
// token was previously granted. It returns a non-empty, actionable message
// when the token is missing any now-required scope, so a CLI that has widened
// its scope set can prompt the user to re-authenticate. The scope set,
// descriptions, and product name all come from the registered identity, so the
// same logic serves any CLI backed by this library.
func CheckScopesMigration(grantedScopes []string) string {
	if len(grantedScopes) == 0 {
		return ""
	}

	granted := make(map[string]bool, len(grantedScopes))
	for _, s := range grantedScopes {
		granted[s] = true
	}

	var missing []string
	for _, required := range config.Scopes() {
		if !granted[required] {
			missing = append(missing, required)
		}
	}

	if len(missing) == 0 {
		return ""
	}

	descriptions := config.ScopeDescriptions()
	msg := fmt.Sprintf("This command requires additional permissions.\nYour token is missing one or more required scopes.\n\nRun '%s init' to re-authenticate with the updated scopes.\n\nNew scopes:\n", config.ProductName())
	for _, s := range missing {
		desc := descriptions[s]
		if desc == "" {
			desc = s
		}
		msg += "  - " + desc + "\n"
	}
	return msg
}

// GetOAuthConfig loads the OAuth client config from the deployment-material
// OAuth client JSON referenced by config.yml's oauth_client_path (§1.2 — not
// a secret; lives on disk, never the keyring), with all scopes.
func GetOAuthConfig() (*oauth2.Config, error) {
	cfg, err := config.LoadConfigForRuntime()
	if err != nil {
		return nil, err
	}
	path := config.ExpandPath(cfg.OAuthClientPath)
	b, err := os.ReadFile(path) //nolint:gosec // deployment-material path from config
	if err != nil {
		return nil, fmt.Errorf("unable to read OAuth client JSON %s (run '%s init'): %w",
			config.ShortenPath(path), config.ProductName(), err)
	}
	return google.ConfigFromJSON(b, config.Scopes()...)
}

// GetHTTPClient returns an HTTP client with OAuth2 authentication. The token
// is read solely from the OS keyring via credstore (§1.1/§2.3 — no
// security/secret-tool shell-out, no token.json fallback). The active
// credential_ref is captured once here; refreshed tokens persist back to that
// exact ref via the closure passed to the token source (the sole sanctioned
// non-ingress keyring write). Returns an actionable error if no token exists.
func GetHTTPClient(ctx context.Context) (*http.Client, error) {
	oauthCfg, err := GetOAuthConfig()
	if err != nil {
		return nil, err
	}

	st, err := keychain.Open()
	if err != nil {
		return nil, err
	}
	tok, err := st.Token()
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("no OAuth token found - please run '%s init' first: %w", config.ProductName(), err)
	}
	ref := st.Ref()
	_ = st.Close() // do not hold the Store for the client's lifetime

	persist := func(t *oauth2.Token) error {
		ps, perr := keychain.OpenRef(ref) // runMigration=false: refresh is not ingress
		if perr != nil {
			return perr
		}
		defer func() { _ = ps.Close() }()
		return ps.SetToken(t)
	}

	tokenSource := keychain.NewPersistentTokenSource(ctx, oauthCfg, tok, persist)
	return oauth2.NewClient(ctx, tokenSource), nil
}

// GetAuthURL returns the OAuth authorization URL for the given config
func GetAuthURL(config *oauth2.Config) string {
	return config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

// ExchangeAuthCode exchanges an authorization code for a token
func ExchangeAuthCode(ctx context.Context, config *oauth2.Config, code string) (*oauth2.Token, error) {
	return config.Exchange(ctx, code)
}
