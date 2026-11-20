# Installing hosts-manager

Each [release](https://github.com/brandonhon/hosts-manager/releases) has an
archive per platform plus a `SHA256SUMS` file:

| Platform | Archive |
|---|---|
| Linux, Intel/AMD | `hosts-manager-<version>-linux-amd64.tar.gz` |
| Linux, ARM | `hosts-manager-<version>-linux-arm64.tar.gz` |
| macOS, Apple silicon | `hosts-manager-<version>-darwin-arm64.tar.gz` |
| macOS, Intel | `hosts-manager-<version>-darwin-amd64.tar.gz` |
| Windows | `hosts-manager-<version>-windows-amd64.zip` |

The steps below download the right one, check it against `SHA256SUMS`, and put
the binary on your `PATH`. Set `VERSION` to the release you want, such as
`v0.7.0`.

## Linux

```bash
VERSION=vX.Y.Z
ARCH=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
base=https://github.com/brandonhon/hosts-manager/releases/download/$VERSION
curl -fsSLO "$base/hosts-manager-$VERSION-linux-$ARCH.tar.gz" -O "$base/SHA256SUMS"
grep "linux-$ARCH.tar.gz" SHA256SUMS | sha256sum -c
tar -xzf "hosts-manager-$VERSION-linux-$ARCH.tar.gz" hosts-manager
sudo install -m 0755 hosts-manager /usr/local/bin/
```

`sha256sum -c` must print `OK`. If it reports a mismatch, delete the download
and try again; do not install it.

**If `sudo hosts-manager` says "command not found":** sudo searches its own
`secure_path`, not your `PATH`, and on RHEL, Fedora, Kali and current Debian
testing/unstable that leaves out `/usr/local/bin`. Run it as
`sudo /usr/local/bin/hosts-manager`, or add `/usr/local/bin` to `secure_path`
with `sudo visudo`. The tool's own "run with sudo" hints always give the full
path.

## macOS

```bash
VERSION=vX.Y.Z
ARCH=$(uname -m | sed 's/x86_64/amd64/')   # arm64 on Apple silicon
base=https://github.com/brandonhon/hosts-manager/releases/download/$VERSION
curl -fsSLO "$base/hosts-manager-$VERSION-darwin-$ARCH.tar.gz" -O "$base/SHA256SUMS"
grep "darwin-$ARCH.tar.gz" SHA256SUMS | shasum -a 256 -c
tar -xzf "hosts-manager-$VERSION-darwin-$ARCH.tar.gz" hosts-manager
sudo install -d /usr/local/bin
sudo install -m 0755 hosts-manager /usr/local/bin/
```

`sudo install -d` creates `/usr/local/bin`, which a new Apple silicon Mac may
not have; Homebrew there uses `/opt/homebrew` instead.

The binary is not signed by Apple. Downloaded with `curl` as above, it runs
as is. If you downloaded the archive in a browser, macOS quarantines it and
refuses to open it; clear the flag before installing:

```bash
xattr -d com.apple.quarantine hosts-manager
```

## Windows

Run these in PowerShell opened with **Run as administrator**:

```powershell
$version = "vX.Y.Z"
$base = "https://github.com/brandonhon/hosts-manager/releases/download/$version"
$zip = "hosts-manager-$version-windows-amd64.zip"
Invoke-WebRequest "$base/$zip" -OutFile $zip
Invoke-WebRequest "$base/SHA256SUMS" -OutFile SHA256SUMS
$expected = ((Select-String -Path SHA256SUMS -Pattern $zip -SimpleMatch).Line -split '\s+')[0]
if ((Get-FileHash $zip -Algorithm SHA256).Hash -ne $expected) { throw "Checksum mismatch for $zip" }

$dir = "$env:ProgramFiles\hosts-manager"
Expand-Archive $zip -DestinationPath $dir -Force
$path = [Environment]::GetEnvironmentVariable("Path", "Machine")
if ($path -notlike "*$dir*") {
    [Environment]::SetEnvironmentVariable("Path", "$path;$dir", "Machine")
}
```

Open a new Administrator terminal so it picks up the new `PATH`, then run
`hosts-manager`. Commands that change the hosts file, and the interactive
editor, need an Administrator terminal every time.

## From source

With Go 1.24 or newer:

```bash
git clone https://github.com/brandonhon/hosts-manager.git
cd hosts-manager
make build && sudo make install
```

Run `make build` as yourself; `sudo make install` only copies the binary it
built into `/usr/local/bin`.

## Upgrading

Repeat the steps for your platform with the new `VERSION`. They replace the
binary in place; your config, backups and audit log are untouched.

## Uninstalling

- Linux and macOS: `sudo rm /usr/local/bin/hosts-manager`
- Windows: delete `C:\Program Files\hosts-manager` and remove it from the
  system `PATH`.

Your config, backups and audit log stay where they are. Their locations are
listed in [configuration.md](configuration.md#where-config-and-data-live);
delete those folders too if you want everything gone. Your hosts file keeps
its current entries.
