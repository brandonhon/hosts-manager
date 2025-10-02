# Examples and Use Cases

This document provides practical examples and common workflows for using the Hosts Manager.

## Table of Contents

- [Basic Usage](#basic-usage)
- [Development Workflows](#development-workflows)
- [Team Collaboration](#team-collaboration)
- [Production Management](#production-management)
- [Advanced Use Cases](#advanced-use-cases)
- [Automation and Scripting](#automation-and-scripting)

## Basic Usage

### Adding Your First Entry

```bash
# Add a simple entry
sudo hosts-manager add 127.0.0.1 myapp.local

# Add with category and comment
sudo hosts-manager add 192.168.1.100 api.dev --category development --comment "Development API server"

# Add multiple hostnames for same IP
sudo hosts-manager add 192.168.1.200 web.staging api.staging admin.staging --category staging
```

### Managing Entries

```bash
# List all entries
hosts-manager list

# List only development entries
hosts-manager list --category development

# Search for entries
hosts-manager search "api"
hosts-manager search "192.168" --fuzzy

# Enable/disable entries
sudo hosts-manager enable myapp.local
sudo hosts-manager disable api.staging

# Delete entries
sudo hosts-manager delete myapp.local
```

## Development Workflows

### Local Development Setup

```bash
# Set up local development environment
sudo hosts-manager add 127.0.0.1 myapp.local --category development --comment "Main application"
sudo hosts-manager add 127.0.0.1 api.myapp.local --category development --comment "API endpoint"
sudo hosts-manager add 127.0.0.1 admin.myapp.local --category development --comment "Admin panel"

# Quick check
hosts-manager list --category development
```

### Microservices Development

```bash
# Add multiple microservices
sudo hosts-manager add 127.0.0.1 auth.local --category development --comment "Authentication service"
sudo hosts-manager add 127.0.0.1 users.local --category development --comment "User management service"
sudo hosts-manager add 127.0.0.1 orders.local --category development --comment "Order processing service"
sudo hosts-manager add 127.0.0.1 payments.local --category development --comment "Payment service"

# Create a development profile
# Edit config to add custom profile, then:
hosts-manager profile activate microservices-dev
```

### Docker Development

```bash
# Map container services to friendly names
sudo hosts-manager add 172.17.0.2 mysql.docker --category development
sudo hosts-manager add 172.17.0.3 redis.docker --category development
sudo hosts-manager add 172.17.0.4 elasticsearch.docker --category development

# Use with docker-compose
sudo hosts-manager add 192.168.65.2 app.docker --category development --comment "Docker Desktop VM"
```

### Creating Custom Categories

```bash
# Create a custom category for specific projects
sudo hosts-manager category add microservices "Microservices development environment"

# Create categories for different client projects
sudo hosts-manager category add client-alpha "Client Alpha project hosts"
sudo hosts-manager category add client-beta "Client Beta project hosts"

# Create environment-specific categories
sudo hosts-manager category add testing "Integration testing environment"
sudo hosts-manager category add demo "Demo environment for presentations"

# Verify categories were created
hosts-manager category list
```

### Switching Between Environments

```bash
# Disable all development entries at once
sudo hosts-manager category disable development

# Enable staging environment
sudo hosts-manager category enable staging

# Or use profiles for quick switching
sudo hosts-manager profile activate production
sudo hosts-manager profile activate development
```

## Team Collaboration

### Sharing Team Configuration

```bash
# Export team development setup
hosts-manager export --category development --format yaml > team-dev-hosts.yaml

# Team members can import
sudo hosts-manager import team-dev-hosts.yaml
```

Example team configuration file (`team-dev-hosts.yaml`):

```yaml
categories:
- name: development
  description: "Team development environment"
  enabled: true
  entries:
  - ip: "192.168.1.100"
    hostnames: ["api.dev", "auth.dev"]
    comment: "Shared development API"
    category: "development"
    enabled: true
  - ip: "192.168.1.101"
    hostnames: ["db.dev"]
    comment: "Shared development database"
    category: "development"
    enabled: true
```

### Project-Specific Setup

```bash
# Create project-specific hosts
sudo hosts-manager add 10.0.0.100 project-alpha.dev --category custom --comment "Project Alpha"
sudo hosts-manager add 10.0.0.101 project-beta.dev --category custom --comment "Project Beta"

# Export for project repository
hosts-manager export --category custom --format hosts > .devcontainer/hosts
```

### VPN and Remote Access

```bash
# VPN-accessible services
sudo hosts-manager add 10.1.0.100 internal-api.company.com --category vpn --comment "Internal API via VPN"
sudo hosts-manager add 10.1.0.200 jenkins.company.com --category vpn --comment "CI/CD Jenkins"
sudo hosts-manager add 10.1.0.300 monitoring.company.com --category vpn --comment "Monitoring dashboard"

# Enable only when VPN is connected
sudo hosts-manager category enable vpn
# Disable when disconnected
sudo hosts-manager category disable vpn
```

## Production Management

### Blue-Green Deployment

```bash
# Blue environment (current production)
sudo hosts-manager add 10.0.1.100 app.production.com --category blue --comment "Blue environment"
sudo hosts-manager add 10.0.1.101 api.production.com --category blue

# Green environment (new deployment)
sudo hosts-manager add 10.0.2.100 app.production.com --category green --comment "Green environment"
sudo hosts-manager add 10.0.2.101 api.production.com --category green

# Switch to green deployment
sudo hosts-manager category disable blue
sudo hosts-manager category enable green

# Rollback if needed
sudo hosts-manager category disable green
sudo hosts-manager category enable blue
```

### Disaster Recovery Testing

```bash
# Primary data center
sudo hosts-manager add 10.1.0.100 db.production.com --category primary --comment "Primary DB"
sudo hosts-manager add 10.1.0.101 cache.production.com --category primary

# Backup data center
sudo hosts-manager add 10.2.0.100 db.production.com --category backup --comment "Backup DB"
sudo hosts-manager add 10.2.0.101 cache.production.com --category backup

# Switch to backup for DR testing
sudo hosts-manager category disable primary
sudo hosts-manager category enable backup
```

### Load Balancer Testing

```bash
# Test individual backend servers
sudo hosts-manager add 10.0.1.10 app.production.com --category lb-test --comment "Backend server 1"
sudo hosts-manager add 10.0.1.11 app.production.com --category lb-test --comment "Backend server 2"
sudo hosts-manager add 10.0.1.12 app.production.com --category lb-test --comment "Backend server 3"

# Enable for testing specific backend
sudo hosts-manager category enable lb-test
# Restore load balancer
sudo hosts-manager category disable lb-test
```

## Interactive TUI Mode

The Terminal User Interface provides an intuitive way to manage hosts entries:

### Starting TUI Mode
```bash
hosts-manager tui
```

### TUI Features and Workflow

#### Adding New Entries
1. Press `a` to enter add mode
2. Use `Tab` to navigate between fields:
   - IP Address (required)
   - Hostnames (required, space-separated)
   - Comment (optional)
   - Category (defaults to config setting)
3. Press `Enter` to create the entry
4. Press `Esc` to cancel

Example workflow:
```
IP Address: 192.168.1.100
Hostnames: api.dev auth.dev
Comment: Development API services
Category: development
```

#### Managing Existing Entries
- Navigate with `↑/↓` or `j/k`
- Press `Space` to toggle entries enabled/disabled
- Press `d` to delete entries
- Press `m` to move entry to different category
- Press `c` to create new custom category
- Press `/` to search and filter entries
- Press `s` to save changes (shows confirmation)

#### Visual Feedback
- ✓ indicates enabled entries (green)
- ✗ indicates disabled entries (gray)
- Selected entry is highlighted
- Status messages show operation results
- Real-time search filtering

#### Moving Entries Between Categories
1. Navigate to the entry you want to move
2. Press `m` to enter move mode
3. Use `↑/↓` to select target category from the list
4. Press `Enter` to confirm the move
5. Press `Esc` to cancel

Example workflow:
```
Selected Entry: api.dev (192.168.1.100) [development]
Available Categories:
> staging
  production
  custom
  
Press Enter to move to 'staging' category
```

#### Creating New Categories
1. Press `c` to enter create category mode
2. Enter category name (e.g., "testing")
3. Press `Tab` to move to description field
4. Enter description (e.g., "Testing environment hosts")
5. Press `Enter` to create the category
6. Press `Esc` to cancel

Example workflow:
```
Create New Category
Name: testing
Description: Testing environment hosts

Press Enter to create category
```

### TUI Best Practices

```bash
# Recommended workflow for bulk changes
hosts-manager tui
# 1. Search for entries: /development
# 2. Navigate with arrow keys or j/k
# 3. Toggle multiple entries with space
# 4. Edit existing entries with 'e'
# 5. Add new entries with 'a'
# 6. Move entries between categories with 'm'
# 7. Create custom categories with 'c'
# 8. Save all changes with 's'
# 9. Confirm save message appears
```

**TUI Advanced Workflows:**

```bash
# Edit existing entries
hosts-manager tui
# 1. Find the entry you want to edit
# 2. Press 'e' to enter edit mode
# 3. Use Tab/Shift+Tab to navigate between fields:
#    - IP Address
#    - Hostnames (space separated)
#    - Comment (optional)
#    - Category
# 4. Press Enter to save changes
# 5. Press Esc to cancel

# Bulk category management
hosts-manager tui
# 1. Create new categories with 'c'
# 2. Select entries and move them with 'm'
# 3. Use search ('/') to filter entries
# 4. Save all changes with 's'
```

## Advanced Use Cases

### Content Delivery Network (CDN) Testing

```bash
# Test different CDN endpoints
sudo hosts-manager add 1.2.3.4 assets.mysite.com --category cdn-edge1 --comment "CDN Edge Server 1"
sudo hosts-manager add 5.6.7.8 assets.mysite.com --category cdn-edge2 --comment "CDN Edge Server 2"
sudo hosts-manager add 9.10.11.12 assets.mysite.com --category cdn-origin --comment "Origin Server"

# Test different CDN endpoints
sudo hosts-manager category enable cdn-edge1
# curl -I https://assets.mysite.com/test.js
sudo hosts-manager category disable cdn-edge1
sudo hosts-manager category enable cdn-origin
```

### Security Testing

```bash
# Block malicious domains (for testing)
sudo hosts-manager add 0.0.0.0 malicious-site.com --category blocked --comment "Blocked for security"
sudo hosts-manager add 0.0.0.0 tracking-service.com --category blocked --comment "Block tracking"

# Redirect to honeypot
sudo hosts-manager add 192.168.1.250 suspicious-domain.com --category honeypot --comment "Honeypot redirect"
```

### Debugging and Troubleshooting

```bash
# Capture specific service traffic
sudo hosts-manager add 127.0.0.1 api.external.com --category debug --comment "Route to local proxy"

# Set up local proxy on 127.0.0.1:8080 to capture traffic
# Enable debugging
sudo hosts-manager category enable debug
# Run tests
# Disable when done
sudo hosts-manager category disable debug
```

### A/B Testing

```bash
# Version A
sudo hosts-manager add 10.0.1.100 feature.myapp.com --category version-a --comment "Feature Version A"

# Version B
sudo hosts-manager add 10.0.1.200 feature.myapp.com --category version-b --comment "Feature Version B"

# Test Version A
sudo hosts-manager category enable version-a
sudo hosts-manager category disable version-b

# Switch to Version B
sudo hosts-manager category disable version-a
sudo hosts-manager category enable version-b
```

## Automation and Scripting

### Backup Before Major Changes

```bash
#!/bin/bash
# backup-and-update.sh

echo "Creating backup..."
sudo hosts-manager backup

echo "Importing new configuration..."
sudo hosts-manager import new-config.yaml

echo "Verifying changes..."
hosts-manager list

echo "Backup created and configuration updated!"
```

### Environment Switching Script

```bash
#!/bin/bash
# switch-env.sh

ENVIRONMENT=$1

if [ -z "$ENVIRONMENT" ]; then
    echo "Usage: $0 <development|staging|production>"
    exit 1
fi

echo "Switching to $ENVIRONMENT environment..."

# Disable all environments
sudo hosts-manager category disable development
sudo hosts-manager category disable staging
sudo hosts-manager category disable production

# Enable requested environment
sudo hosts-manager category enable "$ENVIRONMENT"

echo "Switched to $ENVIRONMENT environment"
hosts-manager list --category "$ENVIRONMENT"
```

### Automated Testing Integration

```bash
#!/bin/bash
# test-with-hosts.sh

# Save current state
hosts-manager export --format yaml > hosts-backup.yaml

# Apply test configuration
sudo hosts-manager import test-hosts.yaml

# Run tests
npm test

# Restore original configuration
sudo hosts-manager import hosts-backup.yaml

# Clean up
rm hosts-backup.yaml
```

### Docker Integration

```dockerfile
# Dockerfile.dev
FROM node:16

# Install hosts-manager
RUN curl -L https://github.com/your-username/hosts-manager/releases/latest/download/hosts-manager-linux-amd64.tar.gz | tar xz -C /usr/local/bin/

# Copy hosts configuration
COPY dev-hosts.yaml /app/

# Apply configuration in container
RUN hosts-manager import /app/dev-hosts.yaml

WORKDIR /app
COPY . .
CMD ["npm", "start"]
```

### CI/CD Pipeline Integration

```yaml
# .github/workflows/test.yml
name: Test with Custom Hosts

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2

    - name: Install hosts-manager
      run: |
        curl -L https://github.com/your-username/hosts-manager/releases/latest/download/hosts-manager-linux-amd64.tar.gz | tar xz
        sudo mv hosts-manager /usr/local/bin/

    - name: Setup test hosts
      run: |
        sudo hosts-manager add 127.0.0.1 api.test --category test
        sudo hosts-manager add 127.0.0.1 db.test --category test

    - name: Run tests
      run: npm test

    - name: Cleanup
      run: sudo hosts-manager category disable test
```

### Monitoring and Alerting

```bash
#!/bin/bash
# monitor-hosts.sh

# Check if critical hosts are reachable
CRITICAL_HOSTS=("api.production.com" "db.production.com" "cdn.production.com")

for host in "${CRITICAL_HOSTS[@]}"; do
    if ! ping -c 1 "$host" &> /dev/null; then
        echo "ALERT: $host is not reachable!"
        # Send alert to monitoring system
        curl -X POST "https://alerts.example.com/webhook" \
             -H "Content-Type: application/json" \
             -d "{\"message\": \"Host $host is not reachable\"}"
    fi
done
```

## Best Practices

### Regular Maintenance

```bash
# Weekly maintenance script
#!/bin/bash

echo "Performing weekly hosts maintenance..."

# Create backup
sudo hosts-manager backup

# Clean up old backups (keeps last 10)
# hosts-manager automatically manages this based on config

# Verify all entries are valid
hosts-manager list --show-disabled > /tmp/hosts-check.txt

# Look for duplicates or conflicts
hosts-manager search "" | sort | uniq -d

echo "Maintenance complete!"
```

### Configuration Management

```bash
# Store configuration in version control
git add ~/.config/hosts-manager/config.yaml

# Export current hosts for backup
hosts-manager export --format yaml > hosts-$(date +%Y%m%d).yaml

# Document changes
echo "$(date): Updated staging environment IPs" >> hosts-changelog.txt
```

These examples demonstrate the flexibility and power of the Hosts Manager for various workflows and environments. Adapt them to your specific needs and always test changes in a safe environment first.