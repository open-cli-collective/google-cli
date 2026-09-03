// Package grw provides the grw identity and top-level command.
package grw

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	config "github.com/open-cli-collective/google-cli/internal/cmd/config"
	initcmd "github.com/open-cli-collective/google-cli/internal/cmd/init"
	"github.com/open-cli-collective/google-cli/internal/cmd/me"
	profilescmd "github.com/open-cli-collective/google-cli/internal/cmd/profiles"
	refreshcmd "github.com/open-cli-collective/google-cli/internal/cmd/refresh"
	"github.com/open-cli-collective/google-cli/internal/cmd/setcred"
	"github.com/open-cli-collective/google-cli/internal/rootutil"
	rwcalendar "github.com/open-cli-collective/google-cli/internal/rwcmd/calendar"
	rwcontacts "github.com/open-cli-collective/google-cli/internal/rwcmd/contacts"
	"github.com/open-cli-collective/google-cli/internal/rwcmd/mail"
	"github.com/open-cli-collective/google-cli/internal/version"
)

var (
	verbose bool
	noColor bool
)

var rootCmd = &cobra.Command{
	Use:   "grw",
	Short: "A read-write CLI for Google services",
	Long: `grw is a read-write command-line interface for Google services.

It reads and organizes mail like its read-only sibling gro, and adds the
operations gro deliberately cannot perform: deleting messages (Trash by
default, permanent behind a guard), managing labels as folders/subfolders, and
creating Gmail filters. It also reads profile information and creates, updates,
and deletes Google Calendar events, contacts, and contact groups.

grw stores its credentials separately from gro (keyring namespace
"google-readwrite"), so the two can be used in isolation — e.g. give an agent
gro only, which structurally cannot delete anything, and keep grw gated.

To get started, run:
  grw init`,
	Version: version.Version,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		return rootutil.ApplyGlobalFlags(cmd, verbose, noColor)
	},
}

// Execute runs the root command with a background context.
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
	rootCmd.SetVersionTemplate("grw " + version.Info() + "\n")

	// Global flags (verbose, no-color, backend, ref)
	rootutil.AddGlobalFlags(rootCmd, &verbose, &noColor)

	// Register commands
	rootCmd.AddCommand(initcmd.NewCommand())
	rootCmd.AddCommand(config.NewCommand())
	rootCmd.AddCommand(profilescmd.NewCommand())
	rootCmd.AddCommand(setcred.NewCmd())
	rootCmd.AddCommand(me.NewCommand())
	rootCmd.AddCommand(mail.NewCommand())
	rootCmd.AddCommand(rwcalendar.NewCommand())
	rootCmd.AddCommand(rwcontacts.NewCommand())
	rootCmd.AddCommand(refreshcmd.NewCommand())
}
