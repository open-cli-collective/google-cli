package gro

import (
	"os"
	"testing"

	"github.com/open-cli-collective/google-cli/internal/config"
)

// TestMain registers gro's real identity before any test runs, mirroring what
// main does at startup, so config/keychain/auth paths and the scope set resolve.
func TestMain(m *testing.M) {
	config.Register(Identity())
	os.Exit(m.Run())
}
