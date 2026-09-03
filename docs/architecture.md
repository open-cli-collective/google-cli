# Architecture

The module produces a constrained binary (`gro`) and a write-capable binary (`grw`) from one package graph.

## Package graph

```text
cmd/gro
  -> internal/app/gro
       -> internal/cmd/{mail,calendar,contacts,drive,me}
            -> internal/api/<domain>
       -> internal/{auth,config,keychain,...}

cmd/grw
  -> internal/app/grw
       -> internal/rwcmd/mail
            -> internal/cmd/mail
            -> internal/rw/gmail
                 -> internal/api/gmail
       -> internal/rwcmd/calendar
            -> internal/cmd/calendar
            -> internal/rw/calendar
                 -> internal/api/calendar
       -> internal/{auth,config,keychain,...}
```

The dependency direction is deliberate:

- API packages do not import command packages.
- Read command packages depend on read API packages.
- Write clients in `internal/rw/<domain>` embed the corresponding concrete read client from `internal/api/<domain>`.
- Write commands in `internal/rwcmd/<domain>` compose the read command and add leaves; read packages never import write packages.
- `cmd/gro` cannot reach `internal/rw` or `internal/rwcmd` through its link graph.

Support packages such as `auth`, `bulk`, `config`, `keychain`, `output`, and `testutil` stay under `internal` and are available to both applications where appropriate.

## Domain shape

An API package exposes a concrete `Client` and a `NewClient(context.Context)` constructor. The consuming command package owns the smallest exported interface it needs, a replaceable `ClientFactory`, and a `NewCommand()` constructor:

```text
internal/api/<domain>/Client
             ^
internal/cmd/<domain>/<Domain>Client + ClientFactory + NewCommand()
```

This interface-at-consumer shape keeps production construction direct and tests able to replace the factory with a function-field mock.

A write-enabled domain adds two layers:

```text
internal/rw/<domain>/Client
  embeds *internal/api/<domain>.Client

internal/rwcmd/<domain>/<Domain>Client
  embeds internal/cmd/<domain>'s client interface
  adds write methods, ClientFactory, and NewCommand()
```

The write `NewCommand()` starts with `internal/cmd/<domain>.NewCommand()` and attaches only the additional leaves. This guarantees that its read surface stays aligned.

## Application composition and identity

`cmd/gro/main.go` registers `internal/app/gro.Identity()` before running the `gro` root. That root composes setup/configuration commands and all five read domains.

`cmd/grw/main.go` registers `internal/app/grw.Identity()` before running the `grw` root. That root composes setup/configuration and profile commands with the extended Gmail and Calendar commands.

Both identities drive the same config, cache, credential-reference, and keyring code. They use different directory names, default credential references, environment-variable prefixes, and keyring namespaces. Their OAuth client JSON may be reused across identities, but tokens and consent remain separate. `gro` requests its non-destructive multi-service scopes; `grw` requests Gmail write scopes, Calendar read/write scopes, and basic profile access.

## Structural enforcement

Tests live in `internal/architecture/architecture_test.go` unless noted:

| Test | Enforced rule |
| --- | --- |
| `TestDomainPackagesDefineClientInterface` | Every read/write command package owns an exported client interface. |
| `TestDomainPackagesHaveClientFactory` | Every command package exposes `ClientFactory`. |
| `TestDomainPackagesExportNewCommand` | Every domain command package exports `NewCommand()`. |
| `TestWriteCommandsExtendReadCommands` | Each write command preserves all read leaves and adds at least one write leaf. |
| `TestWriteClientsEmbedReadClients` | Each write API client embeds its read client. |
| `TestWriteCommandClientsEmbedReadCommandClients` | Each write command interface embeds its read command interface. |
| `TestWriteLeavesHaveDryRun` | Added mutating leaves expose boolean `--dry-run` (`-n`). |
| `TestPermanentWriteLeavesRequireYes` | Any leaf with `--permanent` also has `--yes`. |
| `TestResourceLeavesHaveNoJSONFlag` and `TestResourceLeaf_RejectsJSON_EndToEnd` | Resource leaves are text-only. |
| `TestAllScopesAreNonDestructive` | `gro` scopes remain on the non-destructive allowlist. |
| `TestGrwScopesCoverGroPerService` and `TestGrwScopesAreKnown` | A service touched by `grw` retains that service's `gro` scopes and adds only known scopes. |
| `TestGroNeverLinksWriteCode` | The `gro` link graph cannot include write packages. |
| `TestGrwLinksWriteCode` | The `grw` graph includes its required write packages. |
| `TestNoDestructiveAPIMethodsInProductionCode` | `gro` production dependencies contain no forbidden destructive calls. |
| `TestLdflagsStampTheLinkedVersionPackage` | Makefile version stamps target the linked version package. |

See [golden principles](golden-principles.md) for the coding rules around this structure.
