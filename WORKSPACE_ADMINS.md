# OAuth setup for `gro` and `grw`

This guide covers OAuth clients for personal Google accounts and Google Workspace organizations. A CLI profile selects a saved account token; an OAuth client identifies the app shown during consent. Each profile can use its own OAuth client, and each CLI stores its token separately.

## Choose an audience

- **Personal Google account:** use an **External** app. Testing is the simplest initial setup: add your account as a test user and expect a testing warning. Because these CLIs request scopes beyond basic profile information, refresh tokens for an External app in Testing expire after seven days. An External app in Production avoids this Testing-specific limit, but unverified warnings and the 100-user cap can still apply. Personal-only apps or apps for a few personally known users may qualify for a verification exemption; review Google's current requirements before publishing.
- **Google Workspace organization:** a Workspace admin can create an **Internal** app for accounts in the project's organization. This avoids the External-app verification path, while Workspace admin policies still apply. The project must belong to the organization for Internal to be available.

## Required APIs and scopes

Enable Gmail API, Google Calendar API, People API, and Google Drive API for the APIs your CLI uses. If both binaries will be used, enable all four and declare the union of scopes below in Google Auth Platform's Data Access settings.

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

## Personal account: External app

1. In [Google Cloud Console](https://console.cloud.google.com/), create or select a project you control.
2. Enable the required APIs above.
3. In Google Auth Platform's **Branding** page, enter the app name, user support email, and developer contact email. Under **Audience**, choose **External**. For the simplest initial setup, leave publishing status at **Testing** and add your Google account under **Test users**.
4. Add the scopes needed by your CLI under **Data Access**.
5. Create an OAuth client with application type **Desktop app** and download its JSON file.

The testing warning is expected, and reauthorization is required seven days after each authorization. Moving an External app to Production removes this Testing-specific token limit, but other token expiration rules still apply. Publishing and verification are separate steps: personal-only use or a few personally known users may qualify for a verification exemption, while unverified warnings and the 100-user cap can remain. If **Publish app** is unavailable, complete the Branding requirements shown in Google Auth Platform, including any required app homepage, privacy policy, and terms links. Review Google's current [personal-use and branding requirements](https://developers.google.com/identity/protocols/oauth2/production-readiness/brand-verification#personal-use), [OAuth app states](https://developers.google.com/identity/protocols/oauth2/production-readiness/overview), [Gmail scope requirements](https://developers.google.com/workspace/gmail/api/auth/scopes), and [restricted-scope verification](https://developers.google.com/identity/protocols/oauth2/production-readiness/restricted-scope-verification) before distributing an app more broadly.

Import the client for the profile you want to authorize:

```bash
grw --profile personal init --credentials-file /path/to/oauth-client.json
grw --profile personal me

# If you also use gro, import the same client for its separate profile/token:
gro --profile personal init --credentials-file /path/to/oauth-client.json
```

## Workspace organization: Internal app

You need a Google Workspace administrator account, permission to create a Google Cloud project under the organization, and a controlled channel for distributing the OAuth client JSON.

1. In Google Cloud Console, create or select a project under the Workspace organization.
2. Enable the required APIs above.
3. In Google Auth Platform, set the audience to **Internal**, provide the app and support details, and add the union of scopes above under **Data Access**.
4. Create an OAuth client with application type **Desktop app** and download its JSON.

Distribute the JSON through an access-controlled vault, MDM, or internal file service. Each CLI still asks the user for consent and keeps its token in a separate keyring namespace. Use `gro init --credentials-file /path/to/oauth-client.json` and `grw init --credentials-file /path/to/oauth-client.json`, adding `--profile <name>` before `init` when authorizing named profiles.

## Profile-specific OAuth clients

`--credentials-file` imports the JSON for the selected CLI profile and stores it in a profile-specific managed file. For example, `grw --profile work init --credentials-file ...` associates the client with `google-readwrite/work`; it does not replace the legacy shared client or another profile's client. Run a matching `gro` command to import that JSON for a `gro` profile. The automatic sibling-client lookup applies to the legacy shared `oauth_client.json`; profile-specific imports are not discovered by the other binary.

If the selected profile already has a token, re-importing the same OAuth client ID is allowed. Importing a different client ID is refused so the existing token is not used with the wrong app. Use a new profile, or clear only the selected profile's token with the same `--profile` selector before authorizing a different client.

## Administration and troubleshooting

- If a personal-account user cannot consent while the app is in Testing, confirm that their Google account is listed under **Test users**.
- If Google returns `org_internal`, the OAuth app is restricted to the Workspace organization that owns the project. Use an External app/client for a personal Google account; CLI `--profile` names do not change an OAuth app's audience.
- If Workspace access is blocked, use Admin Console → Security → Access and data control → API controls to trust or allow the OAuth client for the intended organizational units or groups.
- If an API reports `SERVICE_DISABLED`, enable the named API in the Cloud project and wait for propagation.
- Revoke a user's grant through Google Account permissions, or block the client in Admin Console to revoke organization access.
- Rotate and redistribute the client JSON if its distribution boundary is breached; test the rotation with one user first.

See Google's guides for [app audiences and test users](https://support.google.com/cloud/answer/15549945), [OAuth 2.0 for desktop apps](https://developers.google.com/identity/protocols/oauth2/native-app), and [refresh-token expiration](https://developers.google.com/identity/protocols/oauth2).
