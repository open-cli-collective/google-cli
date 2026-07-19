package mail

import (
	"bytes"
	"context"

	"github.com/spf13/cobra"
)

// withMockClient swaps the package ClientFactory to return m for the duration
// of f, then restores it.
func withMockClient(m WriteClient, f func()) {
	orig := ClientFactory
	ClientFactory = func(_ context.Context) (WriteClient, error) { return m, nil }
	defer func() { ClientFactory = orig }()
	f()
}

// runCmd executes a freshly-built leaf command with args and returns combined
// stdout+stderr the command wrote via cmd.OutOrStdout(), plus any error. The
// bulk.Result / fmt.Printf output goes to os.Stdout, so tests that assert on
// that use captureStdout instead.
func runCmd(cmd *cobra.Command, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}
