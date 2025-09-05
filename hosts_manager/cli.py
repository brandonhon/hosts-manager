import click
from .editor import HostsEditor
from .interactive import interactive_menu

@click.group()
@click.option("--file", default="/etc/hosts", help="Path to hosts file")
@click.pass_context
def cli(ctx, file):
    """Hosts file manager"""
    ctx.obj = HostsEditor(file)

@cli.command()
@click.pass_obj
def list(editor):
    """List all hosts entries"""
    editor.list_entries()

@cli.command()
@click.argument("ip")
@click.argument("hostname")
@click.option("--section", default=None, help="Optional section name")
@click.option("--comment", default=None, help="Optional comment")
@click.pass_obj
def add(editor, ip, hostname, section, comment):
    """Add a new host entry"""
    editor.add_entry(ip, hostname, section, comment)

@cli.command()
@click.argument("hostname")
@click.pass_obj
def delete(editor, hostname):
    """Delete a host entry"""
    editor.delete_entry(hostname)

@cli.command()
@click.argument("hostname")
@click.pass_obj
def disable(editor, hostname):
    """Comment out a hosts entry"""
    editor.comment_entry(hostname)

@cli.command()
@click.argument("hostname")
@click.pass_obj
def enable(editor, hostname):
    """Uncomment a hosts entry"""
    editor.uncomment_entry(hostname)

@cli.command()
@click.option("--query", default=None, help="Search by hostname or IP")
@click.pass_obj
def search(editor, query):
    """Search for hosts by name or IP"""
    editor.search_entries(query)

@cli.command()
@click.pass_obj
def list_sections(editor):
    """List all available sections"""
    editor.list_sections()

@cli.command()
@click.argument("section")
@click.option("--site", default=None, help="Optional site within the section to disable (e.g., booger.com)")
@click.pass_obj
def disable_section(editor, section, site):
    """Disable (comment) all entries in a section or a specific site within a section."""
    updated = editor.toggle_section(section, site=site, enable=False)
    if updated:
        print(f"Disabled {updated} entries in section '{section}'" + (f" for site '{site}'" if site else ""))

@cli.command()
@click.argument("section")
@click.option("--site", default=None, help="Optional site within the section to enable (e.g., booger.com)")
@click.pass_obj
def enable_section(editor, section, site):
    """Enable (uncomment) all entries in a section or a specific site within a section."""
    updated = editor.toggle_section(section, site=site, enable=True)
    if updated:
        print(f"Enabled {updated} entries in section '{section}'" + (f" for site '{site}'" if site else ""))

@cli.command()
@click.pass_obj
def interactive(editor):
    """Run interactive TUI menu"""
    interactive_menu(editor)
