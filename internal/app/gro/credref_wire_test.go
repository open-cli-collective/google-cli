package gro

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

// TestWireCredentialRefSelection_FlagSet proves a --ref on a real command
// path is recorded in the override the keychain.open resolver reads.
func TestWireCredentialRefSelection_FlagSet(t *testing.T) {
	resetState(t)
	t.Setenv(keychain.CredentialRefEnvVar(), "")

	probe := newProbeCmd("probe-ref-flagset")
	rootCmd.AddCommand(probe)
	defer removeChild(t, probe)
	rootCmd.SetArgs([]string{"probe-ref-flagset", "--ref", "google-readonly/acct-a"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	v, set := keychain.GetCredentialRefOverride()
	if !set {
		t.Fatalf("override flagSet = false, want true")
	}
	if v != "google-readonly/acct-a" {
		t.Errorf("override value = %q, want %q", v, "google-readonly/acct-a")
	}
}

// TestWireCredentialRefSelection_FlagInvalid asserts a malformed --ref fails
// up front with a clear "--ref" error, before any keyring work.
func TestWireCredentialRefSelection_FlagInvalid(t *testing.T) {
	resetState(t)

	probe := newProbeCmd("probe-ref-invalid")
	rootCmd.AddCommand(probe)
	defer removeChild(t, probe)
	rootCmd.SetArgs([]string{"probe-ref-invalid", "--ref", "no-slash"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--"+rootutil.CredentialRefFlagName) {
		t.Errorf("error should mention --%s: %v", rootutil.CredentialRefFlagName, err)
	}
}

// TestWireCredentialRefSelection_ShadowingSubcommand regresses the
// cobra-doesn't-chain-PersistentPreRunE bug for --ref, mirroring the
// --backend guard.
func TestWireCredentialRefSelection_ShadowingSubcommand(t *testing.T) {
	resetState(t)
	t.Setenv(keychain.CredentialRefEnvVar(), "")

	shadow := &cobra.Command{
		Use: "shadow-ref",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return WireCredentialRefSelection(cmd)
		},
	}
	leaf := newProbeCmd("leaf")
	shadow.AddCommand(leaf)
	rootCmd.AddCommand(shadow)
	defer removeChild(t, shadow)

	rootCmd.SetArgs([]string{"shadow-ref", "leaf", "--ref", "google-readonly/acct-b"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute through shadowing PreRunE: %v", err)
	}
	v, set := keychain.GetCredentialRefOverride()
	if !set || v != "google-readonly/acct-b" {
		t.Errorf("override = (%q, %v); want (\"google-readonly/acct-b\", true) — shadower's PreRunE failed to invoke WireCredentialRefSelection", v, set)
	}
}

// TestCredentialRef_SetCredentialShadowsPersistent documents the intentional
// exception to the inherit-everywhere rule: `set-credential` keeps its own
// local --ref (the write target), so it must resolve to a DIFFERENT *pflag.Flag
// than the root's persistent selector — while a read command inherits the
// canonical persistent one. A regression that dropped set-credential's local
// flag (or that made a read command shadow --ref) would flip these.
func TestCredentialRef_SetCredentialShadowsPersistent(t *testing.T) {
	canonical := rootCmd.PersistentFlags().Lookup(rootutil.CredentialRefFlagName)
	if canonical == nil {
		t.Fatalf("root persistent flag --%s not registered", rootutil.CredentialRefFlagName)
	}

	var sc *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "set-credential" {
			sc = c
			break
		}
	}
	if sc == nil {
		t.Fatal("set-credential command not registered on rootCmd")
	}
	if got := sc.Flag(rootutil.CredentialRefFlagName); got == nil {
		t.Fatalf("set-credential has no --%s", rootutil.CredentialRefFlagName)
	} else if got == canonical {
		t.Errorf("set-credential --%s resolved to the persistent flag; expected its own local shadow", rootutil.CredentialRefFlagName)
	}

	// A read command (no local --ref) must inherit the canonical persistent flag.
	me := newProbeCmd("probe-ref-inherit")
	rootCmd.AddCommand(me)
	defer removeChild(t, me)
	if got := me.Flag(rootutil.CredentialRefFlagName); got != canonical {
		t.Errorf("read command --%s = %p, want canonical %p (unexpected shadow)", rootutil.CredentialRefFlagName, got, canonical)
	}
}

func selectorTestRoot() *cobra.Command {
	var verbose, noColor bool
	root := &cobra.Command{
		Use: "gro",
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
			if got, set := keychain.GetCredentialRefOverride(); !set || got != "google-readonly/work" {
				t.Fatalf("selector after init path = (%q, %v), want google-readonly/work", got, set)
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
			st, err := keychain.OpenRef("google-readonly/work")
			if err != nil {
				t.Fatal(err)
			}
			tok, err := st.Token()
			_ = st.Close()
			if err != nil || tok.AccessToken != "profile-token" {
				t.Fatalf("named profile token = %+v, err=%v", tok, err)
			}
			assertNoTokenAtRef(t, "google-readonly/default")
		})
	}
}

func TestProfileAndSetCredentialRefAreMutuallyExclusive(t *testing.T) {
	credtest.Setup(t)
	root := selectorTestRoot()
	root.SetIn(strings.NewReader(`{"access_token":"profile-token"}`))
	root.SetArgs([]string{"--profile", "work", "set-credential", "--ref", "google-readonly/other", "--key", "oauth_token", "--stdin"})
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
