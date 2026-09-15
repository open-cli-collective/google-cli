package rootutil

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/keychain"
)

func TestMain(m *testing.M) {
	config.RegisterForTest()
	os.Exit(m.Run())
}

func selectorRoot() *cobra.Command {
	var verbose, noColor bool
	root := &cobra.Command{
		Use: "gro",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return WireCredentialRefSelection(cmd)
		},
	}
	AddGlobalFlags(root, &verbose, &noColor)
	root.AddCommand(&cobra.Command{Use: "probe", Run: func(*cobra.Command, []string) {}})
	return root
}

func TestWireCredentialRefSelection_ProfileExpandsToRegisteredService(t *testing.T) {
	keychain.SetCredentialRefOverride("", false)
	t.Cleanup(func() { keychain.SetCredentialRefOverride("", false) })
	root := selectorRoot()
	root.SetArgs([]string{"--profile", "work", "probe"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, set := keychain.GetCredentialRefOverride(); !set || got != "google-readonly/work" {
		t.Fatalf("override = (%q, %v), want (google-readonly/work, true)", got, set)
	}
}

func TestWireCredentialRefSelection_ProfileBeatsEnvironment(t *testing.T) {
	keychain.SetCredentialRefOverride("", false)
	t.Cleanup(func() { keychain.SetCredentialRefOverride("", false) })
	t.Setenv(keychain.CredentialRefEnvVar(), "google-readonly/env")
	root := selectorRoot()
	root.SetArgs([]string{"--profile", "work", "probe"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got, set := keychain.GetCredentialRefOverride()
	if !set || got != "google-readonly/work" {
		t.Fatalf("override = (%q, %v), want explicit profile to win", got, set)
	}
}

func TestWireCredentialRefSelection_RejectsEmptyAndBothSelectors(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "empty profile", args: []string{"--profile=", "probe"}, want: "--profile"},
		{name: "empty ref preserves fall-through", args: []string{"--ref=", "probe"}},
		{name: "both", args: []string{"--profile", "work", "--ref", "google-readonly/other", "probe"}, want: "mutually exclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keychain.SetCredentialRefOverride("", false)
			root := selectorRoot()
			root.SetArgs(tc.args)
			err := root.Execute()
			if tc.want == "" {
				if err != nil {
					t.Fatalf("empty --ref should preserve legacy fall-through: %v", err)
				}
				if value, set := keychain.GetCredentialRefOverride(); !set || value != "" {
					t.Fatalf("override = (%q, %v), want explicit empty ref", value, set)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestWireCredentialRefSelection_NoSelectorClearsPreviousOverride(t *testing.T) {
	keychain.SetCredentialRefOverride("google-readonly/old", true)
	t.Cleanup(func() { keychain.SetCredentialRefOverride("", false) })
	root := selectorRoot()
	root.SetArgs([]string{"probe"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, set := keychain.GetCredentialRefOverride(); set || got != "" {
		t.Fatalf("override = (%q, %v), want cleared", got, set)
	}
}

func TestWireCredentialRefSelection_SeesShadowedRootRefForConflict(t *testing.T) {
	keychain.SetCredentialRefOverride("", false)
	root := selectorRoot()
	shadow := &cobra.Command{Use: "shadow", Run: func(*cobra.Command, []string) {}}
	shadow.Flags().String("ref", "", "local write target")
	root.AddCommand(shadow)
	root.SetArgs([]string{"--ref", "google-readonly/root", "--profile", "work", "shadow"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("shadowed --ref plus --profile = %v, want mutual-exclusion error", err)
	}
}
