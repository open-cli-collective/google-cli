# Adding a read domain

Use this checklist for a new Google API read surface. Keep the domain unavailable until its scope, client, commands, composition, and tests land together.

## 1. Add the scope

Add the narrowest adequate scope and description to `internal/app/gro/identity.go`. Add it to the non-destructive allowlist in `internal/architecture/architecture_test.go`. If `grw` will expose the domain too, add the same scope and description to `internal/app/grw/identity.go`; `TestGrwScopesCoverGroPerService` requires per-service coverage.

## 2. Add the API client

Create `internal/api/<domain>` with a concrete `Client`, `NewClient(ctx context.Context)`, I/O methods taking context first, error wrapping, parsing helpers, and focused unit tests. API code must not import command packages.

## 3. Add the command package

Create `internal/cmd/<domain>` with:

- An exported consumer-owned interface ending in `Client`.
- `ClientFactory`, defaulting to the API package's `NewClient`.
- An exported `NewCommand() *cobra.Command` that registers the domain leaves.
- One file per substantial leaf and text output helpers.
- No `--json` flag on resource leaves.

Use `cmd.Context()` in handlers and pass it to both `ClientFactory` and client methods.

## 4. Add tests

Add a function-field mock with a compile-time interface assertion, factory replacement via `testutil.WithFactory`, success and failure handler tests, and fixtures in `internal/testutil` only when multiple packages benefit. Add the domain to `domainPackages` and `domainCommands()` in `internal/architecture/architecture_test.go`.

## 5. Compose both binaries

Register the command with the `gro` root in `internal/app/gro/root.go`. Register it with `grw` in `internal/app/grw/root.go` as well so both binaries expose the read surface. If the domain also needs writes, compose its extended command there instead and follow [adding a write surface](adding-a-write-surface.md).

## 6. Verify

```bash
make check
./bin/gro --help
./bin/grw --help
```

Confirm the new domain appears in both help trees, resource leaves reject `--json`, and the new scope descriptions appear during setup.
