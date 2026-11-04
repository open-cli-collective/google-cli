package gmail

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	gmailv1 "google.golang.org/api/gmail/v1"
)

const vacationPath = "/gmail/v1/users/me/settings/vacation"

func TestGetVacation(t *testing.T) {
	t.Parallel()
	client := testDraftClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != vacationPath {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"enableAutoReply":true,"responseSubject":"Away","endTime":"42"}`)
	})
	got, err := client.GetVacation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.EnableAutoReply || got.ResponseSubject != "Away" || got.EndTime != 42 {
		t.Errorf("settings = %+v", got)
	}
}

func TestUpdateVacation_SendsForcedFalse(t *testing.T) {
	t.Parallel()
	client := testDraftClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != vacationPath {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(string(body), `"enableAutoReply":false`) {
			t.Errorf("body = %s", body)
		}
		_, _ = fmt.Fprint(w, `{"enableAutoReply":false}`)
	})
	settings := &gmailv1.VacationSettings{ForceSendFields: []string{"EnableAutoReply"}}
	if _, err := client.UpdateVacation(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
}

func TestVacationAPIErrorsWrap(t *testing.T) {
	t.Parallel()
	client := testDraftClient(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no", http.StatusInternalServerError)
	})
	if _, err := client.GetVacation(context.Background()); err == nil || !strings.Contains(err.Error(), "getting vacation responder") {
		t.Errorf("GetVacation error = %v", err)
	}
	if _, err := client.UpdateVacation(context.Background(), &gmailv1.VacationSettings{}); err == nil || !strings.Contains(err.Error(), "updating vacation responder") {
		t.Errorf("UpdateVacation error = %v", err)
	}
}
