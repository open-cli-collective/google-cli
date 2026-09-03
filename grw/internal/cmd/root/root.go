// Package root provides the top-level grw command and global flags.
package root

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	config "github.com/open-cli-collective/google-cli-common/configcmd"
	"github.com/open-cli-collective/google-cli-common/initcmd"
	"github.com/open-cli-collective/google-cli-common/refreshcmd"
	"github.com/open-cli-collective/google-cli-common/rootutil"
	"github.com/open-cli-collective/google-cli-common/setcred"
	"github.com/open-cli-collective/google-cli-common/version"

	"github.com/open-cli-collective/google-readwrite/internal/cmd/mail"
)

var (
	verbose bool
	noColor bool
)

var rootCmd = &cobra.Command{
	Use:   "grw",
	Short: "A read-write CLI for Gmail cleanup and organization",
	Long: `grw is a read-write command-line interface for Gmail.

It reads and organizes mail like its read-only sibling gro, and adds the
operations gro deliberately cannot perform: deleting messages (Trash by
default, permanent behind a guard), managing labels as folders/subfolders, and
creating Gmail filters.

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
	rootCmd.AddCommand(setcred.NewCmd())
	rootCmd.AddCommand(mail.NewCommand())
	rootCmd.AddCommand(refreshcmd.NewCommand())
}
