package keychain

import (
	"os"
	"testing"

	"github.com/open-cli-collective/google-cli/common/config"
)

// TestMain registers the canonical test identity up front so any keychain test
// that reads DirName-derived paths before its first credtest.Setup call still
// sees a valid identity. credtest.Setup keeps whatever is already registered.
func TestMain(m *testing.M) {
	config.RegisterForTest()
	os.Exit(m.Run())
}
