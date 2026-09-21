package grw

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/keychain"
	"github.com/open-cli-collective/google-cli/internal/rootutil"
)

func TestProfileSelectionQualifiesGrwService(t *testing.T) {
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
