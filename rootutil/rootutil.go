// Package rootutil holds the root-command scaffolding shared by every CLI built
// on this library: the standard global flags (--verbose, --no-color,
// --backend, --ref), the PersistentPreRunE wiring that records the
// backend/credential-ref selection for the next keychain.Open call, and the
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

	"github.com/open-cli-collective/google-cli-common/keychain"
	"github.com/open-cli-collective/google-cli-common/log"
	"github.com/open-cli-collective/google-cli-common/migrationsink"
)

// CredentialRefFlagName is the global per-invocation credential-ref selector.
// It shares its name with set-credential's own write-target --ref (a local flag
// that shadows this persistent one for that command only).
const CredentialRefFlagName = "ref"

// AddGlobalFlags registers the standard persistent flags on cmd, binding
// verbose and noColor to the given pointers. The --ref help shows the env var
// by its <SERVICE>_ pattern rather than the resolved name because flags are
// registered at package-init time, before config.Register runs.
func AddGlobalFlags(cmd *cobra.Command, verbose, noColor *bool) {
	cmd.PersistentFlags().BoolVarP(verbose, "verbose", "v", false, "Enable verbose output for debugging")
	cmd.PersistentFlags().BoolVar(noColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().String(cccredstore.BackendFlagName, "", cccredstore.BackendFlagUsage())
	cmd.PersistentFlags().String(CredentialRefFlagName, "", fmt.Sprintf(
		"Credential ref <service>/<profile> for this invocation, so concurrent commands "+
			"can target different accounts without racing on config.yml "+
			"(precedence: --%s flag > <SERVICE>_CREDENTIAL_REF env > config credential_ref)",
		CredentialRefFlagName))
}

// ApplyGlobalFlags runs the shared PersistentPreRunE logic: verbosity, color,
// and backend/ref wiring. Each root calls it from its own PersistentPreRunE.
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

// WireCredentialRefSelection records the user-supplied --ref flag for the next
// keychain.Open* call and validates its <service>/<profile> shape up front so a
// bad value fails with a clear "--ref" error before any keyring work. The
// resolved precedence (--ref flag > <SERVICE>_CREDENTIAL_REF env > config
// credential_ref) is applied at keychain.open; this hook only records the flag.
func WireCredentialRefSelection(cmd *cobra.Command) error {
	f := cmd.Flag(CredentialRefFlagName)
	if f == nil {
		return nil
	}
	value := f.Value.String()
	changed := f.Changed
	if changed && value != "" {
		if _, _, err := cccredstore.ParseRef(value); err != nil {
			return fmt.Errorf("--%s: %w", CredentialRefFlagName, err)
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
