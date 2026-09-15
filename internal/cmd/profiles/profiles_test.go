package profiles

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/credtest"
	"github.com/open-cli-collective/google-cli/internal/identitycache"
	"github.com/open-cli-collective/google-cli/internal/keychain"
)

func TestMain(m *testing.M) {
	config.RegisterForTest()
	os.Exit(m.Run())
}

// capture redirects os.Stdout for f (the command prints with fmt.Printf).
func capture(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() { var b bytes.Buffer; _, _ = io.Copy(&b, r); done <- b.String() }()
	func() {
		defer func() {
			os.Stdout = orig
			_ = w.Close()
		}()
		f()
	}()
	return <-done
}

func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	done := make(chan string, 1)
	go func() { var b bytes.Buffer; _, _ = io.Copy(&b, r); done <- b.String() }()
	func() {
		defer func() {
			os.Stderr = orig
			_ = w.Close()
		}()
		f()
	}()
	return <-done
}

// seedToken stores a token under the given profile of the test service.
func seedToken(t *testing.T, profile string) {
	t.Helper()
	st, err := keychain.OpenRef("google-readonly/" + profile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if err := st.SetToken(&oauth2.Token{AccessToken: "A-" + profile, RefreshToken: "R"}); err != nil {
		t.Fatal(err)
	}
}

func withVerify(t *testing.T, f func(ctx context.Context, ref string) (string, error)) {
	t.Helper()
	orig := VerifyRef
	VerifyRef = f
	t.Cleanup(func() { VerifyRef = orig })
}

func TestNewCommandSurface(t *testing.T) {
	cmd := NewCommand()
	if cmd.Use != "profiles" {
		t.Errorf("Use = %q, want profiles", cmd.Use)
	}
	var names []string
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	for _, want := range []string{"list", "use", "rename"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("missing subcommand %q (have %v)", want, names)
		}
	}
}

func TestListFlags(t *testing.T) {
	cmd := newListCommand()
	if f := cmd.Flags().Lookup("json"); f == nil || f.Shorthand != "j" {
		t.Errorf("expected --json/-j flag, got %+v", f)
	}
	if f := cmd.Flags().Lookup("check"); f == nil {
		t.Error("expected --check flag")
	}
}

// TestRunList_ListsStoredProfiles is the headline behavior: every stored
// profile visible in one command, active one marked, no keychain dumping.
func TestRunList_ListsStoredProfiles(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "default")
	seedToken(t, "work")

	out := capture(t, func() {
		if err := runList(context.Background(), false, false); err != nil {
			t.Errorf("runList: %v", err)
		}
	})

	for _, want := range []string{"default", "work", "present", "Active: google-readonly/default", "profiles use"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	// The active row carries the '*' marker.
	activeMarked := false
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "*") && strings.Contains(line, "default") {
			activeMarked = true
		}
	}
	if !activeMarked {
		t.Errorf("active profile not marked with '*':\n%s", out)
	}
}

// TestRunList_FreshInstall shows the active (default) profile even before any
// token exists, with the init hint — a fresh user must not see an empty
// table with no way forward.
func TestRunList_FreshInstall(t *testing.T) {
	credtest.Setup(t)

	out := capture(t, func() {
		if err := runList(context.Background(), false, false); err != nil {
			t.Errorf("runList: %v", err)
		}
	})

	for _, want := range []string{"default", "missing", "init"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// TestRunList_ShowsCachedEmail proves the resolved-account column comes from
// the identity cache without any API call.
func TestRunList_ShowsCachedEmail(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "default")
	if err := identitycache.Put("default", "user@example.com"); err != nil {
		t.Fatal(err)
	}

	out := capture(t, func() {
		if err := runList(context.Background(), false, false); err != nil {
			t.Errorf("runList: %v", err)
		}
	})
	if !strings.Contains(out, "user@example.com") {
		t.Errorf("output missing cached email:\n%s", out)
	}
}

func TestRunList_JSON(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "default")

	out := capture(t, func() {
		if err := runList(context.Background(), true, false); err != nil {
			t.Errorf("runList: %v", err)
		}
	})
	for _, want := range []string{`"profile": "default"`, `"ref": "google-readonly/default"`, `"active": true`, `"token_present": true`} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON missing %q:\n%s", want, out)
		}
	}
}

// TestRunList_Check is the incident scenario: one stale profile, one healthy.
// --check must say so per profile and cache the healthy one's email.
func TestRunList_Check(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "default")
	seedToken(t, "work")
	seedToken(t, "broken")

	withVerify(t, func(_ context.Context, ref string) (string, error) {
		switch ref {
		case "google-readonly/default":
			return "", &oauth2.RetrieveError{
				Response:         &http.Response{StatusCode: http.StatusBadRequest},
				ErrorCode:        "invalid_grant",
				ErrorDescription: "Token has been expired or revoked.",
			}
		case "google-readonly/work":
			return "work@example.com", nil
		default:
			return "", errors.New("dial tcp: connection refused\nsecond line noise")
		}
	})

	out := capture(t, func() {
		if err := runList(context.Background(), false, true); err != nil {
			t.Errorf("runList: %v", err)
		}
	})

	for _, want := range []string{"expired or revoked", "ok", "error: dial tcp: connection refused", "work@example.com"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "second line noise") {
		t.Errorf("multi-line error leaked past firstLine:\n%s", out)
	}
	// Healthy check must refresh the identity cache.
	if got := identitycache.Load()["work"].Email; got != "work@example.com" {
		t.Errorf("identity cache after check = %q, want work@example.com", got)
	}
}

func TestRunUse_SwitchesActiveProfile(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "default")
	seedToken(t, "work")

	out := capture(t, func() {
		if err := runUse("work"); err != nil {
			t.Errorf("runUse: %v", err)
		}
	})
	if !strings.Contains(out, "Active profile is now google-readonly/work.") {
		t.Errorf("missing confirmation:\n%s", out)
	}

	cfg, err := config.LoadConfigForRuntime()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CredentialRef != "google-readonly/work" {
		t.Errorf("credential_ref = %q, want google-readonly/work", cfg.CredentialRef)
	}
	// Switching must not touch the previous profile's token.
	st, err := keychain.OpenRef("google-readonly/default")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if has, _ := st.HasToken(); !has {
		t.Error("switching away removed the previous profile's token")
	}
}

// TestRunUse_MissingTokenWarnsButProceeds pins the add-second-account flow:
// `use work` then `init`. The warning names existing profiles so a typo is
// visible immediately.
func TestRunUse_MissingTokenWarnsButProceeds(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "default")

	out := capture(t, func() {
		if err := runUse("wrk"); err != nil {
			t.Errorf("runUse: %v", err)
		}
	})
	for _, want := range []string{"no token is stored for google-readonly/wrk", "Existing profiles: default", "init"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	cfg, _ := config.LoadConfigForRuntime()
	if cfg.CredentialRef != "google-readonly/wrk" {
		t.Errorf("credential_ref = %q, want google-readonly/wrk", cfg.CredentialRef)
	}
}

func TestRunUse_AlreadyActive(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "default")

	out := capture(t, func() {
		if err := runUse("default"); err != nil {
			t.Errorf("runUse: %v", err)
		}
	})
	if !strings.Contains(out, "already the active profile") {
		t.Errorf("expected already-active message:\n%s", out)
	}
}

func TestRunUse_FullRefSameService(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "work")

	if err := runUseQuiet(t, "google-readonly/work"); err != nil {
		t.Fatalf("runUse(full ref): %v", err)
	}
	cfg, _ := config.LoadConfigForRuntime()
	if cfg.CredentialRef != "google-readonly/work" {
		t.Errorf("credential_ref = %q, want google-readonly/work", cfg.CredentialRef)
	}
}

func TestRunUse_CrossServiceRefRejected(t *testing.T) {
	credtest.Setup(t)

	err := runUseQuiet(t, "google-readwrite/work")
	if err == nil || !strings.Contains(err.Error(), "google-readwrite") {
		t.Fatalf("expected cross-service rejection, got %v", err)
	}
}

func TestRunUse_InvalidProfileRejected(t *testing.T) {
	credtest.Setup(t)

	err := runUseQuiet(t, "bad.profile")
	if err == nil {
		t.Fatal("expected invalid-profile error")
	}
}

func TestRunRename_MovesTokenCacheAndImplicitActiveProfile(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "default")
	if err := identitycache.Put("default", "default@example.com"); err != nil {
		t.Fatal(err)
	}

	out := capture(t, func() {
		if err := runRename("default", "primary"); err != nil {
			t.Errorf("runRename: %v", err)
		}
	})
	for _, want := range []string{
		"Renamed profile google-readonly/default to google-readonly/primary.",
		"Active profile is now google-readonly/primary.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	cfg, err := config.LoadConfigForRuntime()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CredentialRef != "google-readonly/primary" {
		t.Fatalf("credential_ref = %q, want google-readonly/primary", cfg.CredentialRef)
	}
	old, err := keychain.OpenRef("google-readonly/default")
	if err != nil {
		t.Fatal(err)
	}
	oldHas, err := old.HasToken()
	_ = old.Close()
	if err != nil || oldHas {
		t.Fatalf("old token after rename = (%v, %v), want (false, nil)", oldHas, err)
	}
	newStore, err := keychain.OpenRef("google-readonly/primary")
	if err != nil {
		t.Fatal(err)
	}
	newTok, err := newStore.Token()
	_ = newStore.Close()
	if err != nil || newTok.AccessToken != "A-default" {
		t.Fatalf("new token after rename = %+v, err=%v", newTok, err)
	}
	cached := identitycache.Load()
	if _, ok := cached["default"]; ok {
		t.Fatal("old cached identity remains after rename")
	}
	if got := cached["primary"].Email; got != "default@example.com" {
		t.Fatalf("new cached identity = %q, want default@example.com", got)
	}
}

func TestRunRename_NonActivePreservesSavedConfig(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	clientPath := filepath.Join(t.TempDir(), "client.json")
	original := &config.Config{
		CredentialRef:   "google-readonly/current",
		OAuthClientPath: clientPath,
		GrantedScopes:   []string{"scope:mail", "scope:profile"},
		Keyring:         config.KeyringConfig{Backend: "file"},
	}
	if err := config.SaveConfig(original); err != nil {
		t.Fatal(err)
	}

	if err := runRenameQuiet(t, "old", "new"); err != nil {
		t.Fatalf("runRename: %v", err)
	}

	got, err := config.LoadConfigForRuntime()
	if err != nil {
		t.Fatal(err)
	}
	if got.CredentialRef != original.CredentialRef {
		t.Errorf("credential_ref after non-active rename = %q, want %q", got.CredentialRef, original.CredentialRef)
	}
	if got.OAuthClientPath != original.OAuthClientPath {
		t.Errorf("oauth_client_path after non-active rename = %q, want %q", got.OAuthClientPath, original.OAuthClientPath)
	}
	if len(got.GrantedScopes) != len(original.GrantedScopes) {
		t.Fatalf("granted_scopes after non-active rename = %v, want %v", got.GrantedScopes, original.GrantedScopes)
	}
	for i := range original.GrantedScopes {
		if got.GrantedScopes[i] != original.GrantedScopes[i] {
			t.Errorf("granted_scopes[%d] after non-active rename = %q, want %q", i, got.GrantedScopes[i], original.GrantedScopes[i])
		}
	}
	if got.Keyring.Backend != original.Keyring.Backend {
		t.Errorf("keyring backend after non-active rename = %q, want %q", got.Keyring.Backend, original.Keyring.Backend)
	}
	assertNoToken(t, "old")
	assertToken(t, "new", "A-old")
}

func TestRunRename_CollisionRetainsSourceAndDestination(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	seedToken(t, "new")

	err := runRenameQuiet(t, "old", "new")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("rename collision = %v, want occupied-destination error", err)
	}
	cfg, cfgErr := config.LoadConfigForRuntime()
	if cfgErr != nil {
		t.Fatal(cfgErr)
	}
	if cfg.CredentialRef != "google-readonly/default" {
		t.Fatalf("credential_ref after collision = %q, want default", cfg.CredentialRef)
	}
	for _, tc := range []struct {
		profile string
		access  string
	}{
		{profile: "old", access: "A-old"},
		{profile: "new", access: "A-new"},
	} {
		st, openErr := keychain.OpenRef("google-readonly/" + tc.profile)
		if openErr != nil {
			t.Fatal(openErr)
		}
		tok, tokErr := st.Token()
		_ = st.Close()
		if tokErr != nil || tok.AccessToken != tc.access {
			t.Errorf("%s token after collision = %+v, err=%v", tc.profile, tok, tokErr)
		}
	}
}

func TestRunRename_CopyFailureRetainsSource(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	original := renameCopy
	renameCopy = func(_ *keychain.Store, _, _ string) error { return errors.New("copy failed") }
	t.Cleanup(func() { renameCopy = original })

	err := runRenameQuiet(t, "old", "new")
	if err == nil || !strings.Contains(err.Error(), "copy failed") {
		t.Fatalf("copy failure = %v, want injected error", err)
	}
	assertToken(t, "old", "A-old")
	assertNoToken(t, "new")
}

func TestRunRename_ConfigFailureRollsBackCopy(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	if err := config.SaveConfig(&config.Config{CredentialRef: "google-readonly/old"}); err != nil {
		t.Fatal(err)
	}
	original := renameSaveConfig
	renameSaveConfig = func(*config.Config) error { return errors.New("config unavailable") }
	t.Cleanup(func() { renameSaveConfig = original })

	err := runRenameQuiet(t, "old", "new")
	if err == nil || !strings.Contains(err.Error(), "source was retained") {
		t.Fatalf("config failure = %v, want source-retained error", err)
	}
	assertToken(t, "old", "A-old")
	assertNoToken(t, "new")
	cfg, loadErr := config.LoadConfigForRuntime()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if cfg.CredentialRef != "google-readonly/old" {
		t.Fatalf("credential_ref after config failure = %q, want old", cfg.CredentialRef)
	}
}

func TestRunRename_ConfigFailureRollbackFailureRetainsBothBundles(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	if err := config.SaveConfig(&config.Config{CredentialRef: "google-readonly/old"}); err != nil {
		t.Fatal(err)
	}
	originalSave := renameSaveConfig
	renameSaveConfig = func(*config.Config) error { return errors.New("config unavailable") }
	originalDelete := renameDelete
	renameDelete = func(st *keychain.Store, profile string) error {
		if profile == "new" {
			return errors.New("rollback unavailable")
		}
		return originalDelete(st, profile)
	}
	t.Cleanup(func() {
		renameSaveConfig = originalSave
		renameDelete = originalDelete
	})

	err := runRenameQuiet(t, "old", "new")
	if err == nil || !strings.Contains(err.Error(), "rollback failed") {
		t.Fatalf("config and rollback failure = %v, want rollback detail", err)
	}
	assertToken(t, "old", "A-old")
	assertToken(t, "new", "A-old")
}

func TestRunRename_RetryAfterTransientConfigFailure(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	if err := config.SaveConfig(&config.Config{CredentialRef: "google-readonly/old"}); err != nil {
		t.Fatal(err)
	}
	original := renameSaveConfig
	attempts := 0
	renameSaveConfig = func(cfg *config.Config) error {
		attempts++
		if attempts == 1 {
			return errors.New("transient config failure")
		}
		return original(cfg)
	}
	t.Cleanup(func() { renameSaveConfig = original })

	if err := runRenameQuiet(t, "old", "new"); err == nil {
		t.Fatal("first rename should fail while saving config")
	}
	if err := runRenameQuiet(t, "old", "new"); err != nil {
		t.Fatalf("retry rename: %v", err)
	}
	assertNoToken(t, "old")
	assertToken(t, "new", "A-old")
	cfg, err := config.LoadConfigForRuntime()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CredentialRef != "google-readonly/new" {
		t.Fatalf("credential_ref after retry = %q, want google-readonly/new", cfg.CredentialRef)
	}
}

func TestRunRename_DeleteFailureRetainsBothBundles(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	if err := config.SaveConfig(&config.Config{CredentialRef: "google-readonly/old"}); err != nil {
		t.Fatal(err)
	}
	original := renameDelete
	renameDelete = func(_ *keychain.Store, _ string) error { return errors.New("delete failed") }
	t.Cleanup(func() { renameDelete = original })

	err := runRenameQuiet(t, "old", "new")
	if err == nil || !strings.Contains(err.Error(), "could not be removed") {
		t.Fatalf("delete failure = %v, want source-removal error", err)
	}
	assertToken(t, "old", "A-old")
	assertToken(t, "new", "A-old")
	cfg, loadErr := config.LoadConfigForRuntime()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if cfg.CredentialRef != "google-readonly/new" {
		t.Fatalf("credential_ref after delete failure = %q, want new", cfg.CredentialRef)
	}
}

func TestRunRename_IgnoresInvocationSelectorForSource(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	keychain.SetCredentialRefOverride("google-readonly/other", true)
	t.Cleanup(func() { keychain.SetCredentialRefOverride("", false) })

	if err := runRenameQuiet(t, "old", "new"); err != nil {
		t.Fatalf("runRename with selector override: %v", err)
	}
	assertToken(t, "new", "A-old")
	assertNoToken(t, "other")
}

func TestRunRename_ReplacesStaleCachedDestination(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	if err := identitycache.Put("old", "old@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := identitycache.Put("new", "stale@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := runRenameQuiet(t, "old", "new"); err != nil {
		t.Fatalf("runRename: %v", err)
	}
	cached := identitycache.Load()
	if _, ok := cached["old"]; ok {
		t.Fatal("old cached identity remains after rename")
	}
	if got := cached["new"].Email; got != "old@example.com" {
		t.Fatalf("destination cached identity = %q, want old@example.com", got)
	}
}

func TestRunRename_RemovesStaleCachedDestinationWithoutSourceCache(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	if err := identitycache.Put("new", "stale@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := runRenameQuiet(t, "old", "new"); err != nil {
		t.Fatalf("runRename: %v", err)
	}
	if _, ok := identitycache.Load()["new"]; ok {
		t.Fatal("stale destination cached identity remains")
	}
}

func TestRunRename_CacheFailureWarnsAfterCredentialSuccess(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	original := renameIdentity
	renameIdentity = func(_, _ string) error { return errors.New("cache unavailable") }
	t.Cleanup(func() { renameIdentity = original })

	var runErr error
	out := capture(t, func() {
		stderr := captureStderr(t, func() { runErr = runRename("old", "new") })
		if !strings.Contains(stderr, "cached identity was not moved") {
			t.Errorf("stderr = %q, want cache warning", stderr)
		}
	})
	if runErr != nil {
		t.Fatalf("runRename with cache failure: %v", runErr)
	}
	if !strings.Contains(out, "Renamed profile google-readonly/old to google-readonly/new.") {
		t.Fatalf("stdout = %q, want successful rename", out)
	}
	assertNoToken(t, "old")
	assertToken(t, "new", "A-old")
}

func assertToken(t *testing.T, profile, want string) {
	t.Helper()
	st, err := keychain.OpenRef("google-readonly/" + profile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	tok, err := st.Token()
	if err != nil || tok.AccessToken != want {
		t.Fatalf("%s token = %+v, err=%v; want access token %q", profile, tok, err, want)
	}
}

func assertNoToken(t *testing.T, profile string) {
	t.Helper()
	st, err := keychain.OpenRef("google-readonly/" + profile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if has, err := st.HasToken(); err != nil || has {
		t.Fatalf("%s token presence = (%v, %v), want (false, nil)", profile, has, err)
	}
}

func TestRunRenameWarnsWhenEnvironmentStillNamesOldRef(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "old")
	t.Setenv(keychain.CredentialRefEnvVar(), "google-readonly/old")

	out := capture(t, func() {
		if err := runRename("old", "new"); err != nil {
			t.Errorf("runRename: %v", err)
		}
	})
	for _, want := range []string{keychain.CredentialRefEnvVar(), "update it in this shell"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunRenameSameProfileIsNoOp(t *testing.T) {
	credtest.Setup(t)
	seedToken(t, "work")

	out := capture(t, func() {
		if err := runRename("work", "work"); err != nil {
			t.Errorf("runRename: %v", err)
		}
	})
	if !strings.Contains(out, "already named") {
		t.Fatalf("same-profile output = %q, want no-op message", out)
	}
	st, err := keychain.OpenRef("google-readonly/work")
	if err != nil {
		t.Fatal(err)
	}
	has, err := st.HasToken()
	_ = st.Close()
	if err != nil || !has {
		t.Fatalf("token after same-profile rename = (%v, %v), want (true, nil)", has, err)
	}
}

func TestRunRename_MissingSourceRejected(t *testing.T) {
	credtest.Setup(t)
	err := runRenameQuiet(t, "missing", "new")
	if err == nil || !errors.Is(err, keychain.ErrProfileNotFound) {
		t.Fatalf("missing source error = %v, want ErrProfileNotFound", err)
	}
	assertNoToken(t, "new")
}

// runUseQuiet runs runUse with stdout swallowed (these cases assert on error
// or config state, not output).
func runUseQuiet(t *testing.T, arg string) error {
	t.Helper()
	var err error
	capture(t, func() { err = runUse(arg) })
	return err
}

func runRenameQuiet(t *testing.T, oldProfile, newProfile string) error {
	t.Helper()
	var err error
	capture(t, func() { err = runRename(oldProfile, newProfile) })
	return err
}
