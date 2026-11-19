# Contributing

Bug reports and pull requests are welcome. For anything larger than a small
fix, open an issue first so the approach can be agreed before you write it.

## Setup

You need Go 1.24+, Git and Make.

```bash
git clone https://github.com/brandonhon/hosts-manager.git
cd hosts-manager
make tools   # golangci-lint, gosec and govulncheck, at the versions CI uses
make check   # lint, tests with -race, gosec, govulncheck: what CI runs
```

`make help` lists every target. The ones you will use most:

| Target | |
|---|---|
| `make build` | build `build/hosts-manager` |
| `make test` | run the tests with the race detector |
| `make check` | everything CI checks |
| `make snapshot` | build the release archives into `dist/` without publishing (needs GoReleaser v2) |

The tests never touch your real hosts file. To try a change against it safely,
use `--dry-run`, which needs no privileges and writes nothing.

## Layout

```
cmd/hosts-manager/   commands; every write, CLI or editor, goes through saveHosts() in main.go
internal/hosts/      parsing, writing (atomic) and validating the hosts file
internal/tui/        the interactive editor
internal/backup/     backups and restore
internal/config/     config file loading and validation
internal/audit/      the one-line-per-change audit log
internal/platform/   OS paths, privilege checks, sudo ownership
```

## Pull requests

- Branch from `main` and open a PR against it. CI must pass; `Quality Gate` is
  the required check.
- Keep each commit to one change, with tests. A bug fix should come with a test
  that fails without it.
- If a change alters behavior a user can see, update README.md in the same PR.
- Commit subjects become the release notes, so make them read well on their
  own. They take the form `<scope>: <description>`; `.gitmessage` has
  examples (`git config commit.template .gitmessage` to use it).

## Releases

Maintainers release by tagging `main`:

```bash
git tag v1.2.3 && git push origin v1.2.3
```

The release workflow runs the tests and GoReleaser, which builds the archives
and `SHA256SUMS` and publishes a GitHub release with notes generated from the
commits since the previous tag. A tag with a suffix, such as `v1.2.3-rc.1`, is
published as a prerelease.
