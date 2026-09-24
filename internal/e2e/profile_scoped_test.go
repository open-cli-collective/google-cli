package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	clicache "github.com/open-cli-collective/cli-common/cache"

	"github.com/open-cli-collective/google-cli/internal/cache"
)

func TestBuiltCLIProfileStateIsHermetic(t *testing.T) {
	root := repositoryRoot(t)
	tmp := t.TempDir()
	configHome := filepath.Join(tmp, "config")
	cacheHome := filepath.Join(tmp, "cache")
	if err := os.MkdirAll(filepath.Join(configHome, "google-readonly"), 0o700); err != nil {
		t.Fatal(err)
	}
	defaultClient := `{"installed":{"client_id":"default-client.apps.googleusercontent.com","project_id":"p","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","client_secret":"default-secret","redirect_uris":["http://localhost"]}}`
	workClient := `{"installed":{"client_id":"work-client.apps.googleusercontent.com","project_id":"p","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","client_secret":"work-secret","redirect_uris":["http://localhost"]}}`
	defaultPath := filepath.Join(configHome, "google-readonly", "default.json")
	workPath := filepath.Join(configHome, "google-readonly", "work.json")
	if err := os.WriteFile(defaultPath, []byte(defaultClient), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workPath, []byte(workClient), 0o600); err != nil {
		t.Fatal(err)
	}
	configYAML := "credential_ref: google-readonly/default\nprofiles:\n" +
		"  default:\n" +
		"    oauth_client_path: " + defaultPath + "\n" +
		"    granted_scopes:\n" +
		"      - default-scope\n" +
		"  work:\n" +
		"    oauth_client_path: " + workPath + "\n" +
		"    granted_scopes:\n" +
		"      - work-scope\n"
	if err := os.WriteFile(filepath.Join(configHome, "google-readonly", "config.yml"), []byte(configYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(tmp, "gro")
	build := exec.Command("go", "build", "-tags", "keyring_no1password,keyring_nopassage", "-o", bin, "./cmd/gro")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build gro: %v\n%s", err, output)
	}
	runtimeEnv := hermeticEnv(configHome, cacheHome, tmp)

	defaultStatus := showCLI(t, bin, runtimeEnv, "--backend=file", "config", "show", "--json")
	workStatus := showCLI(t, bin, runtimeEnv, "--backend=file", "--profile", "work", "config", "show", "--json")
	if defaultStatus.OAuthClientFingerprint == "" || workStatus.OAuthClientFingerprint == "" || defaultStatus.OAuthClientFingerprint == workStatus.OAuthClientFingerprint {
		t.Fatalf("profile fingerprints = %q and %q, want distinct non-empty values", defaultStatus.OAuthClientFingerprint, workStatus.OAuthClientFingerprint)
	}
	if got := defaultStatus.GrantedScopes; len(got) != 1 || got[0] != "default-scope" {
		t.Fatalf("default scopes = %v, want [default-scope]", got)
	}
	if got := workStatus.GrantedScopes; len(got) != 1 || got[0] != "work-scope" {
		t.Fatalf("work scopes = %v, want [work-scope]", got)
	}
	unconfigured := showCLI(t, bin, runtimeEnv, "--backend=file", "--profile", "other", "config", "show", "--json")
	if unconfigured.OAuthClientPresent || unconfigured.OAuthClientFingerprint != "" || unconfigured.OAuthClientPath != "" {
		t.Fatalf("unconfigured profile exposed active client: %+v", unconfigured)
	}

	cacheBase := cacheHome
	switch runtime.GOOS {
	case "darwin":
		cacheBase = filepath.Join(tmp, "home", "Library", "Caches")
	case "windows":
		cacheBase = filepath.Join(tmp, "home", "AppData", "Local")
	}
	rootCache := filepath.Join(cacheBase, "google-readonly")
	if err := clicache.WriteResource(clicache.Locator{Root: rootCache, InstanceKey: "default"}, "drives", "24h", []*cache.CachedDrive{{ID: "default", Name: "Default"}}); err != nil {
		t.Fatal(err)
	}
	defaultFresh := refreshStatus(t, bin, runtimeEnv, "--backend=file", "--profile", "default", "refresh", "--status", "--json")
	workMiss := refreshStatus(t, bin, runtimeEnv, "--backend=file", "--profile", "work", "refresh", "--status", "--json")
	if defaultFresh != "fresh" || workMiss != "uninitialized" {
		t.Fatalf("cache status = default %q, work %q; want fresh/uninitialized", defaultFresh, workMiss)
	}
}

type status struct {
	OAuthClientPath        string   `json:"oauth_client_path"`
	OAuthClientPresent     bool     `json:"oauth_client_present"`
	OAuthClientFingerprint string   `json:"oauth_client_fingerprint"`
	GrantedScopes          []string `json:"granted_scopes"`
}

func showCLI(t *testing.T, bin string, env []string, args ...string) status {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gro %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	var got status
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("parse config show output: %v\n%s", err, output)
	}
	return got
}

func refreshStatus(t *testing.T, bin string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gro %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	var got struct {
		Resources []struct {
			Status string `json:"status"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("parse refresh output: %v\n%s", err, output)
	}
	if len(got.Resources) != 1 {
		t.Fatalf("refresh resources = %d, want 1", len(got.Resources))
	}
	return got.Resources[0].Status
}

func hermeticEnv(configHome, cacheHome, tmp string) []string {
	env := os.Environ()
	values := map[string]string{
		"HOME":                                    filepath.Join(tmp, "home"),
		"XDG_CONFIG_HOME":                         configHome,
		"XDG_CACHE_HOME":                          cacheHome,
		"XDG_DATA_HOME":                           filepath.Join(tmp, "data"),
		"XDG_STATE_HOME":                          filepath.Join(tmp, "state"),
		"GOOGLE_READONLY_KEYRING_BACKEND":         "file",
		"GOOGLE_READONLY_KEYRING_PASSPHRASE":      "built-cli-test-passphrase",
		"GRO_TEST_DISABLE_LEGACY_KEYCHAIN_SCAN":   "1",
		"GRO_TEST_DISABLE_LEGACY_SECRETTOOL_SCAN": "1",
	}
	for key, value := range values {
		setEnv(&env, key, value)
	}
	return env
}

func setEnv(env *[]string, key, value string) {
	prefix := key + "="
	for i, entry := range *env {
		if strings.HasPrefix(entry, prefix) {
			(*env)[i] = prefix + value
			return
		}
	}
	*env = append(*env, prefix+value)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
