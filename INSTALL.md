# Installation Guide

This guide covers different ways to install and set up the Hosts Manager.

## System Requirements

- **Operating Systems**: Linux, macOS, Windows
- **Architecture**: amd64, arm64
- **Go version**: 1.19+ (if building from source)
- **Privileges**: Administrator/root access for hosts file modifications

## Installation Methods

### 1. Pre-compiled Binaries (Recommended)

Download the latest release for your platform:

#### Linux (x64)
```bash
curl -L https://github.com/your-username/hosts-manager/releases/latest/download/hosts-manager-linux-amd64.tar.gz | tar xz
sudo mv hosts-manager /usr/local/bin/
```

#### Linux (ARM64)
```bash
curl -L https://github.com/your-username/hosts-manager/releases/latest/download/hosts-manager-linux-arm64.tar.gz | tar xz
sudo mv hosts-manager /usr/local/bin/
```

#### macOS (Intel)
```bash
curl -L https://github.com/your-username/hosts-manager/releases/latest/download/hosts-manager-darwin-amd64.tar.gz | tar xz
sudo mv hosts-manager /usr/local/bin/
```

#### macOS (Apple Silicon)
```bash
curl -L https://github.com/your-username/hosts-manager/releases/latest/download/hosts-manager-darwin-arm64.tar.gz | tar xz
sudo mv hosts-manager /usr/local/bin/
```

#### Windows
1. Download `hosts-manager-windows-amd64.exe.zip`
2. Extract to a folder in your PATH (e.g., `C:\Windows\System32\`)
3. Rename to `hosts-manager.exe`

### 2. From Source

#### Prerequisites
```bash
# Install Go
# Linux/macOS
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Or use your package manager
# Ubuntu/Debian
sudo apt install golang-go

# macOS with Homebrew
brew install go

# Windows - Download from https://golang.org/dl/
```

#### Build and Install
```bash
git clone https://github.com/your-username/hosts-manager.git
cd hosts-manager

# Build
make build

# Install globally
sudo make install

# Or install to custom location
make build
cp build/hosts-manager ~/bin/  # Make sure ~/bin is in your PATH
```

### 3. Package Managers

#### Homebrew (macOS/Linux)
```bash
# Add tap (once repository is published)
brew tap your-username/hosts-manager
brew install hosts-manager
```

#### Scoop (Windows)
```powershell
# Add bucket (once repository is published)
scoop bucket add hosts-manager https://github.com/your-username/scoop-hosts-manager
scoop install hosts-manager
```

#### Snap (Linux)
```bash
# Once published to Snap Store
sudo snap install hosts-manager
```

## Post-Installation Setup

### 1. Verify Installation
```bash
hosts-manager --help
hosts-manager --version
```

### 2. Initialize Configuration
```bash
# This will create default configuration if it doesn't exist
hosts-manager config --show
```

### 3. Test Basic Functionality
```bash
# List current entries (should work without sudo)
hosts-manager list

# Test backup creation (requires sudo)
sudo hosts-manager backup
```

## Platform-Specific Setup

### Linux

#### Permissions
The hosts file (`/etc/hosts`) requires root privileges:
```bash
# Always use sudo for modifications
sudo hosts-manager add 127.0.0.1 myapp.local
```

#### Shell Completion (Optional)
```bash
# Bash
hosts-manager completion bash | sudo tee /etc/bash_completion.d/hosts-manager

# Zsh
mkdir -p ~/.zsh/completions
hosts-manager completion zsh > ~/.zsh/completions/_hosts-manager
# Add to .zshrc: fpath=(~/.zsh/completions $fpath)
```

### macOS

#### Permissions
Similar to Linux, requires `sudo`:
```bash
sudo hosts-manager add 127.0.0.1 myapp.local
```

#### Homebrew Integration
If installed via Homebrew, completions are automatically set up.

### Windows

#### Running as Administrator
Always run PowerShell or Command Prompt as Administrator for hosts file modifications.

#### PowerShell Execution Policy
You may need to adjust execution policy:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

## Configuration Directories

After first run, configuration directories are created:

- **Linux**: `~/.config/hosts-manager/`
- **macOS**: `~/.config/hosts-manager/`
- **Windows**: `%APPDATA%\hosts-manager\`

Data directories (backups, history):

- **Linux**: `~/.local/share/hosts-manager/`
- **macOS**: `~/Library/Application Support/hosts-manager/`
- **Windows**: `%LOCALAPPDATA%\hosts-manager\`

## Troubleshooting

### Common Issues

#### "Permission denied" error
```bash
# Use sudo (Linux/macOS)
sudo hosts-manager add 127.0.0.1 example.local

# Run as Administrator (Windows)
# Right-click PowerShell -> "Run as Administrator"
```

#### "Command not found"
```bash
# Check if binary is in PATH
echo $PATH

# Add to PATH (Linux/macOS)
export PATH=$PATH:/usr/local/bin
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc

# Windows - Add to System PATH through Control Panel
```

#### Go version conflicts (when building from source)
```bash
# Check Go version
go version

# Ensure Go 1.19+
# Remove old Go installations before installing newer version
```

### Verification Steps

1. **Check installation**:
   ```bash
   which hosts-manager  # Linux/macOS
   where hosts-manager  # Windows
   ```

2. **Test permissions**:
   ```bash
   sudo hosts-manager list  # Should show current hosts entries
   ```

3. **Verify configuration**:
   ```bash
   hosts-manager config --show
   ```

## Uninstallation

### From Binary Installation
```bash
# Remove binary
sudo rm /usr/local/bin/hosts-manager  # Linux/macOS
# Or delete from Windows PATH location

# Remove configuration (optional)
rm -rf ~/.config/hosts-manager/          # Linux/macOS
rm -rf ~/.local/share/hosts-manager/     # Linux
rm -rf ~/Library/Application\ Support/hosts-manager/  # macOS
# Windows: Delete %APPDATA%\hosts-manager\ and %LOCALAPPDATA%\hosts-manager\
```

### From Package Managers
```bash
# Homebrew
brew uninstall hosts-manager

# Scoop
scoop uninstall hosts-manager

# Snap
sudo snap remove hosts-manager
```

## Development Setup

For contributors and developers:

```bash
git clone https://github.com/your-username/hosts-manager.git
cd hosts-manager

# Install development dependencies
make init

# Run development build
make dev

# Run tests
make test

# Run with development binary
./build/hosts-manager-dev --help
```

## Next Steps

After installation, see:
- [README.md](README.md) - Usage guide and examples
- [EXAMPLES.md](EXAMPLES.md) - Common use cases and workflows
- `hosts-manager --help` - Built-in help system