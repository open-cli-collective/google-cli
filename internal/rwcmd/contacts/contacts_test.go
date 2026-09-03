package contacts

import "testing"

func TestNewCommandComposition(t *testing.T) {
	want := map[string]bool{
		"list": false, "search": false, "get": false, "groups": false, "add-to-group": false, "remove-from-group": false,
		"star": false, "unstar": false, "create": false, "update": false, "delete": false, "group": false,
	}
	for _, command := range NewCommand().Commands() {
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
