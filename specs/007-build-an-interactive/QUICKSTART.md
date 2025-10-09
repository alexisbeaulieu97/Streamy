# Dashboard Quick Start Guide

Get started with the Streamy interactive dashboard in 5 minutes!

## Installation

```bash
# Build Streamy
go build -o streamy ./cmd/streamy

# Or use the install script
./scripts/install.sh
```

## Quick Start (5 Steps)

### 1. Register Your First Pipeline

```bash
# Create a simple config file
cat > my-env.yaml << 'EOF'
version: "1.0"
name: "My Dev Environment"
description: "Basic development environment setup"
steps:
  - id: create_dir
    type: command
    command: "mkdir -p ~/.myenv"
  - id: create_file
    type: command
    command: "echo 'Hello from Streamy!' > ~/.myenv/README"
    depends_on:
      - create_dir
EOF

# Register the pipeline
./streamy register my-env my-env.yaml --description "My first pipeline"
```

### 2. Launch the Dashboard

```bash
./streamy dashboard
```

You should see:
```
🚀 Streamy Dashboard

🟢 0  🟡 0  🔴 0  ⚪ 1

1. ⚪  my-env
   My first pipeline
   Last checked: Never

↑/↓: navigate  •  enter: select  •  r: refresh  •  ?: help  •  q: quit
```

### 3. Verify Your Pipeline

1. Press `Enter` to view pipeline details
2. Press `v` to verify the pipeline
3. Watch the progress indicator
4. See the status update to 🟢 (satisfied) or 🟡 (drifted)

### 4. Explore the Interface

Try these keyboard shortcuts:
- `?` - Show comprehensive help
- `Esc` - Go back to list
- `r` - Refresh all pipelines
- `q` - Quit dashboard

### 5. Make a Change and Apply

```bash
# Modify the config to add another step
cat >> my-env.yaml << 'EOF'
  - id: another_file
    type: command
    command: "echo 'Another file' > ~/.myenv/file2"
EOF
```

Back in the dashboard:
1. Select the pipeline (press `1` or navigate and press `Enter`)
2. Press `v` to verify - status will show 🟡 (drifted)
3. Press `a` to apply changes
4. Confirm with `y`
5. Watch the apply progress
6. Status auto-verifies and updates to 🟢

## Common Workflows

### Daily Monitoring
```bash
# Launch dashboard
./streamy dashboard

# Quick status check - all pipelines shown with color codes
# Press 'r' to refresh all statuses
```

### Investigating Drift
```bash
# In dashboard:
1. Navigate to yellow (drifted) pipeline
2. Press Enter to see details
3. Review "Last Execution" section for what changed
4. Press 'a' to apply fixes
5. Confirm with 'y'
```

### Managing Multiple Environments
```bash
# Register multiple pipelines
./streamy register dev ./configs/dev.yaml
./streamy register staging ./configs/staging.yaml  
./streamy register prod ./configs/prod.yaml

# Launch dashboard - see all environments at once
./streamy dashboard

# Use number keys 1-9 to jump between pipelines
```

## Status Indicators Guide

| Icon | Status | Meaning |
|------|--------|---------|
| 🟢 | Satisfied | System matches configuration perfectly |
| 🟡 | Drifted | System differs from configuration |
| 🔴 | Failed | Verification or apply operation failed |
| ⚪ | Unknown | Not yet verified |
| ⚙️ | In Progress | Operation currently running |

## Tips & Tricks

### Fast Navigation
- Use number keys `1`-`9` to jump directly to pipelines
- Arrow keys + `Enter` for careful selection
- `Esc` always goes back one level

### Efficient Workflows
1. **Morning Check**: Launch dashboard, press `r` to refresh all
2. **Fix Drift**: Navigate to 🟡 items, press `a` to apply
3. **Verify Changes**: Status auto-updates after apply

### Keyboard Shortcuts
**Remember these 4 keys**:
- `?` - When in doubt, show help
- `Esc` - Go back / cancel
- `r` - Refresh status
- `q` - Quit

### Error Handling
- Red banner appears for errors
- Read the error message and suggestion
- Press `x` to dismiss error banner
- Try operation again or fix config

## Example: Complete Setup

```bash
# 1. Create workspace directory
mkdir ~/streamy-configs
cd ~/streamy-configs

# 2. Create development environment config
cat > dev.yaml << 'EOF'
version: "1.0"
name: "Development Environment"
description: "Local dev tools and configs"
steps:
  - id: install_git
    type: package
    package: git
    manager: apt
  - id: create_workspace
    type: command
    command: "mkdir -p ~/workspace"
  - id: setup_gitconfig
    type: command
    command: "git config --global user.name 'Developer'"
    depends_on:
      - install_git
EOF

# 3. Create production monitoring config
cat > prod-monitor.yaml << 'EOF'
version: "1.0"
name: "Production Monitoring"
description: "Ensure monitoring agents are running"
steps:
  - id: check_agent
    type: command
    command: "systemctl is-active monitoring-agent"
EOF

# 4. Register both pipelines
./streamy register dev dev.yaml
./streamy register prod prod-monitor.yaml

# 5. Launch dashboard
./streamy dashboard

# 6. In dashboard:
#    - Press 'r' to verify all pipelines
#    - Navigate to any drifted items
#    - Press 'a' to apply fixes
#    - Enjoy the unified view!
```

## Troubleshooting

### "No pipelines registered"
```bash
# Register at least one pipeline first
./streamy register my-first ./config.yaml
```

### "Terminal too small"
- Dashboard requires minimum 80x24 terminal
- Resize your terminal window or use fullscreen

### Pipeline shows ⚪ (unknown) forever
- Press `v` to manually trigger verification
- Check that config file still exists at registered path

### Config file changed but dashboard doesn't reflect it
- Restart dashboard to reload configs
- Or unregister and re-register the pipeline

### Operation seems stuck
- Press `Esc` to cancel operation (confirmation required)
- Check system resources if operations are slow

## Next Steps

1. **Explore Help**: Press `?` in dashboard for detailed shortcuts
2. **Add More Pipelines**: Register your actual environment configs
3. **Automate**: Add `streamy dashboard` to your startup scripts
4. **Learn More**: Read [README.md](../../README.md) for full feature list

## Support

- **Documentation**: [README.md](../../README.md)
- **Architecture**: [docs/architecture.md](../../docs/architecture.md)
- **Schema Reference**: [docs/schema.md](../../docs/schema.md)
- **Plugin Guide**: [docs/plugins.md](../../docs/plugins.md)

---

**Pro Tip**: The dashboard status cache means second launches are instant (<500ms). Your pipeline statuses are remembered between sessions!

Happy configuring! 🚀
