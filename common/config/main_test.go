package config

import (
	"os"
	"testing"
)

// TestMain registers the canonical test identity before any test runs, so
// DirName / DefaultCredentialRef and the scope set resolve the way the ported
// google-readonly tests expect. Production CLIs call Register from main; the
// library itself has no built-in identity.
func TestMain(m *testing.M) {
	RegisterForTest()
	os.Exit(m.Run())
}
