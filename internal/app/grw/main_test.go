package grw

import (
	"os"
	"testing"

	"github.com/open-cli-collective/google-cli/internal/config"
)

func TestMain(m *testing.M) {
	config.Register(Identity())
	os.Exit(m.Run())
}
