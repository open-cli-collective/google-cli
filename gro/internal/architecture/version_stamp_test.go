package architecture

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// versionPkg is the package whose ldflags -X stamps set gro's version. Go
// silently ignores -X against a package the binary doesn't contain, so when
// this package once moved without the ldflags following it, every release
// shipped reporting "gro dev (commit: unknown, built: unknown)". If it moves
// again, this test fails until the ldflags move with it.
const versionPkg = "github.com/open-cli-collective/google-cli/common/version"

// ldflagXRe matches an ldflags -X target's package path (the part before the
// final .Var=value).
var ldflagXRe = regexp.MustCompile(`-X ([\w./-]+)\.(?:Version|Commit|Date)=`)

func TestLdflagsStampTheLinkedVersionPackage(t *testing.T) {
	root := repoRoot(t)
	for _, file := range []string{"Makefile"} {
		data, err := os.ReadFile(filepath.Join(root, file)) //nolint:gosec // repo-local test input
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		matches := ldflagXRe.FindAllStringSubmatch(string(data), -1)
		if len(matches) == 0 {
			t.Fatalf("%s: no -X version ldflags found — version stamping removed?", file)
		}
		for _, m := range matches {
			if m[1] != versionPkg {
				t.Errorf("%s stamps %q, but the linked version package is %q — -X against a package the binary doesn't contain is silently ignored and releases report \"dev\"", file, m[1], versionPkg)
			}
		}
	}
}

// repoRoot walks up from the test's working directory to the go.mod dir.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test dir")
		}
		dir = parent
	}
}
