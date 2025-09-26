# Contributing to Hosts Manager

Thank you for your interest in contributing to Hosts Manager! This document provides guidelines and instructions for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Process](#contributing-process)
- [Code Style and Standards](#code-style-and-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Release Process](#release-process)

## Code of Conduct

This project follows the [Go Community Code of Conduct](https://golang.org/conduct). Please be respectful and inclusive in all interactions.

## Getting Started

### Prerequisites

- Go 1.19 or higher
- Git
- Make (optional but recommended)
- golangci-lint (for linting)
- gosec (for security checks)

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/your-username/hosts-manager.git
   cd hosts-manager
   ```

2. **Install Dependencies**
   ```bash
   make init  # Installs development tools
   make deps  # Downloads Go dependencies
   ```

3. **Verify Setup**
   ```bash
   make validate  # Runs fmt, vet, lint, and test
   ```

4. **Build and Test**
   ```bash
   make build     # Build binary
   make test      # Run tests
   make coverage  # Generate coverage report
   ```

## Contributing Process

### 1. Issue First

Before starting work, please:
- Check existing issues to avoid duplication
- Create an issue describing the bug or feature
- Discuss the approach with maintainers
- Wait for approval before starting significant work

### 2. Branch Naming

Use descriptive branch names:
- `feature/add-xxx` for new features
- `bugfix/fix-xxx` for bug fixes
- `docs/update-xxx` for documentation
- `refactor/improve-xxx` for refactoring

### 3. Making Changes

```bash
# Create feature branch
git checkout -b feature/add-new-command

# Make your changes
# ...

# Test your changes
make validate

# Commit with descriptive message
git commit -m "feat: add new command for bulk operations

- Implement bulk add/remove operations
- Add tests for bulk operations
- Update documentation
- Closes #123"
```

### 4. Commit Messages

Follow [Conventional Commits](https://conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat`: new feature
- `fix`: bug fix
- `docs`: documentation only
- `style`: formatting, no code change
- `refactor`: code change that neither fixes bug nor adds feature
- `test`: adding missing tests
- `chore`: changes to build process or auxiliary tools

Examples:
```
feat(tui): add vim-style navigation keybindings
fix(parser): handle malformed IP addresses correctly
docs(readme): update installation instructions
```

### 5. Pull Request Process

1. **Create Pull Request**
   - Use descriptive title and description
   - Link related issues with "Closes #123"
   - Fill out the PR template completely

2. **PR Requirements**
   - All tests pass
   - Code coverage maintained or improved
   - Documentation updated
   - No linting errors
   - Commit messages follow convention

3. **Review Process**
   - Address all review comments
   - Update commits or create fixup commits
   - Request re-review after changes

## Code Style and Standards

### Go Style Guidelines

Follow standard Go practices:
- Use `go fmt` (automated in make commands)
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions small and focused

### Project-Specific Standards

1. **Error Handling**
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to parse hosts file: %w", err)
   }

   // Avoid
   if err != nil {
       panic(err)
   }
   ```

2. **Logging and Output**
   ```go
   // Use structured logging
   if verbose {
       fmt.Println("Backup created successfully")
   }

   // For errors, use stderr
   fmt.Fprintf(os.Stderr, "Error: %v\n", err)
   ```

3. **Configuration**
   - Use the existing config system
   - Add new options to `internal/config/config.go`
   - Update default configuration appropriately

### Code Organization

```
cmd/hosts-manager/     # Main application
internal/             # Internal packages
├── config/          # Configuration management
├── hosts/           # Hosts file operations
├── tui/             # Terminal user interface
└── backup/          # Backup and restore
pkg/                 # Reusable packages
├── platform/        # Platform-specific code
└── search/          # Search functionality
```

## Testing

### Running Tests

```bash
make test         # Run all tests
make coverage     # Generate coverage report
make bench        # Run benchmarks
```

### Writing Tests

1. **Test File Naming**
   ```
   parser.go      -> parser_test.go
   config.go      -> config_test.go
   ```

2. **Test Function Naming**
   ```go
   func TestParser_Parse(t *testing.T) { ... }
   func TestConfig_Load(t *testing.T) { ... }
   ```

3. **Table-Driven Tests**
   ```go
   func TestFormatEntry(t *testing.T) {
       tests := []struct {
           name     string
           entry    Entry
           expected string
       }{
           {
               name: "basic entry",
               entry: Entry{
                   IP: "127.0.0.1",
                   Hostnames: []string{"localhost"},
                   Enabled: true,
               },
               expected: "127.0.0.1 localhost",
           },
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               result := formatEntry(tt.entry)
               if result != tt.expected {
                   t.Errorf("expected %q, got %q", tt.expected, result)
               }
           })
       }
   }
   ```

4. **Test Coverage**
   - Aim for >80% coverage
   - Focus on critical paths and edge cases
   - Don't sacrifice readability for coverage

### Integration Tests

For tests that modify system files:

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    // Use temporary files for testing
    tmpFile := createTempHostsFile(t)
    defer os.Remove(tmpFile)

    // Run test with temporary file
}
```

## Documentation

### Code Documentation

- Document all exported functions, types, and packages
- Use Go doc conventions
- Include examples for complex functions

```go
// ParseHostsFile parses a hosts file and returns a structured representation.
// It handles both enabled and disabled entries, organizing them by category.
//
// Example:
//   parser := hosts.NewParser("/etc/hosts")
//   hostsFile, err := parser.Parse()
//   if err != nil {
//       return err
//   }
func (p *Parser) Parse() (*HostsFile, error) { ... }
```

### User Documentation

- Update README.md for new features
- Add examples to EXAMPLES.md
- Update command help text
- Consider adding blog posts for major features

### CLI Help

Update help text in cobra commands:

```go
cmd := &cobra.Command{
    Use:   "add <ip> <hostname> [hostname...]",
    Short: "Add a new hosts entry",
    Long: `Add a new entry to the hosts file with the specified IP and hostnames.

The entry will be added to the specified category (default: custom) and can
include a comment for documentation purposes.`,
    Example: `  # Add simple entry
  hosts-manager add 127.0.0.1 myapp.local

  # Add with category and comment
  hosts-manager add 192.168.1.100 api.dev --category development --comment "Dev API"`,
}
```

## Release Process

### Version Numbering

We follow [Semantic Versioning](https://semver.org/):
- `MAJOR.MINOR.PATCH`
- Major: breaking changes
- Minor: new features, backwards compatible
- Patch: bug fixes, backwards compatible

### Release Checklist

1. **Pre-release**
   ```bash
   # Update version in relevant files
   # Update CHANGELOG.md
   # Run full test suite
   make ci
   ```

2. **Create Release**
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

3. **Build Release Assets**
   ```bash
   make release  # Cross-compile for all platforms
   make dist     # Create distribution packages
   ```

4. **GitHub Release**
   - Create GitHub release from tag
   - Upload distribution packages
   - Write release notes

## Getting Help

- **Questions**: Open a GitHub issue with the "question" label
- **Bugs**: Open a GitHub issue with the "bug" label
- **Features**: Open a GitHub issue with the "enhancement" label
- **Security**: Email maintainers privately for security issues

## Recognition

Contributors are recognized in:
- CONTRIBUTORS.md file
- GitHub contributors page
- Release notes for significant contributions

Thank you for contributing to Hosts Manager!