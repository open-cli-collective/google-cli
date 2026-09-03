// Package main is the entry point for the grw (google-readwrite) CLI.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-cli-collective/google-cli/internal/app/grw"
	"github.com/open-cli-collective/google-cli/internal/config"
)

func main() {
	// Register this CLI's identity before any config/keychain/auth call: it
	// stamps the config dir, keyring service (google-readwrite), env-var prefixes,
	// and scope set — kept distinct from gro's so the two never collide.
	config.Register(grw.Identity())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	grw.ExecuteContext(ctx)
}
