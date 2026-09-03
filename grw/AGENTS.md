# AGENTS.md

Codex entrypoint for the google-readwrite (`grw`) repository.

## Start Here

`grw` is a thin read-write Gmail CLI built on the shared
`github.com/open-cli-collective/google-cli-common` module. Almost all logic
(OAuth, the Gmail client, rendering, the shared `mail`/init/config command
surface, root wiring) lives in google-cli-common. This repo contains only:
`internal/appidentity` (grw's identity + scopes), `internal/gmailrw` (the
read-write Gmail client: trash, permanent delete, label lifecycle, filters),
`internal/cmd/mail` (grw's destructive/settings leaves composed onto the shared
mail command), `internal/cmd/root`, and `cmd/grw`.

See `README.md` for the command surface and safety model.

## Shared Standards

Source of truth: https://github.com/open-cli-collective/google-cli-common
Family-wide: https://github.com/open-cli-collective/cli-common/tree/main/docs

## Shared Automation

Source of truth: https://github.com/open-cli-collective/.github

## grw-specific invariants

- grw is read-WRITE: unlike gro it may call destructive Gmail methods
  (`.BatchDelete`) and requests `gmail.settings.basic` + `mail.google.com`.
  Keep destructive code in `internal/gmailrw`, never in google-cli-common.
- `delete` defaults to Trash; permanent deletion stays behind `--permanent`
  plus a typed confirmation.
- grw's keyring namespace is `google-readwrite/*`, kept separate from gro's.
