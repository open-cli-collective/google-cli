package config

import (
	"os"
	"path/filepath"

	"github.com/open-cli-collective/cli-common/statedir"
)

// Identity carries the per-CLI values that were previously compile-time
// constants in google-readonly (DirName, DefaultCredentialRef) plus the OAuth
// scope set and user-facing product name. It is the single seam that lets one
// shared library back multiple Google CLIs (e.g. gro, grw): each CLI process
// registers its identity exactly once at startup — from main, before any other
// config/keychain/auth call — so the library resolves the correct config/cache
// dir, keyring service, env-var prefixes, requested scopes, and message wording
// for that CLI.
//
// DirName drives everything downstream: the native per-OS config dir and cache
// dir (via cli-common/statedir), the keyring service segment, and the derived
// env-var prefixes (<SERVICE>_KEYRING_BACKEND, <SERVICE>_KEYRING_PASSPHRASE,
// <SERVICE>_CREDENTIAL_REF). DefaultRef is the fallback <service>/<profile>
// credential ref when config.yml omits credential_ref.
type Identity struct {
	// DirName is the config/cache dir name, keyring service segment, and
	// env-var prefix source (e.g. "google-readonly").
	DirName string
	// DefaultRef is the default <service>/<profile> credential ref applied
	// when config.yml is absent or omits credential_ref (e.g.
	// "google-readonly/default"). Callers still resolve it via
	// credstore.ParseRef — the structure is never assumed (§1.3).
	DefaultRef string
	// ProductName is the user-facing command/product name spliced into
	// messages (e.g. "gro" -> "run 'gro init'").
	ProductName string
	// Scopes is the OAuth scope set requested at init and validated for drift.
	Scopes []string
	// ScopeDescriptions maps scope URLs to human-friendly descriptions used by
	// the init wizard and the scope-drift re-auth prompt.
	ScopeDescriptions map[string]string
	// SiblingDirNames lists the DirNames of sibling CLIs whose OAuth client
	// JSON (deployment material, not a secret) this CLI may reuse during init,
	// so a user who has already set up a sibling need not paste it again. E.g.
	// grw lists "google-readonly" so `grw init` can adopt gro's OAuth client.
	SiblingDirNames []string
}

// DirName is the config/cache directory name and keyring service segment for
// the registered CLI. Populated by Register; empty until then.
var DirName string

// DefaultCredentialRef is the default <service>/<profile> credential ref for
// the registered CLI. Populated by Register; empty until then.
var DefaultCredentialRef string

var (
	productName       string
	scopes            []string
	scopeDescriptions map[string]string
	siblingDirNames   []string
)

// Register stores the process-wide CLI identity. Call it exactly once from
// main, before any other config/keychain/auth call. It panics if DirName or
// DefaultRef is empty, because every downstream path (keyring service, config
// dir, env-var names) derives from them and a silent empty value would resolve
// to the wrong — possibly shared — location.
func Register(id Identity) {
	if id.DirName == "" || id.DefaultRef == "" {
		panic("config.Register: DirName and DefaultRef are required")
	}
	DirName = id.DirName
	DefaultCredentialRef = id.DefaultRef
	productName = id.ProductName
	scopes = append([]string(nil), id.Scopes...)
	scopeDescriptions = id.ScopeDescriptions
	siblingDirNames = append([]string(nil), id.SiblingDirNames...)
}

// SiblingOAuthClientPath returns the path to an existing OAuth client JSON
// belonging to a registered sibling CLI (deployment material, not a secret),
// so init can reuse it instead of asking the user to paste it again. It returns
// the first sibling whose oauth_client.json exists, along with that sibling's
// DirName. ok is false when no sibling has one.
func SiblingOAuthClientPath() (path, siblingDirName string, ok bool) {
	for _, name := range siblingDirNames {
		dir, err := (statedir.Scope{Name: name}).ConfigDir()
		if err != nil {
			continue
		}
		p := filepath.Join(dir, OAuthClientFile)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, name, true
		}
	}
	return "", "", false
}

// RegisterForTest registers a canonical identity for the shared library's own
// tests (and for any consumer test helper that needs a valid identity without
// caring about the specific CLI). It mirrors the historical google-readonly
// values so identity-specific assertions ported from that CLI keep passing. It
// is exported (not a _test.go helper) so sibling test packages such as
// credtest can call it without an import cycle; production CLIs never call it —
// they call Register with their own Identity from main.
func RegisterForTest() {
	Register(Identity{
		DirName:     "google-readonly",
		DefaultRef:  "google-readonly/default",
		ProductName: "gro",
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/calendar.readonly",
			"https://www.googleapis.com/auth/calendar.events",
			"https://www.googleapis.com/auth/contacts",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/drive.readonly",
			"https://www.googleapis.com/auth/drive.metadata",
		},
		ScopeDescriptions: map[string]string{
			"https://www.googleapis.com/auth/gmail.modify":      "Gmail Modify — read messages, plus label, archive, star, and mark read/unread.",
			"https://www.googleapis.com/auth/calendar.readonly": "Calendar Read-Only — read calendars and events.",
			"https://www.googleapis.com/auth/calendar.events":   "Calendar Events — read and update events (RSVP, color).",
			"https://www.googleapis.com/auth/contacts":          "Contacts — read contacts and groups, plus manage group membership and starring.",
			"https://www.googleapis.com/auth/userinfo.profile":  "Profile — read the authenticated user's name and email address.",
			"https://www.googleapis.com/auth/drive.readonly":    "Drive Read-Only — read files and metadata.",
			"https://www.googleapis.com/auth/drive.metadata":    "Drive Metadata — read and update file metadata (star/unstar).",
		},
	})
}

// ProductName returns the registered user-facing product/command name.
func ProductName() string { return productName }

// Scopes returns a copy of the registered OAuth scope set.
func Scopes() []string { return append([]string(nil), scopes...) }

// ScopeDescriptions returns the registered scope-description map (may be nil).
func ScopeDescriptions() map[string]string { return scopeDescriptions }
