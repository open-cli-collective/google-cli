# google-readwrite (`grw`)

A read-write command-line interface for Gmail cleanup and organization. `grw` is
the read-write companion to [`gro`](https://github.com/open-cli-collective/google-readonly):
it reads and organizes mail the same way, and adds the operations `gro`
deliberately cannot perform.

## What it adds over `gro`

| Command | What it does |
| --- | --- |
| `grw mail delete` | Move messages to Trash (default). `--permanent` erases them irreversibly, behind a typed confirmation and the broad `mail.google.com` scope. |
| `grw mail folder create/rename/rm` | Manage labels as folders. `"Receipts/2026"` nests a subfolder. |
| `grw mail filter list/create/rm` | Manage Gmail filters — match on `--from/--to/--subject/--query/--has-attachment`, act with `--add-label/--archive/--mark-read/--star/--trash`. |

Everything `gro` does — `search`, `read`, `archive`, `label`, `move`, `star`,
`mark-read`, drafts — is available under `grw mail` too, so a full cleanup
workflow lives in one tool.

## Install

```bash
brew install open-cli-collective/tap/google-readwrite
grw init
```

`grw init` reuses an existing `gro` OAuth client automatically (no re-paste); you
grant consent once for grw's scopes.

## Isolation and safety

`grw` stores its credentials in a **separate keyring namespace**
(`google-readwrite/*`) from `gro` (`google-readonly/*`). Because `gro` is
structurally incapable of deleting, sending, or trashing anything, you can hand
an agent `gro` alone and keep `grw` gated to contexts you trust. The two never
collide.

Delete defaults to Trash (recoverable ~30 days). Permanent deletion requires an
explicit `--permanent` flag **and** a typed confirmation.

## Development

```bash
make check   # tidy + lint + test (race) + build
make install # build and install grw locally
```

`grw` is a thin CLI built on
[`google-cli-common`](https://github.com/open-cli-collective/google-cli-common);
almost all logic (OAuth, the Gmail client, rendering, the shared command
surface) lives there. Family-wide standards are in
[`cli-common`](https://github.com/open-cli-collective/cli-common/tree/main/docs).
