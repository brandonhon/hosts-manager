# Changelog

**This file stops at 0.7.0.** The `[Unreleased]` section below is the curated summary of
everything since 0.6.1; rename it to `## [0.7.0] - <date>` before tagging and the release
workflow publishes it as the 0.7.0 release notes. Releases after that get notes generated
from the commit history on the [releases page](https://github.com/brandonhon/hosts-manager/releases),
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
  same five archives plus `SHA256SUMS`; release notes are generated from commits. This
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

## [0.6.1](https://github.com/brandonhon/hosts-manager/compare/v0.6.0...v0.6.1) (2025-10-02)

### 🐛 Bug Fixes

* update module path to github.com/brandonhon/hosts-manager ([4394802](https://github.com/brandonhon/hosts-manager/commit/439480201dd2ac59d4b822192de34b16b016be24))

## [0.6.0](https://github.com/brandonhon/hosts-manager/compare/v0.5.2...v0.6.0) (2025-10-02)

### 🚀 Features

* add CLI category creation and TUI entry editing ([a068eff](https://github.com/brandonhon/hosts-manager/commit/a068eff1f9f6a9994bd488f79da0fbc744706a39))

### 🐛 Bug Fixes

* resolve CLI test failures and update TUI controls display ([595abed](https://github.com/brandonhon/hosts-manager/commit/595abed5e0e50fdb72ef74bb65a1219d66f8c01c))
* resolve root user test failures and enhance TUI controls display ([47d1ae9](https://github.com/brandonhon/hosts-manager/commit/47d1ae95fcaca6aba054d2345f56039ac783374c))
* skip flaky concurrent test on Windows CI ([87abebf](https://github.com/brandonhon/hosts-manager/commit/87abebfe7c9ccfb8d7bd5d3ab53859ec14413b5d))

## [0.5.2](https://github.com/brandonhon/hosts-manager/compare/v0.5.1...v0.5.2) (2025-10-01)

### 📚 Documentation

* improve version variable documentation in main.go ([d727d6e](https://github.com/brandonhon/hosts-manager/commit/d727d6e2a51974b2d53bdb392931a55f2ea8dcae))

## [0.5.1](https://github.com/brandonhon/hosts-manager/compare/v0.5.0...v0.5.1) (2025-10-01)

### 📚 Documentation

* add Global Flags section documenting --version flag ([7248126](https://github.com/brandonhon/hosts-manager/commit/724812605bd22a2c8dbb09019f86363e574e4362))

## [0.5.0](https://github.com/brandonhon/hosts-manager/compare/v0.4.7...v0.5.0) (2025-10-01)

### 🚀 Features

* add --version flag to CLI ([9995799](https://github.com/brandonhon/hosts-manager/commit/99957994cce18e56ba14ccb345fc585b69f69274))

## [0.4.7](https://github.com/brandonhon/hosts-manager/compare/v0.4.6...v0.4.7) (2025-10-01)

### 🐛 Bug Fixes

* add ProgramData path validation for Windows config directory test ([5c2441d](https://github.com/brandonhon/hosts-manager/commit/5c2441dbc45faad8bf88b07c803b802900da2629))

## [0.4.6](https://github.com/brandonhon/hosts-manager/compare/v0.4.5...v0.4.6) (2025-09-30)

### 🐛 Bug Fixes

* resolve goimports formatting issues ([9d619cc](https://github.com/brandonhon/hosts-manager/commit/9d619cc806961761a27ffac9a6a89025b6398b94))

## [0.4.5](https://github.com/brandonhon/hosts-manager/compare/v0.4.4...v0.4.5) (2025-09-30)

### 🐛 Bug Fixes

* remove deprecated service section from golangci-lint config ([0fcd533](https://github.com/brandonhon/hosts-manager/commit/0fcd5339f7bb538f616028cff8fb2d7729e9eb1d))

## [0.4.4](https://github.com/brandonhon/hosts-manager/compare/v0.4.3...v0.4.4) (2025-09-30)

### 🐛 Bug Fixes

* pin golangci-lint version for consistent CI behavior ([3b55585](https://github.com/brandonhon/hosts-manager/commit/3b555859448fdfa423cab00490ab16cf9fbf9a26))
* resolve CI failures with golangci-lint version and Windows tests ([8e60ae1](https://github.com/brandonhon/hosts-manager/commit/8e60ae19328b7f66b4a707ec7ddf8447565a597a))

## [0.4.3](https://github.com/brandonhon/hosts-manager/compare/v0.4.2...v0.4.3) (2025-09-30)

### 🐛 Bug Fixes

* resolve Go formatting issues and add golangci-lint version spec ([6b6c600](https://github.com/brandonhon/hosts-manager/commit/6b6c60024d336142a5c0264e809a6347605bad23))

## [0.4.2](https://github.com/brandonhon/hosts-manager/compare/v0.4.1...v0.4.2) (2025-09-30)

### 🐛 Bug Fixes

* apply De Morgan's law to improve code clarity in TUI validation ([d14c9e0](https://github.com/brandonhon/hosts-manager/commit/d14c9e02a4957e6d443d7474253be894883a1a9f))

## [0.4.1](https://github.com/brandonhon/hosts-manager/compare/v0.4.0...v0.4.1) (2025-09-30)

### 🐛 Bug Fixes

* update CI status badge and add configuration validation comment ([f69ce3b](https://github.com/brandonhon/hosts-manager/commit/f69ce3bc45b09e6323ad9569728d1d0249d3c564))

## [0.4.0](https://github.com/brandonhon/hosts-manager/compare/v0.3.2...v0.4.0) (2025-09-30)

### 🚀 Features

* optimize CI workflows to run only on Go code changes ([0c44734](https://github.com/brandonhon/hosts-manager/commit/0c447342994085b2edc41e7e1774e1d426e67297))

## [0.3.2](https://github.com/brandonhon/hosts-manager/compare/v0.3.1...v0.3.2) (2025-09-30)

### 📚 Documentation

* fix README security rating and clarify release process ([9c1bf22](https://github.com/brandonhon/hosts-manager/commit/9c1bf228a2e8787dff4d967035a3dafd8edb5943))

## [0.3.1](https://github.com/brandonhon/hosts-manager/compare/v0.3.0...v0.3.1) (2025-09-30)

### 📚 Documentation

* update EXAMPLES.md and developer notes with new TUI features ([151d70a](https://github.com/brandonhon/hosts-manager/commit/151d70a9b195d1761a2face940ffd915f71279eb))

## [0.3.0](https://github.com/brandonhon/hosts-manager/compare/v0.2.0...v0.3.0) (2025-09-30)

### 🚀 Features

* optimize release assets to reduce total count ([da2d69a](https://github.com/brandonhon/hosts-manager/commit/da2d69a0e22e371c34cccd121c5e95c4747e2c0b))

## [0.2.0](https://github.com/brandonhon/hosts-manager/compare/v0.1.9...v0.2.0) (2025-09-30)

### 🚀 Features

* reset to development versioning and enhance TUI features ([afa49e3](https://github.com/brandonhon/hosts-manager/commit/afa49e3e4acf6f97400f30bc4546fc105bd28b20))

## Development Release Notes

This project follows a 0.x.x versioning scheme during development. The API and behavior may change between releases until version 1.0.0.

### Upcoming Release (0.2.0)

#### 🚀 New Features
- **Enhanced TUI Mode**: Added interactive category management
  - Move entries between categories with guided interface (`m` key)
  - Create new custom categories with name and description (`c` key)
  - Improved navigation and user experience
- **Advanced Security Framework**: Comprehensive security hardening
  - Input validation and sanitization against injection attacks
  - Secure file operations with atomic writes and locking
  - Audit logging with tamper-evident trails
  - Privilege escalation only when necessary
- **Cross-Platform Compatibility**: Enhanced Windows support
  - Platform-specific file locking implementations
  - Improved permission handling across operating systems

#### 🐛 Bug Fixes
- Resolved CI workflow linting and configuration issues
- Fixed cross-platform build compatibility
- Improved error handling and user feedback

#### ♻️ Code Refactoring
- Reorganized TUI code with proper separation of concerns
- Enhanced test coverage with comprehensive unit tests
- Improved build system with automated quality gates

#### 🛠 Build System
- Automated release workflow with semantic versioning
- Multi-platform binary distribution (Linux, macOS, Windows)
- Comprehensive linting and security analysis pipeline

---

### Previous Development History

The project previously used 1.x.x versioning during initial development phases. All previous releases have been reset to establish a proper 0.x.x development versioning scheme leading to a stable 1.0.0 release.

#### Key Milestones Achieved
- ✅ Complete rewrite in Go with modern architecture
- ✅ Cross-platform hosts file management
- ✅ Interactive TUI with Bubble Tea framework
- ✅ Comprehensive security hardening (A- security rating)
- ✅ Automated CI/CD pipeline with quality gates
- ✅ Multi-platform binary distribution
- ✅ Template system with category management
- ✅ Backup and restore functionality
- ✅ Configuration management system

---

[Unreleased]: https://github.com/brandonhon/hosts-manager/compare/v0.6.1...HEAD
