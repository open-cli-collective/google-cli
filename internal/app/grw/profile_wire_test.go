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
	config.Register(Identity())
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

func TestInvalidProfileStopsBeforeLeaf(t *testing.T) {
	resetProfileTestState(t)
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

func TestRefFlagIsRejectedBySetCredential(t *testing.T) {
	resetProfileTestState(t)
	rootCmd.SetArgs([]string{"set-credential", "--ref", "google-readwrite/work"})
	if err := rootCmd.Execute(); err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("set-credential must reject removed --ref, got %v", err)
	}
}
