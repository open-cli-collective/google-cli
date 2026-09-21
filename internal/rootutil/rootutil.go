// Package rootutil holds the root-command scaffolding shared by every CLI built
// on this library: the standard global flags (--verbose, --no-color,
// --backend, --profile), the PersistentPreRunE wiring that records the
// backend/profile selection for the next keychain.Open call, and the
// migration-notice flush that must wrap execution. Each CLI's root package
// supplies its own Use/Short/Long and command set and calls these helpers, so
// the plumbing lives in exactly one place.
package rootutil

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/keychain"
	"github.com/open-cli-collective/google-cli/internal/log"
	"github.com/open-cli-collective/google-cli/internal/migrationsink"
)

// ProfileFlagName is the global per-invocation credential-profile selector.
const ProfileFlagName = "profile"

// AddGlobalFlags registers the standard persistent flags on cmd, binding
// verbose and noColor to the given pointers. The --profile help shows the env var
// by its <SERVICE>_ pattern rather than the resolved name because flags are
// registered at package-init time, before config.Register runs.
func AddGlobalFlags(cmd *cobra.Command, verbose, noColor *bool) {
	cmd.PersistentFlags().BoolVarP(verbose, "verbose", "v", false, "Enable verbose output for debugging")
	cmd.PersistentFlags().BoolVar(noColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().String(cccredstore.BackendFlagName, "", cccredstore.BackendFlagUsage())
	cmd.PersistentFlags().String(ProfileFlagName, "", fmt.Sprintf(
		"Select profile <name> for this invocation, so concurrent commands "+
			"can target different accounts without racing on config.yml "+
			"(precedence: --%s flag > <SERVICE>_CREDENTIAL_REF env > config credential_ref)",
		ProfileFlagName))
}

// ApplyGlobalFlags runs the shared PersistentPreRunE logic: verbosity, color,
// and backend/profile wiring. Each root calls it from its own PersistentPreRunE.
func ApplyGlobalFlags(cmd *cobra.Command, verbose, noColor bool) error {
	log.Verbose = verbose
	if noColor {
		lipgloss.DefaultRenderer().SetColorProfile(termenv.Ascii)
	}
	if err := WireBackendSelection(cmd); err != nil {
		return err
	}
	return WireCredentialRefSelection(cmd)
}

// WireBackendSelection validates the user-supplied --backend flag and records
// it for the next keychain.Open* call. Cobra-layer only — it does NOT load
// config; openWith binds the flag pair against cfg.Keyring.Backend at the
// single credstore.Open call site. Exported because cobra does NOT chain
// PersistentPreRunE, so a subcommand that defines its own must call it
// explicitly. Reads via cmd.Flag() so persistent-flag inheritance works from
// any subcommand path.
func WireBackendSelection(cmd *cobra.Command) error {
	var value string
	var changed bool
	if bf := cmd.Flag(cccredstore.BackendFlagName); bf != nil {
		value = bf.Value.String()
		changed = bf.Changed
	}
	if err := cccredstore.BindBackendFlag(&cccredstore.Options{}, value, changed, ""); err != nil {
		return fmt.Errorf("--%s: %w", cccredstore.BackendFlagName, err)
	}
	keychain.SetBackendFlagOverride(value, changed)
	return nil
}

// WireCredentialRefSelection qualifies the user-supplied --profile name with
// the registered service and records it for the next keychain.Open* call. The
// credstore formatter validates the bare name before any keyring work. The
// resolved precedence (--profile flag > <SERVICE>_CREDENTIAL_REF env > config
// credential_ref) is applied at keychain.open.
func WireCredentialRefSelection(cmd *cobra.Command) error {
	f := cmd.Flag(ProfileFlagName)
	if f == nil {
		return nil
	}
	value := f.Value.String()
	changed := f.Changed
	if changed {
		service, _, err := cccredstore.ParseRef(config.DefaultCredentialRef)
		if err != nil {
			return fmt.Errorf("invalid default credential ref: %w", err)
		}
		value, err = cccredstore.FormatRef(service, value)
		if err != nil {
			return fmt.Errorf("--%s: %w", ProfileFlagName, err)
		}
	}
	keychain.SetCredentialRefOverride(value, changed)
	return nil
}

// RunWithMigrationNotice executes rootCmd with the deferred §1.8
// migration-notice flush. The defer fires on success AND error, before any
// os.Exit, so a one-time migration signal is never lost. A --json command
// consumes the record via output.JSON, making this a no-op for it; everything
// else gets the human stderr line.
func RunWithMigrationNotice(ctx context.Context, rootCmd *cobra.Command) error {
	defer migrationsink.FlushMigrationNotice(os.Stderr)
	return rootCmd.ExecuteContext(ctx)
}
