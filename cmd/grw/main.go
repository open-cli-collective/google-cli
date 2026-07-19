// Package main is the entry point for the grw (google-readwrite) CLI.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-cli-collective/google-cli-common/config"

	"github.com/open-cli-collective/google-readwrite/internal/appidentity"
	"github.com/open-cli-collective/google-readwrite/internal/cmd/root"
)

func main() {
	// Register this CLI's identity before any config/keychain/auth call: it
	// stamps the config dir, keyring service (google-readwrite), env-var
	// prefixes, and scope set the shared google-cli-common library resolves
	// against — kept distinct from gro's so the two never collide.
	config.Register(appidentity.Identity())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	root.ExecuteContext(ctx)
}
