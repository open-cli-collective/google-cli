package gro

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-cli/internal/keychain"
	"github.com/open-cli-collective/google-cli/internal/rootutil"
)

func TestWireCredentialRefSelection_FlagSet(t *testing.T) {
	resetState(t)
	t.Setenv(keychain.CredentialRefEnvVar(), "")

	probe := newProbeCmd("probe-profile-flagset")
	rootCmd.AddCommand(probe)
	defer removeChild(t, probe)
	rootCmd.SetArgs([]string{"probe-profile-flagset", "--profile", "acct-a"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	v, set := keychain.GetCredentialRefOverride()
	if !set || v != "google-readonly/acct-a" {
		t.Errorf("override = (%q, %v), want (google-readonly/acct-a, true)", v, set)
	}
}

func TestWireCredentialRefSelection_InvalidStopsBeforeLeaf(t *testing.T) {
	resetState(t)
	called := false
	probe := &cobra.Command{
		Use: "probe-profile-sentinel",
		RunE: func(*cobra.Command, []string) error {
			called = true
			return nil
		},
	}
	rootCmd.AddCommand(probe)
	defer removeChild(t, probe)
	rootCmd.SetArgs([]string{"probe-profile-sentinel", "--profile", "bad.profile"})

	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--"+rootutil.ProfileFlagName) {
		t.Fatalf("expected invalid --profile error, got %v", err)
	}
	if called {
		t.Fatal("invalid --profile must stop before the leaf RunE")
	}
	if _, set := keychain.GetCredentialRefOverride(); set {
		t.Fatal("invalid --profile must not record a credential-ref override")
	}
}

func TestWireCredentialRefSelection_ShadowingSubcommand(t *testing.T) {
	resetState(t)
	t.Setenv(keychain.CredentialRefEnvVar(), "")

	shadow := &cobra.Command{
		Use: "shadow-profile",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return WireCredentialRefSelection(cmd)
		},
	}
	leaf := newProbeCmd("leaf")
	shadow.AddCommand(leaf)
	rootCmd.AddCommand(shadow)
	defer removeChild(t, shadow)

	rootCmd.SetArgs([]string{"shadow-profile", "leaf", "--profile", "acct-b"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute through shadowing PreRunE: %v", err)
	}
	v, set := keychain.GetCredentialRefOverride()
	if !set || v != "google-readonly/acct-b" {
		t.Errorf("override = (%q, %v); want (google-readonly/acct-b, true)", v, set)
	}
}

func TestProfile_InheritsPersistentOnRealCommandTree(t *testing.T) {
	canonical := rootCmd.PersistentFlags().Lookup(rootutil.ProfileFlagName)
	if canonical == nil {
		t.Fatalf("root persistent flag --%s not registered", rootutil.ProfileFlagName)
	}

	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		children := cmd.Commands()
		if len(children) == 0 {
			if got := cmd.Flag(rootutil.ProfileFlagName); got != canonical {
				t.Errorf("%q: --%s = %p, want canonical %p", cmd.CommandPath(), rootutil.ProfileFlagName, got, canonical)
			}
			return
		}
		for _, child := range children {
			walk(child)
		}
	}
	walk(rootCmd)
}

func TestNoRefFlagOnRealCommandTree(t *testing.T) {
	resetState(t)

	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if f := cmd.Flag("ref"); f != nil {
			t.Errorf("%q exposes removed --ref flag (%s)", cmd.CommandPath(), f.Usage)
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(rootCmd)
}

func TestRefFlagIsRejectedBySetCredential(t *testing.T) {
	resetState(t)
	rootCmd.SetArgs([]string{"set-credential", "--ref", "google-readonly/work"})
	if err := rootCmd.Execute(); err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("set-credential must reject removed --ref, got %v", err)
	}
}
