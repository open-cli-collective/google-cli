package gmail

import (
	"context"
	"fmt"
	"strings"

	gmailv1 "google.golang.org/api/gmail/v1"

	gmailapi "github.com/open-cli-collective/google-cli/internal/api/gmail"
)

// GetDraft returns the headers and attachment count needed to preview a draft
// before sending it. It asks for the full format because the metadata format
// omits payload parts, which would hide every attachment from the preview.
func (c *Client) GetDraft(ctx context.Context, draftID string) (*gmailapi.DraftSummary, error) {
	draft, err := c.service.Users.Drafts.Get(c.userID, draftID).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting draft: %w", err)
	}

	out := &gmailapi.DraftSummary{ID: draft.Id}
	if draft.Message == nil {
		return out, nil
	}
	out.MessageID = draft.Message.Id
	out.ThreadID = draft.Message.ThreadId
	if draft.Message.Payload == nil {
		return out, nil
	}
	for _, header := range draft.Message.Payload.Headers {
		switch strings.ToLower(header.Name) {
		case "to":
			out.To = header.Value
		case "cc":
			out.Cc = header.Value
		case "bcc":
			out.Bcc = header.Value
		case "subject":
			out.Subject = header.Value
		case "from":
			out.From = header.Value
		}
	}
	out.AttachmentCount = countAttachments(draft.Message.Payload)
	return out, nil
}

// SendDraft sends an existing Gmail draft.
func (c *Client) SendDraft(ctx context.Context, draftID string) (*gmailapi.SentResult, error) {
	message, err := c.service.Users.Drafts.Send(c.userID, &gmailv1.Draft{Id: draftID}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("sending draft: %w", err)
	}
	return &gmailapi.SentResult{ID: message.Id, ThreadID: message.ThreadId, LabelIDs: message.LabelIds}, nil
}

// countAttachments counts the parts a recipient would see as attachments: any
// part whose Content-Disposition is "attachment", plus named parts without a
// disposition. Named parts marked "inline" (embedded images, signatures) are
// not counted.
func countAttachments(part *gmailv1.MessagePart) int {
	count := 0
	switch disposition(part) {
	case "attachment":
		count++
	case "":
		if part.Filename != "" {
			count++
		}
	}
	for _, child := range part.Parts {
		count += countAttachments(child)
	}
	return count
}

func disposition(part *gmailv1.MessagePart) string {
	for _, header := range part.Headers {
		if strings.EqualFold(header.Name, "content-disposition") {
			value, _, _ := strings.Cut(header.Value, ";")
			return strings.ToLower(strings.TrimSpace(value))
		}
	}
	return ""
}
