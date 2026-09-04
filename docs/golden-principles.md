# Golden principles

These rules keep the two binaries mechanically distinct. Structural tests named below run through `make check`; package tests cover the behavioral conventions.

## 1. `gro` stays non-destructive

`gro` must have no dependency path into `internal/rw` or `internal/rwcmd`, its scopes must stay on the non-destructive allowlist, and its production graph must contain no forbidden destructive Google API calls.

The only send path is `grw mail send` in `internal/rw/gmail`; it is unreachable from `gro` by the link-graph test.

Enforced by `TestGroNeverLinksWriteCode`, `TestAllScopesAreNonDestructive`, and `TestNoDestructiveAPIMethodsInProductionCode`.

## 2. `grw` covers the read scope for every service it touches

If `grw` requests a scope for a Google service, it must retain every `gro` scope for that service. New scopes must also be added to the known write allowlist.

Enforced by `TestGrwScopesCoverGroPerService` and `TestGrwScopesAreKnown`.

## 3. Write layers extend read layers

`internal/rw/<domain>.Client` embeds the concrete read client. The write command interface embeds the read command interface, and the write `NewCommand()` begins with the read command before adding leaves.

Enforced by `TestWriteClientsEmbedReadClients`, `TestWriteCommandClientsEmbedReadCommandClients`, and `TestWriteCommandsExtendReadCommands`.

## 4. Every added mutation previews safely

Every leaf added by a write command declares boolean `--dry-run` with shorthand `-n`. The read verbs `list`, `get`, `show`, and `search` are exempt. A command exposing `--permanent` must also expose `--yes`; recoverable behavior remains the default.

Confirmation is reserved for unrecoverable data loss. Actions that are irreversible but not destructive, such as `grw mail send`, take an explicit ID, print what will happen, and proceed; `--dry-run` is the review step, and a prompt would only break the piping and agent flows grw exists for.

A leaf offering `--stdin` or `--query` resolves IDs through `internal/bulk`.

Enforced by `TestWriteLeavesHaveDryRun`, `TestPermanentWriteLeavesRequireYes`, and `TestBulkLeavesRouteThroughResolver`. `TestDelete_DryRunDoesNotMutate` and `TestDelete_PermanentWithYes` verify behavior.

## 5. Resource leaves are text-only

Leaves under mail, calendar, contacts, drive, and me do not declare `--json`. JSON is reserved for control-plane or diagnostic output such as configuration and refresh state.

Enforced by `TestResourceLeavesHaveNoJSONFlag` and `TestResourceLeaf_RejectsJSON_EndToEnd`.

## 6. Interfaces live at the consumer

Each package under `internal/cmd/<domain>` and `internal/rwcmd/<domain>` declares an exported interface ending in `Client`, containing only the operations its commands consume.

Enforced by `TestDomainPackagesDefineClientInterface`.

## 7. Construction is injectable and commands are composable

Every domain command package has a package-level `ClientFactory` and exports `NewCommand()`. Production factories call the concrete client constructor; tests temporarily replace the factory.

Enforced by `TestDomainPackagesHaveClientFactory` and `TestDomainPackagesExportNewCommand`. Handler tests such as `TestListCommand_ClientCreationError` exercise the injected failure path.

## 8. Context reaches every I/O call

Public I/O methods take `context.Context` first. Cobra handlers use `cmd.Context()` and pass that same context through the factory and client call so cancellation reaches Google requests.

Command and API tests exercise context-bearing interfaces, but there is no dedicated structural context test. New handlers should include one focused context/cancellation assertion when their flow is not already covered.

## 9. Errors retain their cause

Wrap lower-level failures with `fmt.Errorf("doing thing: %w", err)`. Keep messages lowercase and without trailing punctuation. Do not erase typed causes needed by authentication, migration, or callers.

Verified by `TestSystemErrorUnwrap`, `TestAttributedTokenSource_NamesRefOnAuthError`, and command `*_APIError`/`*_ClientCreationError` tests.

## 10. Mocks and helpers stay boring

Use function-field mocks in `mock_test.go` plus a compile-time interface assertion. Use `testutil.WithFactory` for temporary factory replacement, `testutil.CaptureStdout` for command output, the assertion helpers in `internal/testutil`, and existing `Sample*` fixtures before creating local fixtures.

Compile-time assertions enforce mock conformance. Package handler tests exercise `WithFactory` and `CaptureStdout`; assertion helpers have focused tests such as `TestEqual`, `TestErrorIs`, and `TestContains`.
