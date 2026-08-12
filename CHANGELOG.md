# Changelog

All notable changes to hosts-manager are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Per-tag [GitHub Releases](https://github.com/brandonhon/hosts-manager/releases) hold the
full commit-level list; this file curates the highlights and behavior changes. The
release workflow uses the section for the version being cut as the release body, so a
stable release cannot be published without one.

Build-infrastructure changes (CI workflows, pinned tool versions) are recorded too, under
**Changed** and labelled as such. They don't reach the released binaries — they're listed
so the provenance of a build is auditable from one file.

Entries up to and including `[0.6.1]` were generated automatically from commit messages
and use inline compare links in their headings. From `[Unreleased]` onward they are
hand-written, and compare links live in the reference list at the bottom of the file.

## [Unreleased]

### Fixed

- **An unrecognized `$EDITOR` no longer stops the tool from running at all.** The default
  editor is seeded from your environment, and any value outside a small hardcoded
  whitelist failed configuration validation — which aborted config loading, which exited
  every command with an error. `EDITOR=nvim`, `helix`, `micro`, or any absolute path such
  as `/usr/bin/vim` made even read-only commands like `list` and `search` unusable. An
  unrecognized editor is now a warning printed once at startup. Nothing is relaxed:
  `config --edit` still refuses to launch an editor it doesn't recognize, and the editor
  is never run through a shell.

### Changed

- Build infrastructure: **releases are now cut by pushing a tag rather than by a bot.**
  `semantic-release` decided versions from commit messages and published on every
  successful build of `main`; it has been removed along with its configuration. Tagging
  `vX.Y.Z` now builds the binaries and publishes the release, using this file's section
  for that version as the release notes with the commit list appended below. Version
  numbers are a deliberate choice again instead of a side effect of commit prefixes.
- Build infrastructure: the lint configuration was migrated to golangci-lint v2, which had
  been failing to load the v1-format config and taking the whole build with it. The linter
  version is now pinned in both CI and the Makefile, so local and CI linting check the
  same thing.
- Build infrastructure: the CI workflow no longer cross-compiles five targets on every
  push, supersedes its own in-flight runs, and grants the security scan the permission it
  needs to publish results.

## [0.6.1](https://github.com/brandonhon/hosts-manager/compare/v0.6.0...v0.6.1) (2025-10-02)

### 🐛 Bug Fixes

* update module path to github.com/brandonhon/hosts-manager ([0e5313f](https://github.com/brandonhon/hosts-manager/commit/0e5313facf5c4fe35541c2e953b6f4ecce70fbd1))

## [0.6.0](https://github.com/brandonhon/hosts-manager/compare/v0.5.2...v0.6.0) (2025-10-02)

### 🚀 Features

* add CLI category creation and TUI entry editing ([839582d](https://github.com/brandonhon/hosts-manager/commit/839582db90d847cb43a5205bcad71cd8e9643b1b))

### 🐛 Bug Fixes

* resolve CLI test failures and update TUI controls display ([a24bfe3](https://github.com/brandonhon/hosts-manager/commit/a24bfe3471053ebd1175d08c905ad47a3370c743))
* resolve root user test failures and enhance TUI controls display ([67646ab](https://github.com/brandonhon/hosts-manager/commit/67646aba36d37cf5c69425767ec90c2b714a3a4f))
* skip flaky concurrent test on Windows CI ([2edf98a](https://github.com/brandonhon/hosts-manager/commit/2edf98ac5f8f274d4a24ec8ae4c6051691c76848))

## [0.5.2](https://github.com/brandonhon/hosts-manager/compare/v0.5.1...v0.5.2) (2025-10-01)

### 📚 Documentation

* improve version variable documentation in main.go ([ef6c4b8](https://github.com/brandonhon/hosts-manager/commit/ef6c4b827bf1726ea686e2b361c26fc508002005))

## [0.5.1](https://github.com/brandonhon/hosts-manager/compare/v0.5.0...v0.5.1) (2025-10-01)

### 📚 Documentation

* add Global Flags section documenting --version flag ([a1bad31](https://github.com/brandonhon/hosts-manager/commit/a1bad31be577721c8d223b7b51f0cdfe76828e6a))

## [0.5.0](https://github.com/brandonhon/hosts-manager/compare/v0.4.7...v0.5.0) (2025-10-01)

### 🚀 Features

* add --version flag to CLI ([467d6e7](https://github.com/brandonhon/hosts-manager/commit/467d6e7b74b2b67b9748fa34686fc0aff2216c6f))

## [0.4.7](https://github.com/brandonhon/hosts-manager/compare/v0.4.6...v0.4.7) (2025-10-01)

### 🐛 Bug Fixes

* add ProgramData path validation for Windows config directory test ([0f8b92e](https://github.com/brandonhon/hosts-manager/commit/0f8b92efe7c5a44f7afa7bd5e789efb6e545f7e2))

## [0.4.6](https://github.com/brandonhon/hosts-manager/compare/v0.4.5...v0.4.6) (2025-09-30)

### 🐛 Bug Fixes

* resolve goimports formatting issues ([0b3990f](https://github.com/brandonhon/hosts-manager/commit/0b3990f5071456ca0c17013539d843c58e2f518d))

## [0.4.5](https://github.com/brandonhon/hosts-manager/compare/v0.4.4...v0.4.5) (2025-09-30)

### 🐛 Bug Fixes

* remove deprecated service section from golangci-lint config ([c58266f](https://github.com/brandonhon/hosts-manager/commit/c58266f8c945ee05e68026dfb451a3d11550b16c))

## [0.4.4](https://github.com/brandonhon/hosts-manager/compare/v0.4.3...v0.4.4) (2025-09-30)

### 🐛 Bug Fixes

* pin golangci-lint version for consistent CI behavior ([d78d640](https://github.com/brandonhon/hosts-manager/commit/d78d6408f1255575cb5a2f49e70cb9e34a73b9f3))
* resolve CI failures with golangci-lint version and Windows tests ([070d6fb](https://github.com/brandonhon/hosts-manager/commit/070d6fbe7da45daf2995a67cd7353261e99485bd))

## [0.4.3](https://github.com/brandonhon/hosts-manager/compare/v0.4.2...v0.4.3) (2025-09-30)

### 🐛 Bug Fixes

* resolve Go formatting issues and add golangci-lint version spec ([8661cd5](https://github.com/brandonhon/hosts-manager/commit/8661cd535d924b8fe93f6e228bbcff75b7930a4e))

## [0.4.2](https://github.com/brandonhon/hosts-manager/compare/v0.4.1...v0.4.2) (2025-09-30)

### 🐛 Bug Fixes

* apply De Morgan's law to improve code clarity in TUI validation ([54d13f7](https://github.com/brandonhon/hosts-manager/commit/54d13f7aa072c0de00d740ed3dc35980a4cb5081))

## [0.4.1](https://github.com/brandonhon/hosts-manager/compare/v0.4.0...v0.4.1) (2025-09-30)

### 🐛 Bug Fixes

* update CI status badge and add configuration validation comment ([6db55c9](https://github.com/brandonhon/hosts-manager/commit/6db55c925cb0d1d5b2918b8e6b41e76558dd290b))

## [0.4.0](https://github.com/brandonhon/hosts-manager/compare/v0.3.2...v0.4.0) (2025-09-30)

### 🚀 Features

* optimize CI workflows to run only on Go code changes ([8d66fb1](https://github.com/brandonhon/hosts-manager/commit/8d66fb1e369a2d441870719a647d3bf1837019fa))

## [0.3.2](https://github.com/brandonhon/hosts-manager/compare/v0.3.1...v0.3.2) (2025-09-30)

### 📚 Documentation

* fix README security rating and clarify release process ([f3cefab](https://github.com/brandonhon/hosts-manager/commit/f3cefab4d6c9acf6dd08a75c30f476392e2645be))

## [0.3.1](https://github.com/brandonhon/hosts-manager/compare/v0.3.0...v0.3.1) (2025-09-30)

### 📚 Documentation

* update EXAMPLES.md and CLAUDE.md with new TUI features ([e73edfe](https://github.com/brandonhon/hosts-manager/commit/e73edfe9aaddab20148684f941c3c38492a4fe68))

## [0.3.0](https://github.com/brandonhon/hosts-manager/compare/v0.2.0...v0.3.0) (2025-09-30)

### 🚀 Features

* optimize release assets to reduce total count ([a350f86](https://github.com/brandonhon/hosts-manager/commit/a350f861aa6c804543311e7450268decdf65a2d5))

## [0.2.0](https://github.com/brandonhon/hosts-manager/compare/v0.1.9...v0.2.0) (2025-09-30)

### 🚀 Features

* reset to development versioning and enhance TUI features ([26c53ac](https://github.com/brandonhon/hosts-manager/commit/26c53acf7c45020a96cc8a4d1df1b4c811e42039))

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
