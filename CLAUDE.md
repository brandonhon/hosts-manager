# Claude Development Guidelines

This document provides guidelines and commands for Claude when working on the Hosts Manager project.

## Project Overview

Hosts Manager is a cross-platform CLI tool for managing hosts files with advanced features like templates, backup/restore, interactive TUI, and search capabilities.

## Quick Commands

### Build and Test
```bash
# Build the project
make build

# Run all validation (format, vet, lint, test)
make validate

# Run tests with coverage
make coverage

# Build for all platforms
make release
```

### Development Workflow
```bash
# Quick development cycle
make dev

# Run the application
./build/hosts-manager --help
./build/hosts-manager list
./build/hosts-manager tui

# Test specific functionality
sudo ./build/hosts-manager add 127.0.0.1 test.local --dry-run
```

## Project Structure

```
hosts-manager/
├── cmd/hosts-manager/          # Main CLI application
│   ├── main.go                 # Entry point and root command
│   └── commands.go             # All CLI subcommands
├── internal/                   # Internal packages (not importable)
│   ├── backup/                 # Backup and restore functionality
│   │   └── backup.go           # Backup manager implementation
│   ├── config/                 # Configuration system
│   │   └── config.go           # YAML configuration management
│   ├── hosts/                  # Core hosts file operations
│   │   ├── types.go            # Data structures and constants
│   │   └── parser.go           # Hosts file parsing and writing
│   └── tui/                    # Terminal user interface
│       └── tui.go              # Interactive TUI with Bubble Tea
├── pkg/                        # Public packages (importable)
│   ├── platform/               # Cross-platform abstraction
│   │   └── platform.go         # Platform-specific file paths and permissions
│   └── search/                 # Search functionality
│       └── search.go           # Fuzzy search algorithms
├── build/                      # Build artifacts (gitignored)
├── dist/                       # Distribution packages (gitignored)
└── docs/                       # Generated documentation
```

## Key Components

### 1. Hosts File Parser (`internal/hosts/parser.go`)
- Parses hosts files with category support
- Handles comments, disabled entries, and formatting
- Methods: `Parse()`, `Write()`, `AddEntry()`, `RemoveEntry()`

### 2. Configuration System (`internal/config/config.go`)
- YAML-based configuration
- Default categories and profiles
- Cross-platform config directories
- Methods: `Load()`, `Save()`, `DefaultConfig()`

### 3. Platform Abstraction (`pkg/platform/platform.go`)
- Cross-platform hosts file locations
- Permission handling (sudo/admin)
- Config and data directory paths
- Methods: `GetHostsFilePath()`, `ElevateIfNeeded()`

### 4. TUI Interface (`internal/tui/tui.go`)
- Interactive terminal interface using Bubble Tea
- Navigation, search, and editing capabilities
- Vim-like keybindings support

### 5. Search Engine (`pkg/search/search.go`)
- Fuzzy search with scoring
- Search by hostname, IP, comment, category
- Levenshtein distance algorithm

## Common Development Tasks

### Adding New CLI Commands
1. Add command function to `cmd/hosts-manager/commands.go`
2. Register command in `cmd/hosts-manager/main.go`
3. Update help documentation
4. Add tests

Example:
```go
func newCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "new <args>",
        Short: "Description",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation
            return nil
        },
    }
    return cmd
}
```

### Adding Configuration Options
1. Update `Config` struct in `internal/config/config.go`
2. Add to `DefaultConfig()` function
3. Update YAML marshaling tags
4. Document in README.md

### Adding New Categories
1. Add constant to `internal/hosts/types.go`
2. Update default config in `internal/config/config.go`
3. Document in user documentation

### Platform-Specific Code
Use the platform abstraction:
```go
p := platform.New()
hostsPath := p.GetHostsFilePath()  // Cross-platform
configDir := p.GetConfigDir()      // Cross-platform

if err := p.ElevateIfNeeded(); err != nil {
    return err  // Handle permission elevation
}
```

## Testing Guidelines

### Unit Tests
- Place tests in `*_test.go` files alongside source
- Use table-driven tests for multiple scenarios
- Mock external dependencies (file system, network)

### Integration Tests
- Use temporary files for hosts file operations
- Test cross-platform behavior
- Skip in short test mode: `if testing.Short() { t.Skip() }`

### Testing Commands
```bash
make test              # Run all tests
make coverage          # Generate coverage report
make bench             # Run benchmarks
go test -short ./...   # Quick tests only
```

## Error Handling

Follow Go conventions:
```go
// Wrap errors with context
if err := parser.Parse(); err != nil {
    return fmt.Errorf("failed to parse hosts file: %w", err)
}

// Handle permission errors gracefully
if err := platform.ElevateIfNeeded(); err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    fmt.Fprintf(os.Stderr, "Try running with elevated privileges\n")
    return err
}
```

## Logging and Output

### User Output
```go
// Success messages
fmt.Printf("Added entry: %s -> %v\n", ip, hostnames)

// Errors to stderr
fmt.Fprintf(os.Stderr, "Error: %v\n", err)

// Verbose output (check verbose flag)
if verbose {
    fmt.Println("Backup created successfully")
}
```

### Dry Run Mode
Always check the `dryRun` flag:
```go
if dryRun {
    fmt.Printf("Would add: %s %s\n", ip, hostnames)
    return nil
}
// Actual implementation
```

## Security Considerations

1. **Input Validation**: Validate all IP addresses and hostnames
2. **File Permissions**: Check write permissions before operations
3. **Backup Before Changes**: Always create backup before modifications
4. **Path Traversal**: Use filepath.Clean() for user-provided paths
5. **Privilege Escalation**: Only elevate when necessary

Example:
```go
// Validate IP address
if net.ParseIP(ip) == nil {
    return fmt.Errorf("invalid IP address: %s", ip)
}

// Validate hostname
if !isValidHostname(hostname) {
    return fmt.Errorf("invalid hostname: %s", hostname)
}
```

## Dependencies

### Core Dependencies
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - TUI styling

### Development Dependencies
- `golangci-lint` - Linting
- `gosec` - Security analysis

## Release Process

### Version Updates
Update version in:
- Git tags (`git tag -a v1.2.3`)
- Makefile VERSION variable
- Documentation references

### Build Release
```bash
make clean
make validate
make release
make dist
```

## Debugging Tips

### Enable Verbose Output
```bash
./build/hosts-manager --verbose list
```

### Test with Dry Run
```bash
sudo ./build/hosts-manager --dry-run add 127.0.0.1 test.local
```

### Debug TUI Mode
Add debug output to TUI model updates:
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Add temporary debug output
    fmt.Printf("DEBUG: received message %T\n", msg)
    // ... rest of update logic
}
```

### Test Parser with Sample Files
```bash
# Create test hosts file
cat > test-hosts << EOF
127.0.0.1 localhost
# @category development
192.168.1.100 api.dev
EOF

# Test parser
go run cmd/hosts-manager/main.go list
```

## Performance Notes

1. **File I/O**: Minimize hosts file reads/writes
2. **Search**: Use efficient algorithms for large hosts files
3. **Memory**: Avoid loading entire hosts file into memory for simple operations
4. **Cross-platform**: Cache platform detection results

## Troubleshooting

### Common Build Issues
```bash
# Go module issues
go mod tidy
go mod download

# Build cache issues
go clean -cache
make clean && make build

# Permission issues (Linux/macOS)
sudo make install
```

### Runtime Issues
```bash
# Config issues
hosts-manager config --show

# Permission issues
sudo hosts-manager list  # Test with elevation

# Backup issues
ls -la ~/.local/share/hosts-manager/backups/  # Check backup directory
```

## Code Style

Follow standard Go conventions:
- Use `gofmt` (make fmt)
- Follow `go vet` recommendations (make vet)
- Use meaningful names
- Keep functions small and focused
- Add comments for exported functions
- Use early returns to reduce nesting

## Documentation Updates

When adding features, update:
1. `README.md` - User-facing documentation
2. `EXAMPLES.md` - Usage examples
3. Command help text in source code
4. This `CLAUDE.md` file for development guidelines

## Future Enhancements

Potential areas for improvement:
- Shell completion scripts
- Plugin system for custom categories
- Web interface for remote management
- Integration with cloud DNS services
- Automatic host discovery
- Performance optimizations for large files