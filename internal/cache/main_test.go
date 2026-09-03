package cache

import (
	"os"
	"testing"

	"github.com/open-cli-collective/google-cli/internal/config"
)

// TestMain registers the canonical test identity so DirName-derived cache and
// config paths resolve before any cache test runs.
func TestMain(m *testing.M) {
	config.RegisterForTest()
	os.Exit(m.Run())
}
