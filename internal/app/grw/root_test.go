package grw

import "testing"

func TestRootCommands(t *testing.T) {
	want := map[string]bool{"contacts": false, "me": false, "profiles": false}
	for _, command := range rootCmd.Commands() {
		if _, ok := want[command.Name()]; ok {
			want[command.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing %s command", name)
		}
	}
}
