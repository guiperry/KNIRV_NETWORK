# KNIRVCONTROLLER to KNIRVCONTROLLER_public Sync Script

A safe, idempotent script for synchronizing files between the KNIRVCONTROLLER directory and the public KNIRVCONTROLLER_public directory.

## Overview

This script provides a robust solution for copying changed files from `KNIRVCONTROLLER` to `KNIRVCONTROLLER_public` with the following safety features:

- **Safe**: Creates backups before making any changes
- **Idempotent**: Can be run multiple times without causing issues
- **Selective**: Only copies files that have actually changed
- **Restorable**: Easy rollback capability from any backup

## Features

### 🔒 Safety Features
- Automatic backup creation before file operations
- Checksum-based file comparison (SHA256)
- Dry-run mode for testing
- Comprehensive error handling and logging
- Signal trapping for clean exits

### 🔄 Idempotency
- Only copies files that differ between source and target
- Safe to run multiple times - identical files are skipped
- No duplicate operations or file corruption

### 📊 Change Detection
- Compares file contents using SHA256 checksums
- Detects new files, modified files, and unchanged files
- Verbose mode shows detailed change information

### 📈 Progress Tracking
- Real-time progress bars for long operations
- File count and percentage completion
- Current file being processed displayed
- Quiet mode available for automated runs

### � Restore Capability
- Easy restoration from any backup
- Backup metadata tracking
- Selective file restoration

## Usage

### Basic Sync
```bash
./scripts/sync_controller_to_public.sh
```

### Dry Run (Test Mode)
```bash
./scripts/sync_controller_to_public.sh -d
```

### Verbose Mode
```bash
./scripts/sync_controller_to_public.sh -v
```

### Restore from Backup
```bash
./scripts/sync_controller_to_public.sh -r
```

### Create Backup Only
```bash
./scripts/sync_controller_to_public.sh -b
```

### Force Sync (Skip Backup Checks)
```bash
./scripts/sync_controller_to_public.sh -f
```

### Quiet Mode (No Progress Bars)
```bash
./scripts/sync_controller_to_public.sh -q
```

### Help
```bash
./scripts/sync_controller_to_public.sh -h
```

## Command Line Options

| Option | Short | Description |
|--------|-------|-------------|
| `--dry-run` | `-d` | Show what would be copied without making changes |
| `--force` | `-f` | Force sync even if backups exist |
| `--verbose` | `-v` | Enable verbose output with detailed file info |
| `--restore` | `-r` | Restore from the latest backup |
| `--backup` | `-b` | Create backup only (no file copying) |
| `--quiet` | `-q` | Disable progress indicators |
| `--help` | `-h` | Show help message |

## File Structure

### Source Directory
```
KNIRVCONTROLLER/
├── src/
├── config/
├── scripts/
├── public/
└── ... (all project files)
```

### Target Directory
```
../KNIRVCONTROLLER_public/
├── src/
├── config/
├── scripts/
├── public/
└── ... (public version of files)
```

### Backup Directory
```
sync_backups/
├── backup_20250101_120000/
│   ├── changed_files.txt
│   ├── backup_metadata.txt
│   └── ... (backed up files)
└── backup_20250101_130000/
    └── ... (another backup)
```

## How It Works

### 1. Change Detection
- The script scans all files in the source directory
- For each file, it calculates a SHA256 checksum
- Compares with the target file's checksum
- Only files with different checksums are marked for copying

### 2. Backup Creation
- Before any file operations, a timestamped backup is created
- Only changed files are backed up (efficient storage)
- Backup metadata is stored for restoration

### 3. File Copying
- Files are copied using standard `cp` command
- Directory structure is preserved
- Permissions are maintained

### 4. Progress Tracking
- Real-time progress bars show operation status
- File counts and percentages displayed
- Current file being processed shown
- Progress bars automatically clear for log messages

### 5. Logging
- All operations are logged to `sync_controller.log`
- Color-coded output for different log levels
- Timestamped entries for audit purposes

## Safety Considerations

### Backup Strategy
- Backups are created in `./sync_backups/` directory
- Each backup is timestamped for easy identification
- Only changed files are backed up (space efficient)
- Backup metadata ensures reliable restoration

### Error Handling
- Script exits on any critical error
- File operations are atomic where possible
- Cleanup is performed on script interruption
- Comprehensive error messages with suggestions

### Idempotency Guarantees
- File comparison prevents unnecessary copying
- Multiple runs produce identical results
- No side effects from repeated execution

## Examples

### Example 1: First-time Sync
```bash
# Check what will be copied
./scripts/sync_controller_to_public.sh -d -v

# Perform the actual sync
./scripts/sync_controller_to_public.sh -v
```

### Example 2: Regular Maintenance
```bash
# Quick sync (only changed files)
./scripts/sync_controller_to_public.sh

# Check log for changes
tail -f sync_controller.log
```

### Example 3: Recovery Scenario
```bash
# Something went wrong - restore from backup
./scripts/sync_controller_to_public.sh -r

# Verify restoration worked
./scripts/sync_controller_to_public.sh -d
```

### Example 4: Automated Sync (Quiet Mode)
```bash
# For cron jobs or automated scripts
./scripts/sync_controller_to_public.sh -q
```

## Integration

### Cron Job for Automated Sync
```bash
# Add to crontab for hourly sync (quiet mode recommended)
0 * * * * cd /home/gperry/Documents/GitHub/cloud-equities/KNIRV_NETWORK && ./scripts/sync_controller_to_public.sh -q >> sync_controller_cron.log 2>&1
```

### Git Hooks
```bash
# Add to post-commit hook for automatic sync after commits
#!/bin/bash
cd /home/gperry/Documents/GitHub/cloud-equities/KNIRV_NETWORK
./scripts/sync_controller_to_public.sh -q
```

## Troubleshooting

### Common Issues

**Permission Denied**
```bash
chmod +x scripts/sync_controller_to_public.sh
```

**Directory Not Found**
- Ensure you're running from the correct directory
- Check that both source and target directories exist

**Backup Restoration Fails**
- Verify backup directory exists and has valid backups
- Check backup metadata files for completeness

### Log Analysis
The script creates detailed logs in `sync_controller.log`. Key sections to check:

- `[INFO]` - Normal operation messages
- `[WARN]` - Non-critical issues
- `[ERROR]` - Critical failures requiring attention
- `[DEBUG]` - Detailed file information (verbose mode only)

### Performance Considerations
- Large files may take longer due to checksum calculation
- Use `-v` flag sparingly in production for better performance
- Regular syncs are faster than infrequent large syncs
- Progress indicators add minimal overhead but improve user experience

## Best Practices

1. **Always test with dry-run first**: Use `-d` flag before actual sync
2. **Monitor logs regularly**: Check `sync_controller.log` for issues
3. **Use quiet mode for automation**: Use `-q` flag for cron jobs and scripts
4. **Regular backups**: The script maintains its own, but consider additional backups
5. **Version control**: Keep the script under version control
6. **Document changes**: Update this README when modifying the script

## Development

### Adding New Features
- Follow the existing code structure
- Maintain safety and idempotency guarantees
- Update documentation accordingly
- Test thoroughly with various scenarios

### Testing
```bash
# Test basic functionality
./scripts/sync_controller_to_public.sh -d

# Test verbose mode
./scripts/sync_controller_to_public.sh -d -v

# Test restore functionality
./scripts/sync_controller_to_public.sh -r

# Test error conditions (missing directories, etc.)
```

## License

This script is part of the KNIRV Network project. Use according to project licensing terms.

## Support

For issues or questions:
1. Check the log file for detailed error information
2. Review this documentation
3. Consult the project maintainers