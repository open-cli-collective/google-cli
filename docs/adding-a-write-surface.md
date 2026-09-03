# Adding a write surface

Add writes as an extension of an existing read domain; do not fork its read commands.

## 1. Extend the API client

Create `internal/rw/<domain>`. Its exported `Client` embeds `*internal/api/<domain>.Client`, and its constructor creates the read client before returning the wrapper. Add only methods needed by the write surface, accept `context.Context` first, and wrap API errors with `%w`.

When command-facing DTOs differ from Google API types, keep conversion helpers beside the boundary and test both directions. A DTO converted to the API form and back must preserve every supported field.

## 2. Extend the command

Create `internal/rwcmd/<domain>` with:

- An exported client interface embedding the interface from `internal/cmd/<domain>` and adding write methods.
- A `ClientFactory` defaulting to `internal/rw/<domain>.NewClient`.
- `NewCommand()` calling `internal/cmd/<domain>.NewCommand()` and attaching only new leaves.
- Function-field mocks and focused handler tests.

Add the read/write constructors to `writeCommandPairs` in `internal/architecture/architecture_test.go`.

## 3. Make mutations safe

Every added mutating leaf declares `--dry-run` (`-n`) and performs no client construction or API call while previewing. Read verbs named `list`, `get`, `show`, or `search` are exempt. If a leaf supports irreversible `--permanent` behavior, require `--yes` alongside it and keep a recoverable operation as the default.

Resource leaves remain text-only. Do not add `--json`.

## 4. Support bulk IDs consistently

For message/resource mutations, accept exactly one source:

- Positional IDs.
- `--stdin` for newline-delimited IDs.
- `--query` resolved through the domain client.

Read commands that feed pipelines expose `--ids`. Reuse `internal/bulk.ResolveIDs` and `bulk.Result` instead of implementing input arbitration or result text again.

## 5. Add scopes and compose `grw`

Add the minimum required scopes and descriptions to `internal/app/grw/identity.go`, then add each scope to the known write allowlist in the architecture tests. The existing read scope for that service must remain present. Register the extended domain command in `internal/app/grw/root.go`; do not register it in `gro`.

## 6. Verify

Add API-client tests, command success/error tests, dry-run tests, confirmation tests where applicable, and DTO round-trip tests. Then run:

```bash
make check
./bin/gro --help
./bin/grw --help
```

Confirm the `gro` tree is unchanged, the `grw` tree contains all read leaves plus the new leaves, and each mutation previews without credentials.
