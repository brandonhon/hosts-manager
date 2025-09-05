import re
from pathlib import Path
from shutil import copy2
import difflib

class HostsEditor:
    def __init__(self, path):
        self.path = Path(path)
        if not self.path.exists():
            raise FileNotFoundError(f"Hosts file not found: {self.path}")
        self.lines = self._read_file()

    def _read_file(self):
        return self.path.read_text().splitlines()

    def _show_diff(self, new_lines):
        """Show diff between old and new hosts file."""
        diff = difflib.unified_diff(
            self.lines,
            new_lines,
            fromfile=str(self.path),
            tofile=f"{self.path} (updated)",
            lineterm=""
        )
        diff_output = list(diff)

        if not diff_output:
            print("✅ No changes to apply")
            return False

        print("\n🔍 Proposed changes:\n")
        for line in diff_output:
            if line.startswith("+") and not line.startswith("+++"):
                print(f"\033[92m{line}\033[0m")  # Green for additions
            elif line.startswith("-") and not line.startswith("---"):
                print(f"\033[91m{line}\033[0m")  # Red for removals
            else:
                print(line)
        print("\n")
        return True

    def _write_file(self, new_lines=None):
        """Preview and write changes with diff."""
        new_lines = new_lines or self.lines
        has_changes = self._show_diff(new_lines)
        if not has_changes:
            return

        confirm = input("Apply these changes? [y/N]: ").strip().lower()
        if confirm != "y":
            print("❌ Changes discarded")
            return

        backup_path = f"{self.path}.bak"
        copy2(self.path, backup_path)
        self.path.write_text("\n".join(new_lines) + "\n")
        self.lines = new_lines
        print(f"✅ Changes applied (backup at {backup_path})")

    def list_entries(self):
        for i, line in enumerate(self.lines, 1):
            print(f"{i:03}: {line}")

    def _find_section_index(self, section):
        for idx, line in enumerate(self.lines):
            if line.strip().lower() == f"# {section}".lower():
                return idx
        return None

    def list_sections(self):
        sections = []
        current_section = None
        current_sites = []

        for line in self.lines:
            stripped_line = line.strip()

            # Check for section headers (e.g., "# DEVELOPMENT #")
            if stripped_line.startswith("#") and stripped_line.endswith("#") and len(stripped_line) > 2:
                section_name = stripped_line.strip("#").strip()
                if section_name and section_name.isupper():
                    # Save the previous section and its sites, if any
                    if current_section:
                        sections.append((current_section, current_sites))
                        current_sites = []
                    current_section = section_name

            # Check for site names in comments (e.g., "# booger.com")
            elif stripped_line.startswith("#") and not stripped_line.startswith("##"):
                # Extract potential site name, ignoring inline comments like "# p-yoyo1"
                site_name = stripped_line[1:].split("#")[0].strip()
                # Ensure it's a valid site name (not empty, not an IP address line)
                if site_name and not any(site_name.startswith(d) for d in "0123456789") and current_section:
                    current_sites.append(site_name)

        # Append the last section and its sites
        if current_section:
            sections.append((current_section, current_sites))

        # Print the results
        if sections:
            print("Sections found:")
            for section_name, sites in sections:
                print(f"  📁 - {section_name}")
                for site in sites:
                    print(f"      🌐 - {site}")
        else:
            print("⚠️ No sections found")

    def add_entry(self, ip, hostname, section=None, comment=None):
        new_lines = list(self.lines)
        entry = f"{ip}\t{hostname}"
        if comment:
            entry += f"  # {comment}"

        if section:
            idx = self._find_section_index(section)
            if idx is None:
                new_lines.append(f"\n# {section}")
                new_lines.append(entry)
            else:
                new_lines.insert(idx + 1, entry)
        else:
            new_lines.append(entry)

        self._write_file(new_lines)

    def delete_entry(self, hostname):
        new_lines = [l for l in self.lines if hostname not in l]
        if new_lines == self.lines:
            print(f"⚠️ No entry found for {hostname}")
            return
        self._write_file(new_lines)

    def comment_entry(self, hostname):
        new_lines = []
        updated = False
        for line in self.lines:
            if hostname in line and not line.strip().startswith("#"):
                new_lines.append("# " + line)
                updated = True
            else:
                new_lines.append(line)
        if updated:
            self._write_file(new_lines)
        else:
            print(f"⚠️ No active entry found for {hostname}")

    def uncomment_entry(self, hostname):
        new_lines = []
        updated = False
        for line in self.lines:
            if hostname in line and line.strip().startswith("#"):
                new_lines.append(re.sub(r"^#\s*", "", line))
                updated = True
            else:
                new_lines.append(line)
        if updated:
            self._write_file(new_lines)
        else:
            print(f"⚠️ No commented entry found for {hostname}")

    def search_entries(self, query):
        found = False
        for i, line in enumerate(self.lines, 1):
            if query in line:
                print(f"{i:03}: {line}")
                found = True
        if not found:
            print(f"⚠️ No results for '{query}'")

    def toggle_section(self, section, site=None, enable=True):
        # Find the top-level section header
        start_idx = None
        for i, line in enumerate(self.lines):
            stripped_line = line.strip()
            if stripped_line.startswith("#") and stripped_line.endswith("#") and len(stripped_line) > 2:
                section_name = stripped_line.strip("#").strip()
                if section_name == section.upper():
                    start_idx = i
                    break

        if start_idx is None:
            print(f"⚠️ Section '{section}' not found")
            return 0

        new_lines = list(self.lines)
        updated = 0
        current_site = None

        for idx in range(start_idx + 1, len(new_lines)):
            line = new_lines[idx]
            stripped_line = line.strip()

            # Stop at the next top-level section header
            if stripped_line.startswith("#") and stripped_line.endswith("#") and len(stripped_line) > 2:
                break

            # Detect site name (e.g., "# booger.com")
            if stripped_line.startswith("#") and not any(stripped_line.startswith(f"# {d}") for d in "0123456789"):
                current_site = stripped_line[1:].split("#")[0].strip()
                continue

            # Skip empty lines or non-IP lines
            if stripped_line == "" or (stripped_line.startswith("#") and not any(stripped_line.startswith(f"# {d}") for d in "0123456789")):
                continue

            # Toggle IP lines for the specified site (or all if site is None)
            if (site is None or (current_site and current_site == site)):
                if enable and stripped_line.startswith("#"):
                    new_lines[idx] = re.sub(r"^#\s*", "", line)
                    updated += 1
                elif not enable and not stripped_line.startswith("#"):
                    new_lines[idx] = "# " + line
                    updated += 1

        if updated:
            self._write_file(new_lines)
        else:
            if site:
                print(f"⚠️ No changes needed for site '{site}' in section '{section}'")
            else:
                print(f"⚠️ No changes needed for section '{section}'")
