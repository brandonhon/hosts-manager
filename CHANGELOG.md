# Changelog

All notable changes to this project will be documented in this file. See [Conventional Commits](https://conventionalcommits.org) for commit guidelines.

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

*This changelog follows [Conventional Commits](https://conventionalcommits.org) and [Semantic Versioning](https://semver.org/) guidelines.*
