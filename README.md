# Google CLI

This repository builds two Google command-line tools from one Go module:

- `gro` reads and organizes Gmail, Calendar, Contacts, and Drive data without exposing destructive operations.
- `grw` is the read-write Google tool. Its mail, calendar, contacts, and Drive command trees extend `gro` with delete and restore, folder management, filters, uploads, and resource creation, updates, moves, and deletion.

Both tools use the same configuration and keyring machinery, but register separate identities. Their configuration, tokens, environment variables, and keyring entries do not collide. `gro` requests only the scopes needed for its non-destructive surface; `grw` adds Gmail settings, permanent-delete access, Calendar event writes, Contacts writes, and full Drive access.

## Safety model

`gro` has no command path to send, trash, restore, or permanently delete data. Its compiled dependency graph cannot reach packages under `internal/rw` or `internal/rwcmd`. Tests in [`internal/architecture`](internal/architecture) enforce that boundary, allowlist its OAuth scopes, and scan its production graph for forbidden destructive API calls.

`grw mail delete` moves messages to Trash by default. Permanent deletion requires both `--permanent` and `--yes`. Every added write leaf other than read verbs supports `--dry-run` (`-n`).

`grw contacts delete` and `grw contacts group rm` ask for a typed confirmation unless `--yes` is given. Deleting a group never deletes its member contacts, and system groups such as `myContacts` and `starred` are refused.

## Install

### Homebrew

```bash
brew install open-cli-collective/tap/gro
brew install open-cli-collective/tap/grw
```

The existing tap aliases also resolve:

```bash
brew install open-cli-collective/tap/google-readonly
brew install open-cli-collective/tap/google-readwrite
```

Package registries know the tools as `google-readonly` and `google-readwrite`; the binaries are `gro` and `grw`.

### Chocolatey

```powershell
choco install google-readonly
choco install google-readwrite
```

### WinGet

```powershell
winget install OpenCLICollective.google-readonly
winget install OpenCLICollective.google-readwrite
```

### Release archives

Download the `gro` or `grw` archive for your operating system and architecture from [GitHub Releases](https://github.com/open-cli-collective/google-cli/releases), extract it, and place the binary on your `PATH`.

## Quick start

```bash
gro init
gro me
gro mail list
gro calendar today

grw init
grw mail send <draft-id> --dry-run
grw mail delete --query "older_than:1y" --dry-run
grw calendar create --summary "Planning" --start 2026-10-01 --dry-run
grw contacts create --given-name Test --email t@example.com --dry-run
grw drive trash --query "name contains 'old'" --dry-run
```

One desktop OAuth client can be used by both tools, but each tool asks for consent and stores its token under its own identity. Google Workspace administrators should start with [`WORKSPACE_ADMINS.md`](WORKSPACE_ADMINS.md).

## Profiles

Each tool has its own profile namespace. Use a bare profile name with the global
`--profile` shorthand, or pass the full credential reference with `--ref`:

```bash
gro --profile work mail list
grw --profile work calendar today
gro --ref google-readonly/work mail list
```

The selector precedence is explicit flag (`--profile` or `--ref`), credential
reference environment variable, saved `credential_ref`, then the built-in
`default` profile. For environment selection, use
`GOOGLE_READONLY_CREDENTIAL_REF` with `gro` or
`GOOGLE_READWRITE_CREDENTIAL_REF` with `grw`, for example:

```bash
GOOGLE_READONLY_CREDENTIAL_REF=google-readonly/work gro mail list
```

`--profile` and `--ref` cannot be used together. To add an account without
changing the active profile, run `gro --profile work init` (or the equivalent
`grw` command). Inspect and manage profiles with:

```bash
gro profiles list
gro profiles rename old-name new-name
```

Renaming moves the stored credentials without re-authentication, updates the
saved active profile when necessary, and refuses a destination that already
has credentials. If saving the active-profile update fails, the copied
destination is removed when rollback succeeds while the source remains, so the
command can be retried. If rollback also fails, the command reports that the
destination may remain.

## Documentation

- [Development](docs/development.md)
- [Architecture](docs/architecture.md)
- [Golden principles](docs/golden-principles.md)
- [Add a read domain](docs/adding-a-domain.md)
- [Add a write surface](docs/adding-a-write-surface.md)
- [Standards index](STANDARDS.md)

## License

[MIT](LICENSE)
