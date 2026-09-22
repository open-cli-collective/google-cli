package drive

import (
	"os"
	"testing"

	"github.com/open-cli-collective/google-cli/internal/config"
)

func TestMain(m *testing.M) {
	config.RegisterForTest()
	os.Exit(m.Run())
}
