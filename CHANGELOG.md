# Changelog

**This file stops at 0.7.0.** The `[Unreleased]` section below is the curated summary of
everything since 0.6.1; rename it to `## [0.7.0] - <date>` before tagging and the release
workflow publishes it as the 0.7.0 release notes. Releases after that get notes GitHub
generates from the merged pull requests, on the [releases page](https://github.com/brandonhon/hosts-manager/releases),
unless a `## [X.Y.Z]` section is added here for them.

## [Unreleased]

0.7.0 reworks the command line, the interactive editor and the release process, and fixes a
long list of bugs found in an audit of 0.6.1. **Read "Breaking changes" before upgrading.**

### Breaking changes

- **Commands were consolidated from 20 to 11.** Old forms and their replacements:

  | 0.6.1 | 0.7.0 |
  |---|---|
  | `tui` | `hosts-manager` with no arguments |
  | `search <q>` | `list <q>` (plain case-insensitive substring match) |
  | `list --show-disabled` | `list -a` |
  | `delete <host>` | `rm <host>` (`delete` still works as an alias) |
  | `category list` | `list` (categories are its section headings) |
  | `category add <name>` | `add ... -c <name>` creates the category |
  | `category enable/disable <name>` | `enable/disable -c <name>` |
  | `profile list` / `profile activate <name>` | `profile` / `profile <name>` |
  | `restore --list` | `restore` with no argument |
  | `config --show` | `config` |
  | `config --edit` | removed; edit the file `config` prints the path of |
  | `add --comment` | unchanged, and now also `-m` |

- **Every command that changes the hosts file requires `sudo` (Administrator on Windows),
  as does the editor.** Write permission on the file is no longer enough: a group-writable
  `/etc/hosts` used to be edited without elevation. The editor previously had no check at
  all. `--dry-run` needs no privileges.
- **Under `sudo`, config, backups and the audit log are the invoking user's**, in their
  home directory and owned by them. Earlier versions wrote them to root's home on most
  distributions, so backups made by 0.6.1 under `sudo` are in root's home directory (on
  Linux, `/root/.local/share/hosts-manager/backups`) and are not listed by `restore`.
- **The audit log is now `<data dir>/audit/changes.log`, one plain-text line per change**
  (`2026-11-19T15:04:05Z  alice  add  ok  Added ...`), moved to `changes.log.1` at 1 MiB.
  The old JSON `audit.log` is no longer written; it is left in place.
- **Config settings that nothing read were removed:** `ui.*`, `categories`,
  `general.editor` and `export.formats`. Existing config files still load; the keys are
  ignored.
- **`import` and `export -o` use the path you give.** They were restricted to the tool's
  own directories, and `export -o ./file` silently wrote to the data directory instead.
- **Error messages are shown as-is.** IP addresses, paths and user names were masked
  (`add [ip-address] ...`).

### Added

- A rebuilt interactive editor: categories on the left with enabled/total counts, entries
  on the right, a top bar with the active profile and unsaved-change count. Toggle a whole
  category, filter across every category (`/`), switch profile (`p`), and add or edit with
  validation as you type. Scrolls, and adapts to terminals narrower than 80 columns.
- A command used wrongly (wrong arguments, unknown flag, bad flag value) prints its help
  after the error.
- `restore <n>` restores by position in the list; `restore 1` is the newest backup.
- `list` shows how many disabled entries each category hides.

### Fixed

- **Hosts file contents**
  - Comments containing line breaks are rejected; a comment could inject an arbitrary host
    mapping. `import` now validates every entry before writing anything; it bypassed
    validation entirely.
  - `0.0.0.0` and `::` are accepted, so the shipped `blocked` category is usable.
  - Comments below the first host line were deleted on save.
  - Every save reordered categories and added a blank line to the header.
  - Category names containing a hyphen were lost on the next read; their entries moved to
    `default`.
  - `export -f hosts` dropped categories, so a round trip put everything in `default`.
  - Comments are no longer rejected for containing text like `data:` or `eval(`.
- **Backups and restore**
  - A failed restore could modify the hosts file; restore now writes atomically. Three
    write errors that were silently discarded, one of which could commit a truncated hosts
    file, are now reported.
  - The first run had no backup directory, so every command failed while `auto_backup` was
    on (the default).
  - `--dry-run` created real backups, which could push genuine ones out of retention.
  - Unreadable backups were hidden from `restore` and never cleaned up.
- **Commands and the editor**
  - `--dry-run` required root, so the people a preview is for could not use it. The editor
    ignored `--dry-run` and saved anyway.
  - The editor's list ran off the screen once it was longer than the terminal.
  - `q` in the editor discarded unsaved changes without asking.
  - Of two lines with the same IP and first hostname, the editor always toggled, edited or
    deleted the first.
  - The editor saved edits without validating them, and saved without taking a backup or
    writing to the audit log.
  - `enable`, `disable` and `rm` in a dry run reported success for hostnames that do not
    exist; category enable/disable did so for categories that do not exist.
  - Only `add` wrote to the audit log.
  - Commands without arguments, such as `backup`, silently ignored any they were given.
  - An unrecognized `$EDITOR` made every command fail at startup.
  - `XDG_CONFIG_HOME` and `XDG_DATA_HOME` are honored on macOS.

### Changed

- Build infrastructure: releases are cut by pushing a `vX.Y.Z` tag. GoReleaser builds the
  same five archives plus `SHA256SUMS`; release notes are generated from merged pull requests. This
  replaces `semantic-release`, which chose versions from commit prefixes.
- Build infrastructure: the Makefile is 12 targets (`make help`); `make check` runs what
  CI runs. CI runs tests with `-race` on Linux, macOS and Windows, plus golangci-lint v2,
  gosec and govulncheck, all at pinned versions. gosec had never run before: it was
  installed from a module path that does not exist.
- The program is 3,551 lines of Go, down from 6,128 in 0.6.1, after removing unused and
  redundant code: the error sanitizer, fuzzy search, unreachable validators and unused
  backup functions among it.

### Removed

- The `Dockerfile`, which ran the tool as root in a container against a bind-mounted
  `/etc/hosts`.
- `INSTALL.md` and `EXAMPLES.md`; their content is in the README.

## [0.6.1](https://github.com/brandonhon/hosts-manager/releases/tag/v0.6.1) (2025-10-02)

### 🐛 Bug Fixes

* update module path to github.com/brandonhon/hosts-manager ([4394802](https://github.com/brandonhon/hosts-manager/commit/439480201dd2ac59d4b822192de34b16b016be24))

[Unreleased]: https://github.com/brandonhon/hosts-manager/compare/v0.6.1...HEAD
