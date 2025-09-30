# Hosts Manager

[![Release](https://img.shields.io/github/v/release/brandonhon/hosts-manager)](https://github.com/brandonhon/hosts-manager/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/brandonhon/hosts-manager/ci.yml?branch=main)](https://github.com/brandonhon/hosts-manager/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/brandonhon/hosts-manager)](https://goreportcard.com/report/github.com/brandonhon/hosts-manager)
[![License](https://img.shields.io/github/license/brandonhon/hosts-manager)](LICENSE)

A powerful, cross-platform CLI hosts file manager with template system, backup/restore, interactive TUI, and advanced search capabilities.

## Features

### Core Features
- **Cross-platform support** - Works on Linux, macOS, and Windows
- **Template system** - Organize entries by categories (development, staging, production, custom)
- **CRUD operations** - Add, delete, enable/disable, search, and comment on entries
- **Interactive TUI mode** - Navigate and edit entries with a user-friendly terminal interface
- **Backup & restore** - Automatic backups with ISO 8601 timestamps
- **Fuzzy search** - Find entries by hostname, IP, or comments
- **Profile system** - Switch between different sets of enabled categories

### Advanced Features
- **Export/import** - Share configurations in YAML, JSON, or hosts format
- **Configuration management** - Customizable defaults and behavior
- **Permission handling** - Automatic elevation (sudo/admin) when needed
- **Dry run mode** - Preview changes before applying them
- **Audit trail** - Track changes with timestamps and descriptions
- **Lock file protection** - Prevent concurrent modifications

## Installation

### Pre-built Binaries (Recommended)

Download the latest release from the [releases page](https://github.com/brandonhon/hosts-manager/releases).

#### Linux/macOS
```bash
# Download and install (replace with latest version)
curl -L -o hosts-manager.tar.gz https://github.com/brandonhon/hosts-manager/releases/latest/download/hosts-manager-v0.2.0-linux-amd64.tar.gz
tar -xzf hosts-manager.tar.gz
sudo mv hosts-manager /usr/local/bin/hosts-manager
chmod +x /usr/local/bin/hosts-manager
```

#### Windows
Download the `.zip` file from the releases page and add the extracted binary to your PATH.

### From Source

```bash
git clone https://github.com/brandonhon/hosts-manager.git
cd hosts-manager
make build
sudo make install
```

### Package Managers

Coming soon: Homebrew, Chocolatey, and Snap packages.

## Quick Start

```bash
# List current entries
hosts-manager list

# Add a new entry
hosts-manager add 192.168.1.100 myapp.local

# Search entries
hosts-manager search myapp

# Start interactive mode
hosts-manager tui

# Create a backup
hosts-manager backup

# Show help
hosts-manager --help
```

## Usage

### Basic Commands

#### Add Entry
```bash
hosts-manager add <ip> <hostname> [hostname...] [flags]

# Examples
hosts-manager add 127.0.0.1 myapp.local
hosts-manager add 192.168.1.100 api.dev web.dev --category development --comment "Development services"
```

#### List Entries
```bash
hosts-manager list [flags]

# Examples
hosts-manager list                          # List all entries
hosts-manager list --category development   # List development entries only
hosts-manager list --show-disabled         # Include disabled entries
```

#### Delete Entry
```bash
hosts-manager delete <hostname>

# Example
hosts-manager delete myapp.local
```

#### Enable/Disable Entry
```bash
hosts-manager enable <hostname>
hosts-manager disable <hostname>

# Examples
hosts-manager enable myapp.local
hosts-manager disable api.staging
```

#### Search Entries
```bash
hosts-manager search <query> [flags]

# Examples
hosts-manager search myapp                    # Basic search
hosts-manager search "192.168" --fuzzy       # Fuzzy search on IP
hosts-manager search api --category staging  # Search within category
```

### Backup and Restore

#### Create Backup
```bash
hosts-manager backup
# Creates: /etc/hosts.backup.2023-12-07T10-30-45
```

#### List Backups
```bash
hosts-manager restore --list
```

#### Restore Backup
```bash
hosts-manager restore /path/to/backup/file
# or
hosts-manager restore hosts.backup.2023-12-07T10-30-45
```

### Category Management

#### List Categories
```bash
hosts-manager category list
```

#### Enable/Disable Category
```bash
hosts-manager category enable development
hosts-manager category disable staging
```

### Profile Management

Profiles allow you to quickly switch between different sets of enabled categories.

#### List Profiles
```bash
hosts-manager profile list
```

#### Activate Profile
```bash
hosts-manager profile activate development
```

### Export/Import

#### Export
```bash
hosts-manager export [flags]

# Examples
hosts-manager export --format yaml > my-hosts.yaml
hosts-manager export --format json --output hosts.json
hosts-manager export --format hosts --category development > dev-hosts.txt
```

#### Import
```bash
hosts-manager import <file> [flags]

# Examples
hosts-manager import hosts.yaml
hosts-manager import hosts.json --merge  # Merge with existing entries
```

### Interactive TUI Mode

Start the interactive terminal user interface:

```bash
hosts-manager tui
```

**TUI Controls:**
- `↑/↓` or `k/j` - Navigate entries
- `space` - Toggle entry enabled/disabled
- `a` - Add new entry
- `d` - Delete entry
- `m` - Move entry to different category
- `c` - Create new category
- `s` - Save changes (shows confirmation)
- `/` - Search mode
- `r` - Refresh
- `?` - Help
- `q` - Quit

**New TUI Features:**
- **Move entries**: Use `m` to move selected entry to a different category with guided interface
- **Create categories**: Use `c` to create new custom categories with name and description

### Configuration

View and edit configuration:

```bash
hosts-manager config --show    # Display current configuration
hosts-manager config --edit    # Edit configuration file
```

#### Configuration File

The configuration file is automatically created at:
- **Linux/macOS**: `~/.config/hosts-manager/config.yaml`
- **Windows**: `%APPDATA%\hosts-manager\config.yaml`

Example configuration:

```yaml
general:
  default_category: custom
  auto_backup: true
  dry_run: false
  verbose: false
  editor: nano

categories:
  development: "Development environments and local services"
  staging: "Staging and testing environments"
  production: "Production services and critical infrastructure"
  custom: "User-defined entries"

profiles:
  minimal:
    description: "Minimal profile with essential entries only"
    categories: ["production"]
    default: false
  development:
    description: "Development profile"
    categories: ["development", "staging"]
    default: false
  full:
    description: "Full profile with all categories"
    categories: ["development", "staging", "production", "custom"]
    default: true

ui:
  color_scheme: auto
  show_line_numbers: true
  page_size: 20

backup:
  directory: ""  # Auto-detected
  max_backups: 10
  retention_days: 30
  compression_type: gzip
```

## File Structure

The hosts manager organizes entries using special comment markers:

```
# @category development Development services
# =============== DEVELOPMENT ===============
127.0.0.1 myapp.local # My application
192.168.1.100 api.dev web.dev # Development APIs

# @category production Production services
# =============== PRODUCTION ===============
10.0.0.100 api.production.com
```

## Cross-Platform Support

### Linux/macOS
- Hosts file: `/etc/hosts`
- Requires `sudo` for modifications
- Config directory: `~/.config/hosts-manager`

### Windows
- Hosts file: `C:\Windows\System32\drivers\etc\hosts`
- Requires "Run as Administrator"
- Config directory: `%APPDATA%\hosts-manager`

## Development

### Prerequisites
- Go 1.19+
- Make (optional, but recommended for development)

### Development Tools (Auto-installable)
The following tools can be automatically installed using `make install-dev-tools`:
- **golangci-lint** - Comprehensive Go linter
- **gosec** - Go security checker
- **nancy** - Dependency vulnerability scanner
- **govulncheck** - Go vulnerability database checker
- **semgrep** - Semantic code analysis
- **go-licenses** - License compliance checker
- **cyclonedx-gomod** - SBOM generator
- **deadcode** - Dead code detector
- **ineffassign** - Ineffectual assignment detector
- **misspell** - Spelling checker

### Build from Source
```bash
git clone https://github.com/your-username/hosts-manager.git
cd hosts-manager
make deps      # Download dependencies
make build     # Build binary
make test      # Run tests
make install   # Install locally
```

### Development Commands
```bash
make dev         # Quick development build
make test        # Run tests
make coverage    # Generate coverage report
make lint        # Run comprehensive linters
make lint-fast   # Run fast linters (no security checks)
make lint-fix    # Auto-fix linting issues where possible
make fmt         # Format code
make vet         # Vet code
make validate    # Full validation pipeline
make validate-fast    # Fast validation (no security scans)
make validate-full    # Complete validation with all checks
make pre-commit       # Pre-commit validation checks
make quality-gate     # Quality gate for CI/CD
```

### Security and Quality Assurance
```bash
# Security Analysis
make security            # Run all security checks
make security-gosec      # Go security checker
make security-nancy      # Dependency vulnerability scanner
make security-govulncheck # Go vulnerability database check
make security-semgrep    # Semantic code analysis

# Code Quality
make sbom               # Generate Software Bill of Materials
make license-check      # Check license compliance
make deadcode          # Find unused code
make ineffassign       # Find ineffectual assignments
make misspell          # Find spelling mistakes

# Development Tools
make install-linters      # Install all linting tools
make install-security-tools  # Install security analysis tools
make install-dev-tools   # Install all development tools
```

### Cross-Compilation
```bash
make release     # Build for all platforms
make dist        # Create distribution packages
```

## Code Quality and Security Tools

Hosts Manager includes comprehensive linting and security analysis tools to ensure code quality and security:

### Linting Configuration
The project uses `golangci-lint` with an extensive configuration (`.golangci.yml`) that includes:

#### Security Linters
- **gosec** - Security audit for Go code
- **gas** - Additional security checks
- **depguard** - Dependency restrictions and policies

#### Code Quality Linters
- **staticcheck** - Advanced static analysis
- **govet** - Go vet with enhanced checks
- **errcheck** - Unchecked error detection
- **unused** - Dead code detection
- **ineffassign** - Ineffectual assignment detection
- **unconvert** - Unnecessary type conversion detection
- **goconst** - Repeated string constant detection
- **gocyclo** - Cyclomatic complexity analysis
- **gocognit** - Cognitive complexity analysis
- **dupl** - Code clone detection
- **misspell** - Spelling mistake detection

#### Style and Formatting Linters
- **gofmt** - Go formatting
- **goimports** - Import formatting and organization
- **gci** - Import ordering
- **gofumpt** - Stricter formatting rules
- **revive** - Enhanced Go linting (golint replacement)
- **stylecheck** - Style consistency checks

#### Performance Linters
- **prealloc** - Slice preallocation opportunities
- **bodyclose** - HTTP response body closure
- **noctx** - HTTP requests without context

### Security Analysis Tools
The project integrates multiple security analysis tools:

- **gosec** - Go Security Checker for vulnerability detection
- **nancy** - Dependency vulnerability scanner using Sonatype OSS Index
- **govulncheck** - Official Go vulnerability database checker
- **semgrep** - Semantic code analysis for security patterns

### Quality Assurance Features
- **Software Bill of Materials (SBOM)** generation in CycloneDX format
- **License compliance** checking with go-licenses
- **Comprehensive test coverage** reporting with visualization
- **Benchmark performance** tracking
- **Pre-commit hooks** for automated quality checks
- **CI/CD quality gates** for automated validation

### Development Workflow Integration
The build system provides multiple validation levels:
- **Fast validation** (`make validate-fast`) - Essential checks for development
- **Full validation** (`make validate-full`) - Complete analysis including security scans
- **Pre-commit validation** (`make pre-commit`) - Optimized for git hooks
- **Quality gate** (`make quality-gate`) - CI/CD pipeline validation

All tools can be automatically installed using `make install-dev-tools`.

## Security Features

Hosts Manager implements comprehensive security measures to protect your system:

### Input Validation & Sanitization
- **Comprehensive IP validation** - Validates IPv4/IPv6 addresses with security checks for dangerous ranges
- **RFC-compliant hostname validation** - Prevents malicious hostnames and injection attacks
- **Path traversal protection** - Sanitizes file paths to prevent unauthorized file access
- **Anti-injection measures** - Protects against script injection, command injection, and null byte attacks
- **Homograph attack detection** - Prevents IDN spoofing and similar-looking character attacks

### Secure File Operations
- **Atomic file operations** - Prevents corruption during concurrent access
- **Exclusive file locking** - Uses system-level locks to prevent race conditions
- **Stale lock detection** - Automatically cleans up abandoned lock files
- **Secure temporary files** - Creates temporary files with appropriate permissions

### Privilege Management
- **Minimal privilege escalation** - Only requests elevated privileges when necessary
- **Platform-specific elevation** - Uses appropriate methods for each operating system
- **Strict security mode** - Enhanced privilege checking for security-sensitive operations
- **Permission validation** - Verifies write permissions before attempting modifications

### Audit & Monitoring
- **Comprehensive audit logging** - Tracks all security-relevant operations
- **Security violation detection** - Logs and alerts on suspicious activities
- **Automatic log rotation** - Prevents audit logs from consuming excessive disk space
- **Tamper-evident logs** - Uses structured JSON format with timestamps and integrity checking

### Configuration Security
- **Schema validation** - Validates all configuration values against security policies
- **Editor whitelist** - Only allows execution of approved, safe text editors
- **Template sanitization** - Prevents dangerous template constructs and operations
- **Safe error handling** - Sanitizes error messages to prevent information disclosure

### Backup Security
- **Secure deletion** - Overwrites file content before deletion
- **Integrity verification** - Uses SHA-256 hashing to verify backup integrity
- **Compressed backups** - Automatically compresses backups to save space
- **Retention policies** - Automatic cleanup of old backups based on age and count

### Additional Protections
- **Lock file prevents concurrent modifications** - System-level file locking
- **Dry-run mode for safe testing** - Preview changes without applying them
- **Always creates backups before modifications** - Automatic safety net
- **Permission elevation only when needed** - Follows principle of least privilege
- **IPv6 link-local address warnings** - Logs warnings for potentially problematic addresses
- **Null byte injection protection** - Prevents null byte attacks in all inputs

### Security Best Practices
- All sensitive files created with restrictive permissions (0600/0700)
- Comprehensive input validation on all user-provided data
- Error messages sanitized to prevent information disclosure
- Audit trail for all security-relevant operations
- Regular validation of system state and permissions

### Security Audit Trail
This project has undergone comprehensive security auditing:
- **Initial Security Rating**: B+ (Good)
- **Post-Hardening Rating**: A- (Excellent)
- **Critical Issues**: 1 resolved (path validation gap)
- **Medium Issues**: 3 resolved (command injection, IPv6 policy, error disclosure)
- **Low Priority Issues**: 4 resolved (race conditions, log management, config validation, DoS prevention)

Key security improvements implemented:
- Zero tolerance for path traversal attacks
- Complete audit log injection prevention
- Resource exhaustion protection in all file operations
- Comprehensive input sanitization across all attack vectors
- Enterprise-grade temporal and process isolation

## Troubleshooting

### Permission Denied
**Linux/macOS:**
```bash
sudo hosts-manager add 127.0.0.1 myapp.local
```

**Windows:**
Run PowerShell or Command Prompt as Administrator.

### Backup Directory
The backup directory is automatically created. Default locations:
- **Linux**: `~/.local/share/hosts-manager/backups`
- **macOS**: `~/Library/Application Support/hosts-manager/backups`
- **Windows**: `%LOCALAPPDATA%\hosts-manager\backups`

### Configuration Issues
Reset to default configuration:
```bash
rm ~/.config/hosts-manager/config.yaml  # Linux/macOS
hosts-manager config --show             # Will recreate default config
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Install development tools: `make install-dev-tools`
4. Make your changes
5. Add comprehensive tests
6. Run full validation: `make validate-full`
7. Ensure security checks pass: `make security`
8. Verify test coverage: `make coverage`
9. Submit a pull request

### Code Quality Requirements
All contributions must pass:
- **Linting**: `make lint` (golangci-lint with 30+ enabled linters)
- **Security**: `make security` (gosec, nancy, govulncheck, semgrep)
- **Testing**: Comprehensive test coverage with `make test`
- **Formatting**: Consistent code style with `make fmt`

Use `make pre-commit` to run the essential checks before committing.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Changelog

For detailed release notes and version history, see [CHANGELOG.md](CHANGELOG.md).

See the [latest release](https://github.com/brandonhon/hosts-manager/releases/latest) for current version information and download links.

### Development Status

⚠️ **Development Release**: This project is under active development with version 0.x.x releases. The current implementation includes comprehensive features and security hardening, but the API and behavior may still evolve.

**Version Strategy:**
- **0.x.x releases**: Development versions with evolving features and API changes
- **1.0.0 release**: Planned stable release with locked API and guaranteed backward compatibility

The project is suitable for testing, development environments, and feedback. Use with caution in production until the 1.0.0 stable release.

## Support

- Create an issue on GitHub for bugs and feature requests
- Check existing issues before creating new ones
- Provide system information and error messages when reporting bugs

---

**Note**: Always backup your hosts file before making significant changes. While this tool includes automatic backup functionality, manual backups are recommended for critical systems.