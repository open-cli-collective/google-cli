package mail

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	gmailv1 "google.golang.org/api/gmail/v1"

	"github.com/open-cli-collective/google-cli/internal/sanitize"
)

func newVacationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vacation",
		Short: "Get, update, enable, and disable the vacation responder (out-of-office reply)",
	}
	cmd.AddCommand(newVacationGetCommand(), newVacationUpdateCommand(),
		newVacationToggleCommand("enable", true), newVacationToggleCommand("disable", false))
	return cmd
}

func newVacationGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get the vacation responder settings",
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

func newVacationUpdateCommand() *cobra.Command {
	var subject, body, file, start, end string
	var contactsOnly, domainOnly, dryRun bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Change vacation responder settings, leaving other settings and its on/off state as they are",
		Long: `Change vacation responder settings. Only the flags you pass change, and the
responder stays on or off; use 'enable' to turn it on.

Dates are YYYY-MM-DD in the local time zone, or RFC3339. A --start date begins at
the start of that day and an --end date runs through the end of that day. An
empty value clears a date. Setting --body or --file replaces any existing body.`,
		Example: `  grw mail vacation update --subject "Out of office" --body "Back on March 9." --start 2026-03-02 --end 2026-03-08 --dry-run
  grw mail vacation update --file reply.txt --end 2026-03-08 --domain-only`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags := cmd.Flags()
			if !flags.Changed("subject") && !flags.Changed("body") && !flags.Changed("file") && !flags.Changed("start") &&
				!flags.Changed("end") && !flags.Changed("contacts-only") && !flags.Changed("domain-only") {
				return errors.New("at least one setting is required (--subject/--body/--file/--start/--end/--contacts-only/--domain-only)")
			}
			if flags.Changed("body") && flags.Changed("file") {
				return errors.New("--body and --file are mutually exclusive")
			}
			if flags.Changed("file") {
				data, err := os.ReadFile(file) // #nosec G304 -- file path is intentionally user-supplied via --file
				if err != nil {
					return fmt.Errorf("reading --file: %w", err)
				}
				body = string(data)
			}
			startMillis, err := parseVacationTime(start, false)
			if err != nil {
				return fmt.Errorf("invalid --start: %w", err)
			}
			endMillis, err := parseVacationTime(end, true)
			if err != nil {
				return fmt.Errorf("invalid --end: %w", err)
			}
			if dryRun {
				fmt.Println("[dry-run] Would update vacation responder:")
				flags.Visit(func(f *pflag.Flag) {
					switch f.Name {
					case "dry-run":
					case "file":
						fmt.Printf("body:\n%s\n", sanitize.Output(body))
					default:
						fmt.Printf("%s: %s\n", sanitize.Output(f.Name), sanitize.Output(f.Value.String()))
					}
				})
				return nil
			}

			ctx := cmd.Context()
			client, err := newWriteClient(ctx)
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			settings, err := client.GetVacation(ctx)
			if err != nil {
				return err
			}
			if flags.Changed("subject") {
				settings.ResponseSubject = subject
			}
			if flags.Changed("body") || flags.Changed("file") {
				// Gmail sends the HTML body when both are present, so a stale one must go.
				settings.ResponseBodyPlainText = body
				settings.ResponseBodyHtml = ""
			}
			if flags.Changed("start") {
				settings.StartTime = startMillis
			}
			if flags.Changed("end") {
				settings.EndTime = endMillis
			}
			if flags.Changed("contacts-only") {
				settings.RestrictToContacts = contactsOnly
			}
			if flags.Changed("domain-only") {
				settings.RestrictToDomain = domainOnly
			}
			if settings.StartTime != 0 && settings.EndTime != 0 && settings.StartTime >= settings.EndTime {
				return errors.New("the start time must be before the end time")
			}
			return writeVacation(ctx, client, settings, "Vacation responder updated:")
		},
	}
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "Subject line of the reply")
	cmd.Flags().StringVar(&body, "body", "", "Plain-text body of the reply")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Read the plain-text body from this file")
	cmd.Flags().StringVar(&start, "start", "", "First day to send replies (YYYY-MM-DD or RFC3339)")
	cmd.Flags().StringVar(&end, "end", "", "Last day to send replies (YYYY-MM-DD or RFC3339)")
	cmd.Flags().BoolVar(&contactsOnly, "contacts-only", false, "Reply only to people in your contacts")
	cmd.Flags().BoolVar(&domainOnly, "domain-only", false, "Reply only to people in your Workspace domain")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func newVacationToggleCommand(use string, enabled bool) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   use,
		Short: "Turn the vacation responder " + onOff(enabled) + ", keeping its text and dates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dryRun {
				fmt.Printf("[dry-run] Would turn the vacation responder %s.\n", onOff(enabled))
				return nil
			}
			ctx := cmd.Context()
			client, err := newWriteClient(ctx)
			if err != nil {
				return fmt.Errorf("creating Gmail client: %w", err)
			}
			settings, err := client.GetVacation(ctx)
			if err != nil {
				return err
			}
			settings.EnableAutoReply = enabled
			return writeVacation(ctx, client, settings, "Vacation responder turned "+onOff(enabled)+":")
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}

func onOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

// writeVacation sends the full settings and prints the result. The update
// replaces every field, and the API client omits false values unless forced.
func writeVacation(ctx context.Context, client WriteClient, settings *gmailv1.VacationSettings, heading string) error {
	settings.ForceSendFields = []string{"EnableAutoReply", "RestrictToContacts", "RestrictToDomain"}
	updated, err := client.UpdateVacation(ctx, settings)
	if err != nil {
		return err
	}
	fmt.Println(heading)
	printVacation(updated)
	return nil
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
	// Gmail sends the HTML body when both are present.
	body := s.ResponseBodyHtml
	if body == "" {
		body = s.ResponseBodyPlainText
	}
	fmt.Printf("Body:\n%s\n", sanitize.Output(body))
}

func formatVacationTime(ms int64) string {
	if ms == 0 {
		return "-"
	}
	return time.UnixMilli(ms).Local().Format(time.RFC3339)
}
