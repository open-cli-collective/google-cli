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

func TestWireCredentialRefSelection_FlagInvalid(t *testing.T) {
	resetState(t)

	probe := newProbeCmd("probe-profile-invalid")
	rootCmd.AddCommand(probe)
	defer removeChild(t, probe)
	rootCmd.SetArgs([]string{"probe-profile-invalid", "--profile", "bad.profile"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--"+rootutil.ProfileFlagName) {
		t.Errorf("error should mention --%s: %v", rootutil.ProfileFlagName, err)
	}
}

// TestWireCredentialRefSelection_InvalidStopsBeforeLeaf proves invalid
// profile syntax is rejected by the root pre-run hook before a leaf can touch
// a keyring or API collaborator.
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

func TestProfile_SetCredentialInheritsPersistent(t *testing.T) {
	canonical := rootCmd.PersistentFlags().Lookup(rootutil.ProfileFlagName)
	if canonical == nil {
		t.Fatalf("root persistent flag --%s not registered", rootutil.ProfileFlagName)
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
	if got := sc.Flag(rootutil.ProfileFlagName); got != canonical {
		t.Errorf("set-credential --%s = %p, want canonical %p", rootutil.ProfileFlagName, got, canonical)
	}
	for _, name := range []string{"init", "me"} {
		var command *cobra.Command
		for _, c := range rootCmd.Commands() {
			if c.Name() == name {
				command = c
				break
			}
		}
		if command == nil {
			t.Fatalf("%s command not registered on rootCmd", name)
		}
		if got := command.Flag(rootutil.ProfileFlagName); got != canonical {
			t.Errorf("%s --%s = %p, want canonical %p", name, rootutil.ProfileFlagName, got, canonical)
		}
	}

	me := newProbeCmd("probe-profile-inherit")
	rootCmd.AddCommand(me)
	defer removeChild(t, me)
	if got := me.Flag(rootutil.ProfileFlagName); got != canonical {
		t.Errorf("read command --%s = %p, want canonical %p", rootutil.ProfileFlagName, got, canonical)
	}
}

func TestPublicRefFlagIsUnknown(t *testing.T) {
	resetState(t)
	probe := newProbeCmd("probe-ref-unknown")
	rootCmd.AddCommand(probe)
	defer removeChild(t, probe)
	rootCmd.SetArgs([]string{"probe-ref-unknown", "--ref", "google-readonly/acct-a"})
	if err := rootCmd.Execute(); err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--ref should be unknown, got %v", err)
	}
}

// TestNoRefFlagOnRealCommandTree protects the public surface after replacing
// the old service/profile --ref selector with the global bare --profile flag.
// A probe command alone would miss a leaf that accidentally reintroduced a
// local --ref, so walk every production command and inspect real argv paths.
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

func TestRefFlagIsRejectedByRealGroCommands(t *testing.T) {
	resetState(t)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "init", args: []string{"init"}},
		{name: "mail search", args: []string{"mail", "search", "is:unread"}},
		{name: "set-credential", args: []string{"set-credential"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append(append([]string(nil), tc.args...), "--ref", "google-readonly/work")
			rootCmd.SetArgs(args)
			err := rootCmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "unknown flag") {
				t.Fatalf("%v must reject removed --ref, got %v", tc.name, err)
			}
		})
	}
}
