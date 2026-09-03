# Google CLI standards index

This file points to repository-specific rules and family-wide sources. It is not a standalone style guide.

## Repository-specific standards

| Document | Use it for |
| --- | --- |
| [Golden principles](docs/golden-principles.md) | Mechanically enforced safety and package rules |
| [Architecture](docs/architecture.md) | Package graph, composition, identities, and structural tests |
| [Adding a read domain](docs/adding-a-domain.md) | A new Google API read surface |
| [Adding a write surface](docs/adding-a-write-surface.md) | Write-capable client and command extensions |
| [Workspace administrators](WORKSPACE_ADMINS.md) | Organization-managed OAuth setup for both binaries |

## Shared Open CLI standards

The source of truth is the [cli-common documentation](https://github.com/open-cli-collective/cli-common/tree/main/docs). A local convenience copy may exist at `../cli-common/docs`.

| Document | Use it for |
| --- | --- |
| [Repository layout](https://github.com/open-cli-collective/cli-common/blob/main/docs/repo-layout.md) | Repository shape, required files, Go policy, Make targets, and commit hygiene |
| [CI](https://github.com/open-cli-collective/cli-common/blob/main/docs/ci.md) | CI behavior and required checks |
| [Distribution](https://github.com/open-cli-collective/cli-common/blob/main/docs/distribution.md) | Packaging and installation channels |
| [Release](https://github.com/open-cli-collective/cli-common/blob/main/docs/release.md) | Release triggers, tags, and publishing |
| [Command surface](https://github.com/open-cli-collective/cli-common/blob/main/docs/command-surface.md) | Commands, arguments, flags, prompts, aliases, and mutation safety |
| [Output and rendering](https://github.com/open-cli-collective/cli-common/blob/main/docs/output-and-rendering.md) | Text output, JSON carve-outs, streams, color, and pagination |
| [Scriptability](https://github.com/open-cli-collective/cli-common/blob/main/docs/scriptability.md) | Non-interactive setup and automation behavior |
| [Working with secrets](https://github.com/open-cli-collective/cli-common/blob/main/docs/working-with-secrets.md) | Credential ingress, keyrings, migrations, redaction, and no-leak testing |
| [Working with state](https://github.com/open-cli-collective/cli-common/blob/main/docs/working-with-state.md) | Config/cache locations, credential references, migrations, and hermetic tests |

## Shared automation

Reusable actions and workflows live in https://github.com/open-cli-collective/.github. A local convenience copy may exist at `../.github`.

Repository-specific constraints belong here; family-wide changes belong in the relevant source above.
