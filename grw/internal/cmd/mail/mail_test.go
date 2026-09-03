package mail

import "testing"

// TestNewCommandComposition verifies grw's mail command exposes both the shared
// non-destructive leaves (from mailcmd) and grw's read-write additions, and
// that the write leaves are wired as subcommands.
func TestNewCommandComposition(t *testing.T) {
	cmd := NewCommand()

	names := map[string]bool{}
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}

	// grw's read-write additions
	for _, want := range []string{"delete", "restore", "folder", "filter"} {
		if !names[want] {
			t.Errorf("grw mail is missing the %q command", want)
		}
	}
	// a representative shared non-destructive leaf must still be present
	for _, want := range []string{"search", "archive", "label"} {
		if !names[want] {
			t.Errorf("grw mail is missing the shared %q command", want)
		}
	}
}

// TestFolderAndFilterSubcommands checks the lifecycle subcommands exist.
func TestFolderAndFilterSubcommands(t *testing.T) {
	find := func(parent string) map[string]bool {
		for _, sub := range NewCommand().Commands() {
			if sub.Name() == parent {
				got := map[string]bool{}
				for _, c := range sub.Commands() {
					got[c.Name()] = true
				}
				return got
			}
		}
		return nil
	}

	folder := find("folder")
	for _, want := range []string{"create", "rename", "rm"} {
		if !folder[want] {
			t.Errorf("grw mail folder missing %q", want)
		}
	}
	filter := find("filter")
	for _, want := range []string{"list", "create", "rm"} {
		if !filter[want] {
			t.Errorf("grw mail filter missing %q", want)
		}
	}
}
