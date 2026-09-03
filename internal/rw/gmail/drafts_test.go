package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gmailv1 "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func testDraftClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	service, err := gmailv1.NewService(context.Background(), option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return &Client{service: service, userID: "me"}
}

func TestGetDraft(t *testing.T) {
	t.Parallel()
	client := testDraftClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/gmail/v1/users/me/drafts/draft-1" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("format"); got != "full" {
			t.Errorf("format = %q, want full", got)
		}
		_, _ = fmt.Fprint(w, `{
  "id":"draft-1",
  "message":{
    "id":"message-1",
    "threadId":"thread-1",
    "payload":{
      "headers":[
        {"name":"From","value":"sender@example.com"},
        {"name":"To","value":"to@example.com"},
        {"name":"Cc","value":"cc@example.com"},
        {"name":"Bcc","value":"bcc@example.com"},
        {"name":"Subject","value":"Hello"}
      ],
      "parts":[
        {"filename":"report.pdf"},
        {"filename":"logo.png","headers":[{"name":"Content-Disposition","value":"inline; filename=logo.png"}]},
        {"parts":[{"headers":[{"name":"Content-Disposition","value":"attachment; filename=notes.txt"}]}]}
      ]
    }
  }
}`)
	})

	got, err := client.GetDraft(context.Background(), "draft-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "draft-1" || got.MessageID != "message-1" || got.ThreadID != "thread-1" ||
		got.From != "sender@example.com" || got.To != "to@example.com" || got.Cc != "cc@example.com" ||
		got.Bcc != "bcc@example.com" || got.Subject != "Hello" || got.AttachmentCount != 2 {
		t.Errorf("GetDraft() = %+v", got)
	}
}

func TestGetDraftWithoutMessageOrPayload(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"no message": `{"id":"draft-1"}`,
		"no payload": `{"id":"draft-1","message":{"id":"message-1","threadId":"thread-1"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			client := testDraftClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, body) })
			got, err := client.GetDraft(context.Background(), "draft-1")
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != "draft-1" || got.To != "" || got.AttachmentCount != 0 {
				t.Errorf("GetDraft() = %+v", got)
			}
		})
	}
}

func TestSendDraft(t *testing.T) {
	t.Parallel()
	client := testDraftClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/gmail/v1/users/me/drafts/send" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var draft gmailv1.Draft
		if err := json.NewDecoder(r.Body).Decode(&draft); err != nil {
			t.Fatal(err)
		}
		if draft.Id != "draft-1" {
			t.Errorf("draft ID = %q", draft.Id)
		}
		_, _ = fmt.Fprint(w, `{"id":"message-1","threadId":"thread-1","labelIds":["SENT"]}`)
	})

	got, err := client.SendDraft(context.Background(), "draft-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "message-1" || got.ThreadID != "thread-1" || len(got.LabelIDs) != 1 || got.LabelIDs[0] != "SENT" {
		t.Errorf("SendDraft() = %+v", got)
	}
}

func TestDraftAPIErrorsWrap(t *testing.T) {
	t.Parallel()
	client := testDraftClient(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no", http.StatusInternalServerError)
	})
	if _, err := client.GetDraft(context.Background(), "draft-1"); err == nil || !strings.Contains(err.Error(), "getting draft") {
		t.Errorf("GetDraft error = %v", err)
	}
	if _, err := client.SendDraft(context.Background(), "draft-1"); err == nil || !strings.Contains(err.Error(), "sending draft") {
		t.Errorf("SendDraft error = %v", err)
	}
}
