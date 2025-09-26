# Hosts Manager

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

### From Source

```bash
git clone https://github.com/your-username/hosts-manager.git
cd hosts-manager
make build
sudo make install
```

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/your-username/hosts-manager/releases).

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
- `s` - Save changes (shows confirmation)
- `/` - Search mode
- `r` - Refresh
- `?` - Help
- `q` - Quit

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
- Make (optional)

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
make lint        # Run linters
make fmt         # Format code
make vet         # Vet code
make validate    # Full validation pipeline
```

### Cross-Compilation
```bash
make release     # Build for all platforms
make dist        # Create distribution packages
```

## Security Considerations

- Always creates backups before modifications
- Validates IP addresses and hostnames
- Prevents malicious entries
- Supports dry-run mode for safe testing
- Lock file prevents concurrent modifications
- Permission elevation only when needed

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
3. Make your changes
4. Add tests
5. Run `make validate`
6. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Changelog

### v1.0.0 (2023-12-07)
- Initial release
- Cross-platform hosts file management
- Interactive TUI mode
- Template system with categories
- Backup and restore functionality
- Export/import capabilities
- Fuzzy search
- Profile system
- Configuration management

## Support

- Create an issue on GitHub for bugs and feature requests
- Check existing issues before creating new ones
- Provide system information and error messages when reporting bugs

---

**Note**: Always backup your hosts file before making significant changes. While this tool includes automatic backup functionality, manual backups are recommended for critical systems.