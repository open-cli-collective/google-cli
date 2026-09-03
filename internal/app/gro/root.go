// Package gro provides the gro identity and top-level command.
package gro

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/google-cli/internal/cmd/calendar"
	config "github.com/open-cli-collective/google-cli/internal/cmd/config"
	"github.com/open-cli-collective/google-cli/internal/cmd/contacts"
	"github.com/open-cli-collective/google-cli/internal/cmd/drive"
	initcmd "github.com/open-cli-collective/google-cli/internal/cmd/init"
	mail "github.com/open-cli-collective/google-cli/internal/cmd/mail"
	"github.com/open-cli-collective/google-cli/internal/cmd/me"
	profilescmd "github.com/open-cli-collective/google-cli/internal/cmd/profiles"
	refreshcmd "github.com/open-cli-collective/google-cli/internal/cmd/refresh"
	"github.com/open-cli-collective/google-cli/internal/cmd/setcred"
	"github.com/open-cli-collective/google-cli/internal/rootutil"
	"github.com/open-cli-collective/google-cli/internal/version"
)

var (
	verbose bool
	noColor bool
)

var rootCmd = &cobra.Command{
	Use:   "gro",
	Short: "A non-destructive CLI for Google services",
	Long: `gro is a non-destructive command-line interface for Google services.

It provides commands for reading and organizing Gmail messages, Google Calendar
events, Google Contacts, and Google Drive files. Organizational operations
include labeling, archiving, starring, RSVP, and group management. No send,
delete, or trash operations are possible.

To get started, run:
  gro init

This will guide you through OAuth setup for Google API access.`,
	Version: version.Version,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		return rootutil.ApplyGlobalFlags(cmd, verbose, noColor)
	},
}

// WireBackendSelection is a thin wrapper over rootutil so a subcommand that
// defines its own PersistentPreRunE can call it explicitly (cobra does NOT
// chain PersistentPreRunE). Retained on the root package for the regression
// tests that guard the shadowing pattern.
func WireBackendSelection(cmd *cobra.Command) error {
	return rootutil.WireBackendSelection(cmd)
}

// WireCredentialRefSelection is a thin wrapper over rootutil (see
// WireBackendSelection).
func WireCredentialRefSelection(cmd *cobra.Command) error {
	return rootutil.WireCredentialRefSelection(cmd)
}

// Execute runs the root command with a background context
func Execute() {
	ExecuteContext(context.Background())
}

// ExecuteContext runs the root command with the given context. os.Exit stays
// strictly AFTER RunWithMigrationNotice returns so its deferred migration flush
// is never skipped by the exit.
func ExecuteContext(ctx context.Context) {
	if err := rootutil.RunWithMigrationNotice(ctx, rootCmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Set custom version template to include commit and build date
	rootCmd.SetVersionTemplate("gro " + version.Info() + "\n")

	// Global flags (verbose, no-color, backend, ref)
	rootutil.AddGlobalFlags(rootCmd, &verbose, &noColor)

	// Register commands
	rootCmd.AddCommand(initcmd.NewCommand())
	rootCmd.AddCommand(config.NewCommand())
	rootCmd.AddCommand(profilescmd.NewCommand())
	rootCmd.AddCommand(setcred.NewCmd())
	rootCmd.AddCommand(me.NewCommand())
	rootCmd.AddCommand(mail.NewCommand())
	rootCmd.AddCommand(calendar.NewCommand())
	rootCmd.AddCommand(contacts.NewCommand())
	rootCmd.AddCommand(drive.NewCommand())
	rootCmd.AddCommand(refreshcmd.NewCommand())
}
