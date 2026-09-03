# Development

This repository is one Go module, `github.com/open-cli-collective/google-cli`, producing `gro` and `grw` from `cmd/gro` and `cmd/grw`.

## Local workflow

```bash
make build
./bin/gro --help
./bin/grw --help
make check
make test-cover-check
```

`make build` writes both binaries to `bin/`. `make check` runs module tidiness, lint, the race-enabled test suite, and both builds. `make test-cover-check` runs the race-enabled suite with coverage and enforces the 60% floor.

The Makefile exports these build tags for local and CI work:

```text
keyring_no1password,keyring_nopassage
```

They omit optional 1Password and Passage keyring integrations. Do not use them to bypass platform keyring behavior in release builds; follow the [secrets standard](https://github.com/open-cli-collective/cli-common/blob/main/docs/working-with-secrets.md).

## Test layout

- Unit and command tests live beside their packages as `*_test.go`.
- Cross-package structural rules live in `internal/architecture`.
- End-to-end secret checks live in `internal/noleak`.
- Reusable assertions, output capture, factory replacement, and fixtures live in `internal/testutil`.

### Account-backed integration checks

These checks use live Google data and are intentionally manual. Use a disposable or controlled account, build first, and authenticate each binary you intend to test:

```bash
./bin/gro init
./bin/gro me
./bin/gro mail search "is:inbox" --max 1
./bin/gro calendar today

./bin/grw init
./bin/grw mail search "is:inbox" --max 1
./bin/grw mail delete --query "older_than:10y" --dry-run
```

For mutation checks, begin with `--dry-run`, use known test messages, verify the result in Gmail, and restore trashed messages afterward. Test permanent deletion only with disposable data and explicit `--permanent --yes`. Never run live account checks in the ordinary unit suite.

## CI

`.github/workflows/ci.yml` defines:

- `build-platform`: Linux, macOS, and Windows matrix builds; this matrix is not itself a required check.
- `build`: stable aggregate status for the platform matrix.
- `tidy`: `make tidy` and a clean `go.mod`/`go.sum` assertion.
- `test`: the reusable Go test action.
- `lint`: the reusable Go lint action.
- `static-release-guard`: CGO-disabled Linux and Windows builds for both binaries plus a forbidden-keyring dependency check.
- `coverage`: `make test-cover-check`.
- `pr-title`: pull-request title validation.

The required checks on `main` are `build`, `tidy`, `test`, `lint`, `static-release-guard`, `coverage`, and `pr-title`. The source for reusable automation is https://github.com/open-cli-collective/.github.

## Releases

There is one release stream for both binaries. `version.txt` contains the major/minor line (`1.2`), and automatic releases create `v1.2.N` tags. A squash merge to `main` whose final commit begins with `feat:` or `fix:` triggers the automatic release decision; documentation, test, CI, and chore-only merges do not cut a release.

Each tag publishes both binaries and their platform archives through the shared release automation. Homebrew, Chocolatey, WinGet, and Linux package publication fan out from that release. Follow the [release](https://github.com/open-cli-collective/cli-common/blob/main/docs/release.md) and [distribution](https://github.com/open-cli-collective/cli-common/blob/main/docs/distribution.md) standards rather than duplicating workflow policy here.

### Packaging identity

`packaging/identity.yml` (schema `open-cli-identity/v1`, described in the cli-common
[distribution](https://github.com/open-cli-collective/cli-common/blob/main/docs/distribution.md)
standard) declares both binaries once so the shared automation can validate, build, sign, and
publish them without per-repo workflow logic. Naming follows one rule: package registries use the long
names (`google-readonly` and `google-readwrite` on Chocolatey, WinGet, and Linux), while the
Homebrew casks are named after the binaries (`gro`, `grw`) with the long names as aliases. `gro`'s
identifiers predate the merge and cannot change without breaking upgrades; `grw` simply follows
the same pattern. Each binary's `keychain_probe` runs the freshly built macOS binary against a seeded config
and asserts it selects the Keychain backend, which catches a static or mis-tagged build before it
is published.

`version.txt` holds only `MAJOR.MINOR`; the `major_minor_run_patch` scheme appends the workflow
run number, so tags are `v1.2.N` and never collide or need a bump commit. The stream starts at
`1.2` because `gro` had already released `1.1.x`; continuing its line keeps upgrades monotonic for
existing installs.

Use a focused branch, run `make check`, and open a pull request. Keep the pull-request title in conventional-commit form because squash merge makes that title the commit on `main` and therefore the release signal.
