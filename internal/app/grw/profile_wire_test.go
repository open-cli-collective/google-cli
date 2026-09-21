package grw

import (
	"strings"
	"testing"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/keychain"
	"github.com/open-cli-collective/google-cli/internal/rootutil"
)

func resetProfileTestState(t *testing.T) {
	t.Helper()
	keychain.SetCredentialRefOverride("", false)
	keychain.SetBackendFlagOverride("", false)
	for _, name := range []string{rootutil.ProfileFlagName, cccredstore.BackendFlagName} {
		if f := rootCmd.PersistentFlags().Lookup(name); f != nil {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		}
	}
	t.Cleanup(func() {
		keychain.SetCredentialRefOverride("", false)
		keychain.SetBackendFlagOverride("", false)
		rootCmd.SetArgs(nil)
	})
}

func TestProfileSelectionQualifiesGrwService(t *testing.T) {
	resetProfileTestState(t)
	config.Register(Identity())
	keychain.SetCredentialRefOverride("", false)
	t.Cleanup(func() { keychain.SetCredentialRefOverride("", false); rootCmd.SetArgs(nil) })

	probe := &cobra.Command{Use: "probe-profile", RunE: func(*cobra.Command, []string) error { return nil }}
	rootCmd.AddCommand(probe)
	t.Cleanup(func() { rootCmd.RemoveCommand(probe) })
	rootCmd.SetArgs([]string{"probe-profile", "--profile", "work"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, set := keychain.GetCredentialRefOverride(); !set || got != "google-readwrite/work" {
		t.Errorf("override = (%q, %v), want (google-readwrite/work, true)", got, set)
	}
}

func TestProfileSelectionRejectsInvalidBareName(t *testing.T) {
	resetProfileTestState(t)
	config.Register(Identity())
	keychain.SetCredentialRefOverride("", false)
	t.Cleanup(func() { keychain.SetCredentialRefOverride("", false); rootCmd.SetArgs(nil) })

	probe := &cobra.Command{Use: "probe-profile-invalid", RunE: func(*cobra.Command, []string) error { return nil }}
	rootCmd.AddCommand(probe)
	t.Cleanup(func() { rootCmd.RemoveCommand(probe) })
	rootCmd.SetArgs([]string{"probe-profile-invalid", "--profile", "bad.profile"})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--"+rootutil.ProfileFlagName) {
		t.Fatalf("expected invalid --profile error, got %v", err)
	}
}

// TestInvalidProfileStopsBeforeLeaf proves invalid profile syntax is rejected
// by the root pre-run hook before a real leaf can touch a keyring or API.
func TestInvalidProfileStopsBeforeLeaf(t *testing.T) {
	resetProfileTestState(t)
	config.Register(Identity())
	called := false
	probe := &cobra.Command{
		Use: "probe-profile-sentinel",
		RunE: func(*cobra.Command, []string) error {
			called = true
			return nil
		},
	}
	rootCmd.AddCommand(probe)
	t.Cleanup(func() { rootCmd.RemoveCommand(probe) })
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

// TestNoRefFlagOnRealCommandTree walks every production command so a local
// --ref cannot hide behind the global --profile selector.
func TestNoRefFlagOnRealCommandTree(t *testing.T) {
	resetProfileTestState(t)

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

func TestRefFlagIsRejectedByRealGrwCommands(t *testing.T) {
	resetProfileTestState(t)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "init", args: []string{"init"}},
		{name: "mail search", args: []string{"mail", "search", "is:unread"}},
		{name: "set-credential", args: []string{"set-credential"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append(append([]string(nil), tc.args...), "--ref", "google-readwrite/work")
			rootCmd.SetArgs(args)
			err := rootCmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "unknown flag") {
				t.Fatalf("%v must reject removed --ref, got %v", tc.name, err)
			}
		})
	}
}
