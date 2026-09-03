// Package grw declares google-readwrite's CLI identity: the config/
// keyring directory name, default credential ref, product name, and the OAuth
// scope set this CLI requests. It is the single place that defines what makes
// this binary "grw" as opposed to gro.
//
// Unlike gro, grw is read-WRITE: its scopes permit filter management and
// permanent deletion, and its keyring namespace (google-readwrite/*) is
// deliberately separate from gro's (google-readonly/*) so the two tools can be
// used in isolation — e.g. handing an agent gro only, which structurally cannot
// destroy anything.
package grw

import (
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/people/v1"

	"github.com/open-cli-collective/google-cli/internal/config"
)

// Scopes is the OAuth scope set grw requests:
//   - gmail.modify: read + non-destructive organization (label, archive, move,
//     trash-via-label) and label lifecycle (create/rename/delete folders).
//   - gmail.settings.basic: create, list, and delete Gmail filters.
//   - https://mail.google.com/: permanent deletion (batchDelete), gated behind
//     `--permanent` and a typed confirmation. This is the broadest Gmail scope;
//     it is present only because Google offers no narrower delete scope.
var Scopes = []string{
	gmail.GmailModifyScope,
	gmail.GmailSettingsBasicScope,
	gmail.MailGoogleComScope,
	calendar.CalendarReadonlyScope,
	calendar.CalendarEventsScope,
	drive.DriveScope,
	drive.DriveReadonlyScope,
	drive.DriveMetadataScope,
	people.ContactsScope,
	people.UserinfoProfileScope,
}

// ScopeDescriptions maps each requested scope URL to a human-friendly
// description, shown by the init wizard and the scope-drift re-auth prompt.
var ScopeDescriptions = map[string]string{
	gmail.GmailModifyScope:         "Gmail Modify — read messages, plus label, archive, move, trash, and create/rename/delete labels (folders).",
	gmail.GmailSettingsBasicScope:  "Gmail Settings (basic) — create, list, and delete filters.",
	gmail.MailGoogleComScope:       "Gmail Full Access — required for permanent deletion (gated behind --permanent and a typed confirmation).",
	calendar.CalendarReadonlyScope: "Calendar Read-Only — read calendars and events.",
	calendar.CalendarEventsScope:   "Calendar Events — create, update, and delete events.",
	drive.DriveScope:               "Drive Full Access — upload files, create folders, rename, move, trash, restore, and permanently delete files.",
	drive.DriveReadonlyScope:       "Drive Read-Only — read files and metadata.",
	drive.DriveMetadataScope:       "Drive Metadata — read and update file metadata (star/unstar).",
	people.ContactsScope:           "Contacts — read, create, update, and delete contacts and contact groups.",
	people.UserinfoProfileScope:    "Basic Profile — read the authenticated user's name and email for `me`.",
}

// Identity is grw's config.Identity, registered once at startup.
func Identity() config.Identity {
	return config.Identity{
		DirName:           "google-readwrite",
		DefaultRef:        "google-readwrite/default",
		ProductName:       "grw",
		Scopes:            Scopes,
		ScopeDescriptions: ScopeDescriptions,
		// grw can reuse gro's OAuth client JSON during `grw init`, so a gro
		// user needs no second paste. Tokens stay separate (own keyring).
		SiblingDirNames: []string{"google-readonly"},
	}
}
