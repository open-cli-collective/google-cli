package auth

import (
	"os"
	"testing"

	"github.com/open-cli-collective/google-cli-common/config"
)

// TestMain registers the canonical test identity so config.Scopes(),
// config.ProductName(), and the DirName-derived paths are populated before any
// auth test runs.
func TestMain(m *testing.M) {
	config.RegisterForTest()
	os.Exit(m.Run())
}
