// Package main is the entry point for the gro CLI.
//
// Distribution is fully automated: merges to main with feat:/fix: prefixes
// trigger auto-release, which runs GoReleaser (handling Homebrew + binary
// artifacts) and emits a release-published event that fans out to the
// chocolatey and winget publish workflows.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-cli-collective/google-cli/internal/app/gro"
	"github.com/open-cli-collective/google-cli/internal/config"
)

func main() {
	// Register this CLI's identity before any config/keychain/auth call: it
	// stamps the config dir, keyring service, env-var prefixes, and scope set.
	config.Register(gro.Identity())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	gro.ExecuteContext(ctx)
}
