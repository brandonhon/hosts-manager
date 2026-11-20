# Configuration, data and the hosts file

Where settings and data live, every config setting, and how hosts-manager
lays out the hosts file.

## The config file

`hosts-manager config` prints the config file's path and current settings.
The file is created with defaults on first run:

```yaml
general:
  default_category: custom   # used by add without -c
  auto_backup: true          # back up before every change
  dry_run: false             # default for --dry-run
  verbose: false             # default for --verbose
profiles:
  minimal:
    description: Minimal profile with essential entries only
    categories: [production]
    default: false
  development:
    description: Development profile with dev and staging environments
    categories: [development, staging]
    default: false
  full:
    description: Full profile with all categories enabled
    categories: [development, staging, production, custom]
    default: true            # exactly one profile is the default
backup:
  directory: ""              # empty: <data dir>/backups
  max_backups: 10            # 1-100
  retention_days: 30         # 1-3650
  compression_type: gzip     # gzip or none
export:
  default_format: yaml       # json, yaml or hosts
```

## Where config and data live

| | Linux | macOS | Windows |
|---|---|---|---|
| Config | `~/.config/hosts-manager` | `~/.config/hosts-manager` | `%APPDATA%\hosts-manager` |
| Data | `~/.local/share/hosts-manager` | `~/Library/Application Support/hosts-manager` | `%LOCALAPPDATA%\hosts-manager` |

`XDG_CONFIG_HOME` and `XDG_DATA_HOME` are honored on Linux and macOS. Under
`sudo` these are the invoking user's directories, not root's.

The data directory holds `backups/` and `audit/changes.log`, which records one
line per change:

```
2026-11-19T15:04:05Z  alice  add  ok  Added 10.0.0.5 -> api.dev web.dev [dev]
```

## The hosts file

hosts-manager keeps everything a plain hosts file. Categories are comment
markers, and a disabled entry is commented out, so the file stays valid for
the operating system and readable by hand:

```
# @category dev Development services
# =============== DEV ===============
10.0.0.5 api.dev web.dev # dev box
# 10.0.0.6 old.dev

# @category blocked
# =============== BLOCKED ===============
0.0.0.0 ads.example.com
```

Lines it does not recognize, such as comments at the top of the file, are kept.

| | Linux / macOS | Windows |
|---|---|---|
| Hosts file | `/etc/hosts` | `C:\Windows\System32\drivers\etc\hosts` |
