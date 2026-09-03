package calendar

import "testing"

func TestNewCommandComposition(t *testing.T) {
	want := map[string]bool{"list": false, "events": false, "today": false, "week": false, "get": false, "rsvp": false, "color": false, "create": false, "update": false, "delete": false}
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
