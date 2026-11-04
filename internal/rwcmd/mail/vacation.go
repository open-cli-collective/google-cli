package mail

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	gmailv1 "google.golang.org/api/gmail/v1"

	"github.com/open-cli-collective/google-cli/internal/sanitize"
)

func newVacationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vacation",
		Short: "Show, set, and turn off the vacation responder (out-of-office reply)",
	}
	cmd.AddCommand(newVacationShowCommand(), newVacationSetCommand(), newVacationOffCommand())
	return cmd
}

func newVacationShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the vacation responder settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			settings, err := client.GetVacation(cmd.Context())
			if err != nil {
				return err
			}
			printVacation(settings)
			return nil
		},
	}
}

func newVacationSetCommand() *cobra.Command {
	var subject, body, bodyFile, start, end string
	var contactsOnly, domainOnly, dryRun bool

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Turn on the vacation responder, replacing its current settings",
		Long: `Turn on the vacation responder, replacing its current settings.

Dates are YYYY-MM-DD in the local time zone, or RFC3339. A --start date begins at
the start of that day and an --end date runs through the end of that day.`,
		Example: `  grw mail vacation set --subject "Out of office" --body "Back on March 9." --start 2026-03-02 --end 2026-03-08 --dry-run
  grw mail vacation set --body-file reply.txt --end 2026-03-08 --domain-only`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if body != "" && bodyFile != "" {
				return errors.New("--body and --body-file are mutually exclusive")
			}
			if bodyFile != "" {
				data, err := os.ReadFile(bodyFile) // #nosec G304 -- file path is intentionally user-supplied via --body-file
				if err != nil {
					return fmt.Errorf("reading --body-file: %w", err)
				}
				body = string(data)
			}
			if strings.TrimSpace(subject) == "" && strings.TrimSpace(body) == "" {
				return errors.New("--subject or --body is required")
			}
			settings := &gmailv1.VacationSettings{
				EnableAutoReply:       true,
				ResponseSubject:       subject,
				ResponseBodyPlainText: body,
				RestrictToContacts:    contactsOnly,
				RestrictToDomain:      domainOnly,
				ForceSendFields:       []string{"EnableAutoReply"},
			}
			var err error
			if settings.StartTime, err = parseVacationTime(start, false); err != nil {
				return fmt.Errorf("invalid --start: %w", err)
			}
			if settings.EndTime, err = parseVacationTime(end, true); err != nil {
				return fmt.Errorf("invalid --end: %w", err)
			}
			if settings.StartTime != 0 && settings.EndTime != 0 && settings.StartTime >= settings.EndTime {
				return errors.New("--start must be before --end")
			}
			if dryRun {
				fmt.Println("[dry-run] Would set vacation responder:")
				printVacation(settings)
				return nil
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			updated, err := client.UpdateVacation(cmd.Context(), settings)
			if err != nil {
				return err
			}
			fmt.Println("Vacation responder updated:")
			printVacation(updated)
			return nil
		},
	}
	cmd.Flags().StringVar(&subject, "subject", "", "Subject line of the reply")
	cmd.Flags().StringVar(&body, "body", "", "Plain-text body of the reply")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read the plain-text body from this file")
	cmd.Flags().StringVar(&start, "start", "", "First day to send replies (YYYY-MM-DD or RFC3339)")
	cmd.Flags().StringVar(&end, "end", "", "Last day to send replies (YYYY-MM-DD or RFC3339)")
	cmd.Flags().BoolVar(&contactsOnly, "contacts-only", false, "Reply only to people in your contacts")
	cmd.Flags().BoolVar(&domainOnly, "domain-only", false, "Reply only to people in your Workspace domain")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func newVacationOffCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "off",
		Short: "Turn off the vacation responder, keeping its text and dates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dryRun {
				fmt.Println("[dry-run] Would turn off the vacation responder.")
				return nil
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			settings, err := client.GetVacation(cmd.Context())
			if err != nil {
				return err
			}
			settings.EnableAutoReply = false
			// The update replaces every field, and the API client omits false unless forced.
			settings.ForceSendFields = append(settings.ForceSendFields, "EnableAutoReply")
			if _, err := client.UpdateVacation(cmd.Context(), settings); err != nil {
				return err
			}
			fmt.Println("Vacation responder turned off.")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

// parseVacationTime returns epoch milliseconds, or 0 for an empty value. With
// endOfDay, a bare date resolves to the next local midnight because Gmail
// replies only to messages received before the end time.
func parseVacationTime(value string, endOfDay bool) (int64, error) {
	if value == "" {
		return 0, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.UnixMilli(), nil
	}
	t, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return 0, errors.New("expected YYYY-MM-DD or RFC3339")
	}
	if endOfDay {
		t = t.AddDate(0, 0, 1)
	}
	return t.UnixMilli(), nil
}

func printVacation(s *gmailv1.VacationSettings) {
	fmt.Printf("Enabled: %t\n", s.EnableAutoReply)
	fmt.Printf("Starts: %s\n", formatVacationTime(s.StartTime))
	fmt.Printf("Ends before: %s\n", formatVacationTime(s.EndTime))
	fmt.Printf("Contacts only: %t\n", s.RestrictToContacts)
	fmt.Printf("Domain only: %t\n", s.RestrictToDomain)
	fmt.Printf("Subject: %s\n", sanitize.Output(s.ResponseSubject))
	body := s.ResponseBodyPlainText
	if body == "" {
		body = s.ResponseBodyHtml
	}
	fmt.Printf("Body:\n%s\n", sanitize.Output(body))
}

func formatVacationTime(ms int64) string {
	if ms == 0 {
		return "(not set)"
	}
	return time.UnixMilli(ms).Local().Format("Mon, Jan 2, 2006 3:04 PM MST")
}
