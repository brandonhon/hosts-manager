from InquirerPy import inquirer

def interactive_menu(editor):
    """Interactive menu for managing hosts file"""
    while True:
        choice = inquirer.select(
            message="Choose an action:",
            choices=[
                "List entries",
                "List sections",
                "Add entry",
                "Delete entry",
                "Comment entry",
                "Uncomment entry",
                "Search",
                "Enable section",
                "Disable section",
                "Quit"
            ],
        ).execute()

        if choice == "List entries":
            editor.list_entries()
        elif choice == "List sections":
            editor.list_sections()
        elif choice == "Add entry":
            ip = inquirer.text(message="Enter IP:").execute()
            hostname = inquirer.text(message="Enter hostname:").execute()
            section = inquirer.text(message="Section (optional):").execute()
            comment = inquirer.text(message="Comment (optional):").execute()
            editor.add_entry(ip, hostname, section or None, comment or None)
        elif choice == "Delete entry":
            hostname = inquirer.text(message="Hostname to delete:").execute()
            editor.delete_entry(hostname)
        elif choice == "Comment entry":
            hostname = inquirer.text(message="Hostname to comment:").execute()
            editor.comment_entry(hostname)
        elif choice == "Uncomment entry":
            hostname = inquirer.text(message="Hostname to uncomment:").execute()
            editor.uncomment_entry(hostname)
        elif choice == "Search":
            query = inquirer.text(message="Search query:").execute()
            editor.search_entries(query)
        elif choice == "Enable section":
            section = inquirer.text(message="Section name:").execute()
            editor.toggle_section(section, enable=True)
        elif choice == "Disable section":
            section = inquirer.text(message="Section name:").execute()
            editor.toggle_section(section, enable=False)
        else:
            print("👋 Exiting hosts-manager")
            break
