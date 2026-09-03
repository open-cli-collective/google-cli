package initcmd

import (
	"os"
	"testing"

	"github.com/open-cli-collective/google-cli/internal/config"
)

// TestMain registers the canonical test identity (which includes the profile
// scope, so the People-verify path is exercised) before any init test runs.
func TestMain(m *testing.M) {
	config.RegisterForTest()
	os.Exit(m.Run())
}
