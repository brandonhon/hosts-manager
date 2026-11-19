# Security

## Reporting a vulnerability

Please report security issues privately through
[GitHub's private vulnerability reporting](https://github.com/brandonhon/hosts-manager/security/advisories/new),
not in a public issue. Include the version (`hosts-manager --version`), your
platform, and steps to reproduce.

You should get a response within a week. Fixes are released as a new version,
and the advisory is published once a fixed release is available.

## Supported versions

Only the latest release receives security fixes.

## Scope

hosts-manager edits the system hosts file and must run with root or
Administrator privileges to do so. Issues of particular interest:

- a way to change the hosts file without those privileges
- input (entries, comments, import files) that writes host mappings it should
  not, or corrupts the file
- files created under `sudo` ending up owned by, or readable by, the wrong user
