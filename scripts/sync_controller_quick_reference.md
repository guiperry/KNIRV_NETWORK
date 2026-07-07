# KNIRVCONTROLLER Sync - Quick Reference Guide

## 🚀 Quick Start

### Basic Sync (Safe - with backup)
```bash
./scripts/sync_controller_to_public.sh
```

### Test First (Dry Run)
```bash
./scripts/sync_controller_to_public.sh -d
```

### Quick Sync (No progress bars)
```bash
./scripts/sync_controller_to_public.sh -q
```

## 🔧 Common Operations

### Check for Changes
```bash
# See what would be copied
./scripts/sync_controller_to_public.sh -d
```

### Force Sync (Skip backup checks)
```bash
./scripts/sync_controller_to_public.sh -f
```

### Create Backup Only
```bash
./scripts/sync_controller_to_public.sh -b
```

### Restore from Backup
```bash
./scripts/sync_controller_to_public.sh -r
```

### Verbose Mode (Detailed output)
```bash
./scripts/sync_controller_to_public.sh -v
```

## 📊 Monitoring

### Check Logs
```bash
tail -f sync_controller.log
```

### View Recent Backups
```bash
ls -la sync_backups/
```

### Check Backup Contents
```bash
# Replace with actual backup name
ls -la sync_backups/backup_20250101_120000/
```

## ⚙️ Automation

### Cron Job (Hourly sync)
```bash
# Add to crontab -e
0 * * * * cd /home/gperry/Documents/GitHub/cloud-equities/KNIRV_NETWORK && ./scripts/sync_controller_to_public.sh -q >> sync_controller_cron.log 2>&1
```

### Git Hook (Post-commit sync)
```bash
# Add to .git/hooks/post-commit
#!/bin/bash
cd /home/gperry/Documents/GitHub/cloud-equities/KNIRV_NETWORK
./scripts/sync_controller_to_public.sh -q
```

## 🛠️ Troubleshooting

### Permission Issues
```bash
chmod +x scripts/sync_controller_to_public.sh
```

### Directory Not Found
```bash
# Check if directories exist
ls -la KNIRVCONTROLLER/
ls -la ../KNIRVCONTROLLER_public/
```

### Restore Failed
```bash
# Check available backups
ls -la sync_backups/

# Manual restore from specific backup
cp -r sync_backups/backup_20250101_120000/* ../KNIRVCONTROLLER_public/
```

### Script Hangs
```bash
# Kill the script
pkill -f "sync_controller_to_public.sh"

# Check for running processes
ps aux | grep sync_controller
```

## 📈 Performance Tips

### For Large Syncs
```bash
# Use quiet mode for better performance
./scripts/sync_controller_to_public.sh -q

# Or run in background
nohup ./scripts/sync_controller_to_public.sh -q > sync_output.log 2>&1 &
```

### Monitor Progress
```bash
# Watch the log file
tail -f sync_controller.log

# Check disk space
df -h .
```

## 🔄 Idempotency Features

The script is designed to be safe for repeated runs:

- **Safe**: Creates backups before any changes
- **Idempotent**: Only copies changed files (SHA256 comparison)
- **Restorable**: Easy rollback from any backup
- **Logging**: Detailed operation tracking

### Verify Idempotency
```bash
# First run
./scripts/sync_controller_to_public.sh -d

# Second run (should show no changes)
./scripts/sync_controller_to_public.sh -d
```

## 🎯 Key Features Summary

| Feature | Command | Description |
|---------|---------|-------------|
| **Safe Sync** | `./script.sh` | Default operation with backups |
| **Dry Run** | `./script.sh -d` | Test without making changes |
| **Quiet Mode** | `./script.sh -q` | No progress indicators |
| **Force Sync** | `./script.sh -f` | Skip backup checks |
| **Backup Only** | `./script.sh -b` | Create backup without sync |
| **Restore** | `./script.sh -r` | Rollback from latest backup |
| **Verbose** | `./script.sh -v` | Detailed file information |

## 📞 Emergency Commands

### Stop Script Immediately
```bash
pkill -f "sync_controller_to_public.sh"
```

### Manual Restore (Emergency)
```bash
# From latest backup
latest=$(ls -dt sync_backups/backup_* | head -1)
cp -r "$latest"/* ../KNIRVCONTROLLER_public/
```

### Check Script Health
```bash
# Verify script is executable
ls -la scripts/sync_controller_to_public.sh

# Test basic functionality
./scripts/sync_controller_to_public.sh -d -q
```

## 📋 Best Practices Checklist

- [ ] Always test with `-d` flag first
- [ ] Monitor `sync_controller.log` for errors
- [ ] Use `-q` flag for automated runs
- [ ] Keep backups organized in `sync_backups/`
- [ ] Verify sync results with second dry run
- [ ] Document any custom configurations

---

**Need Help?** Check the full documentation: `scripts/sync_controller_to_public_README.md`