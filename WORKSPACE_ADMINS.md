# Google CLI for Workspace administrators

This guide sets up one organization-managed desktop OAuth client that employees can use with both `gro` and `grw`. Each user still grants consent, and each binary stores its token under a separate identity.

## Before you begin

You need:

- A Google Workspace administrator account.
- Permission to create a Google Cloud project under the Workspace organization.
- A controlled channel for distributing the downloaded OAuth client JSON.

An Internal audience limits consent to accounts in the Workspace organization and avoids the External-app verification path. If the project is not attached to the organization, the Internal option will not appear.

## Create the project and client

1. In [Google Cloud Console](https://console.cloud.google.com/), create or select a project under the Workspace organization.
2. Enable Gmail API, Google Calendar API, People API, and Google Drive API.
3. In Google Auth Platform (or OAuth consent screen), set the audience to **Internal** and provide the application/support details.
4. Add the union of scopes below under Data Access.
5. Create an OAuth client with application type **Desktop app** and download its JSON.

One desktop client is sufficient because the binaries request scopes at authorization time. Users authorize each binary independently.

## Scope inventory

### `gro`

```text
https://www.googleapis.com/auth/gmail.modify
https://www.googleapis.com/auth/calendar.readonly
https://www.googleapis.com/auth/calendar.events
https://www.googleapis.com/auth/contacts
https://www.googleapis.com/auth/userinfo.profile
https://www.googleapis.com/auth/drive.readonly
https://www.googleapis.com/auth/drive.metadata
```

These scopes support Gmail read/organization and draft creation without sending, Calendar reads plus RSVP/color, Contacts reads plus group/star changes, profile identity, and Drive reads plus star metadata. The `gro` command graph and architecture tests restrict actual behavior even where Google's scope description is broader.

### `grw`

```text
https://www.googleapis.com/auth/gmail.modify
https://www.googleapis.com/auth/gmail.settings.basic
https://mail.google.com/
https://www.googleapis.com/auth/calendar.readonly
https://www.googleapis.com/auth/calendar.events
https://www.googleapis.com/auth/contacts
https://www.googleapis.com/auth/userinfo.profile
https://www.googleapis.com/auth/drive
https://www.googleapis.com/auth/drive.readonly
https://www.googleapis.com/auth/drive.metadata
```

Gmail settings access supports filters. The broad mail scope is required for permanent deletion; the command defaults to recoverable Trash and gates permanent deletion behind `--permanent --yes`. Calendar scopes support reading and mutating events, the Contacts scope supports reading and mutating contacts and groups, the Drive scopes support reading, uploading, organizing, trashing, restoring, and permanently deleting files, and the profile scope supports `grw me`.

## Distribute and verify

Distribute the OAuth client JSON through an access-controlled vault, MDM, or internal file service. Desktop OAuth client material identifies the application; user refresh tokens are separate secrets and must remain in each user's selected keyring backend.

Each user can give the JSON to the setup wizard from a file, clipboard, or terminal paste:

```bash
gro init --credentials-file /path/to/oauth-client.json
gro me

grw init --credentials-file /path/to/oauth-client.json
grw mail list --max 1
```

If one binary is already configured, the other setup wizard can discover and reuse its OAuth client JSON. It does not reuse the token: the user sees a separate consent flow for the other identity and scope set.

## Administration and troubleshooting

- If users see an unverified-app warning, confirm the project belongs to the Workspace organization and the audience is Internal.
- If access is blocked, use Admin Console → Security → Access and data control → API controls to trust or allow the OAuth client for the intended organizational units or groups.
- If an API reports `SERVICE_DISABLED`, enable the named API in the Cloud project and wait for propagation.
- Revoke a user's grant through Google Account permissions, or block the client in Admin Console to revoke organization access.
- Rotate and redistribute the client JSON if its distribution boundary is breached; test the rotation with one user first.

For personal accounts or cross-organization distribution, create an External-audience client and follow Google's current verification requirements. See Google's guidance on [OAuth app audience](https://support.google.com/cloud/answer/15549945), [when verification is not needed](https://support.google.com/cloud/answer/13464323), and [installed applications](https://developers.google.com/identity/protocols/oauth2/native-app).
