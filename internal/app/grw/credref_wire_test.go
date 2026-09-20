package grw

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	initcmd "github.com/open-cli-collective/google-cli/internal/cmd/init"
	"github.com/open-cli-collective/google-cli/internal/cmd/setcred"
	"github.com/open-cli-collective/google-cli/internal/credtest"
	"github.com/open-cli-collective/google-cli/internal/keychain"
	"github.com/open-cli-collective/google-cli/internal/rootutil"
)

func selectorTestRoot() *cobra.Command {
	var verbose, noColor bool
	root := &cobra.Command{
		Use: "grw",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return rootutil.ApplyGlobalFlags(cmd, verbose, noColor)
		},
	}
	rootutil.AddGlobalFlags(root, &verbose, &noColor)
	root.AddCommand(initcmd.NewCommand())
	root.AddCommand(setcred.NewCmd())
	return root
}

func TestProfileFlagInheritedByInitInBothFlagOrders(t *testing.T) {
	for _, tc := range []struct {
		name string
		args func(string) []string
	}{
		{name: "before command", args: func(path string) []string {
			return []string{"--profile", "work", "init", "--credentials-file", path}
		}},
		{name: "after command", args: func(path string) []string {
			return []string{"init", "--profile", "work", "--credentials-file", path}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			credtest.Setup(t)
			t.Setenv(keychain.CredentialRefEnvVar(), "")
			root := selectorTestRoot()
			root.SetArgs(tc.args(filepath.Join(t.TempDir(), "missing.json")))
			if err := root.Execute(); err == nil {
				t.Fatal("init should fail for the intentionally missing client file")
			}
			if got, set := keychain.GetCredentialRefOverride(); !set || got != "google-readwrite/work" {
				t.Fatalf("selector after init path = (%q, %v), want google-readwrite/work", got, set)
			}
		})
	}
}

func TestProfileFlagInheritedBySetCredentialTargetsNamedProfile(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "before command", args: []string{"--profile", "work", "set-credential", "--key", "oauth_token", "--stdin"}},
		{name: "after command", args: []string{"set-credential", "--profile", "work", "--key", "oauth_token", "--stdin"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			credtest.Setup(t)
			t.Setenv(keychain.CredentialRefEnvVar(), "")
			root := selectorTestRoot()
			root.SetIn(strings.NewReader(`{"access_token":"profile-token","refresh_token":"refresh"}`))
			root.SetArgs(tc.args)
			if err := root.Execute(); err != nil {
				t.Fatalf("set-credential: %v", err)
			}
			st, err := keychain.OpenRef("google-readwrite/work")
			if err != nil {
				t.Fatal(err)
			}
			tok, err := st.Token()
			_ = st.Close()
			if err != nil || tok.AccessToken != "profile-token" {
				t.Fatalf("named profile token = %+v, err=%v", tok, err)
			}
			assertNoTokenAtRef(t, "google-readwrite/default")
		})
	}
}

func TestProfileAndSetCredentialRefAreMutuallyExclusive(t *testing.T) {
	credtest.Setup(t)
	root := selectorTestRoot()
	root.SetIn(strings.NewReader(`{"access_token":"profile-token"}`))
	root.SetArgs([]string{"--profile", "work", "set-credential", "--ref", "google-readwrite/other", "--key", "oauth_token", "--stdin"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("profile/local --ref conflict = %v, want mutual-exclusion error", err)
	}
}

func assertNoTokenAtRef(t *testing.T, ref string) {
	t.Helper()
	st, err := keychain.OpenRef(ref)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if has, err := st.HasToken(); err != nil || has {
		t.Fatalf("%s token presence = (%v, %v), want (false, nil)", ref, has, err)
	}
}
