

---

**Source**: KNIRVROOT/docs/Troubleshooting/completedFixes/peernode_fix_implementation_plan.md

# PeerNode Configuration Fix Implementation Plan

## Issue Summary

The KNIRVCHAIN dev node is crashing on startup after installation due to missing dev port configurations. The installer claims to save the configuration to `dev_config.json`, but upon restart, the application cannot find this file and loads default settings instead, which lack the necessary port information.

## Root Cause Analysis

After analyzing the code, I've identified the following issues:

1. **Filename Mismatch**: There's a discrepancy between the filename used when saving the configuration and the filename expected when loading it:
   - During installation, the configuration is saved to a file named with uppercase first letter: `Peer_config.json` (line 171 in `config/minimal_config.go`)
   - However, when loading the configuration, Viper looks for a lowercase filename: `dev_config.json` (line 77 in `config/viper_loader.go`)

2. **Inconsistent Logging**: The installer logs that it's saving to `dev_config.json`, but it's actually saving to `Peer_config.json`, creating confusion.

3. **Error Handling**: The `SaveMinimalConfigToUserDir` function logs warnings but doesn't return errors, making it difficult to detect configuration save failures.

4. **InstallComplete Flag**: The `InstallComplete` flag is critical for preventing unnecessary reinstallation on subsequent starts. It needs to be consistently set and saved for all non-Root roles.

## Implementation Plan

### 1. Ensure InstallComplete Flag is Set for All Roles

**File**: `install.go`

**Change**: Ensure the InstallComplete flag is explicitly set to true for all roles (except Root) before saving the configuration:

```go
// Add before line 495 (before saving the configuration)
// Ensure InstallComplete flag is set to true for all non-Root roles
if role != config.Root {
    log.Printf("Setting InstallComplete=true for %s role", role.String())
    configToSave.InstallComplete = true
}
```

**File**: `config/config.go`

**Change**: Ensure the InstallComplete field is properly defined in the Config struct and included in the minimal config:

```go
// Verify in the Config struct definition that InstallComplete is present:
type Config struct {
    // ... other fields
    InstallComplete bool `mapstructure:"InstallComplete" json:"InstallComplete"`
    // ... other fields
}

// Verify in the ToMinimalConfig function that InstallComplete is included:
func ToMinimalConfig(cfg *Config, role Role) *MinimalConfig {
    return &MinimalConfig{
        // ... other fields
        InstallComplete: cfg.InstallComplete,
        // ... other fields
    }
}
```

### 2. Fix the Filename Case Inconsistency

**File**: `config/minimal_config.go`

**Change**: Modify line 171 to ensure consistent lowercase naming for configuration files:

```go
// Current:
roleSpecificFilename := fmt.Sprintf("%s_config.json", filepath.Base(role.String()))

// Change to:
roleSpecificFilename := fmt.Sprintf("%s_config.json", strings.ToLower(role.String()))
```

This ensures that the saved filename (`dev_config.json`) matches what Viper looks for when loading.

### 2. Add Error Propagation in SaveConfigToUserDir

**File**: `install.go`

**Change**: Modify the call to `SaveConfigToUserDir` around line 501 to check for errors:

```go
// Current:
log.Printf("Installer: Saving configuration for %s role", role.String())
config.SaveConfigToUserDir(configToSave, role)

// Change to:
log.Printf("Installer: Saving configuration for %s role", role.String())
if err := config.SaveConfigToUserDir(configToSave, role); err != nil {
    log.Printf("ERROR: Failed to save configuration for %s role: %v", role.String(), err)
    return configToSave, fmt.Errorf("failed to save configuration: %w", err)
}
```

### 3. Update SaveConfigToUserDir to Return Errors

**File**: `config/minimal_config.go`

**Change**: Modify the `SaveConfigToUserDir` function to return errors:

```go
// Current:
func SaveConfigToUserDir(cfg *Config, role Role) {
    // ...
}

// Change to:
func SaveConfigToUserDir(cfg *Config, role Role) error {
    // Skip saving for Root role for security reasons
    if role == Root {
        log.Printf("SECURITY: Skipping config file save for Root role to prevent sensitive data exposure")
        return nil
    }

    roleSpecificFilename := fmt.Sprintf("%s_config.json", strings.ToLower(role.String()))

    // Get the role-specific data directory
    roleDataDir, err := GetDataDir(role)
    if err != nil {
        return fmt.Errorf("could not get role-specific data directory: %w", err)
    }

    // First try to save in the role-specific data directory
    targetSavePath := filepath.Join(roleDataDir, roleSpecificFilename)
    _, err = SaveMinimalConfig(targetSavePath, cfg, role)
    if err != nil {
        return fmt.Errorf("failed to save minimal config to role-specific dir %s: %w", targetSavePath, err)
    }
    
    return nil
}
```

### 4. Fix the Log Message in install.go

**File**: `install.go`

**Change**: Update the log message around line 510 to correctly reflect the actual filename being used:

```go
// Current:
roleSpecificFilename := fmt.Sprintf("%s_config.json", strings.ToLower(role.String()))
configPath := filepath.Join(roleDataDir, roleSpecificFilename)
fmt.Printf("Configuration updated and saved to %s\n", configPath)

// Change to:
roleSpecificFilename := fmt.Sprintf("%s_config.json", strings.ToLower(role.String()))
configPath := filepath.Join(roleDataDir, roleSpecificFilename)
fmt.Printf("Configuration updated and saved to %s\n", configPath)
```

### 5. Add Verification Step After Configuration Save

**File**: `install.go`

**Change**: Add a verification step after saving the configuration to ensure it was written correctly:

```go
// Add after line 511 (after the configuration save message)
// Verify that the configuration file exists and contains the expected data
if _, err := os.Stat(configPath); os.IsNotExist(err) {
    log.Printf("WARNING: Configuration file %s was not found after save attempt", configPath)
} else {
    log.Printf("Verified: Configuration file %s exists", configPath)
}
```

### 6. Ensure Port Settings are Properly Saved

**File**: `install.go`

**Change**: Ensure the dev port settings are explicitly set in the configuration before saving:

```go
// Add before line 495 (before saving the configuration)
// Ensure dev port settings are explicitly set in the configuration
if role == config.RolePeer {
    log.Printf("Setting dev ports in configuration: HTTP=%d, P2P=%d", devHTTPPort, devP2PPort)
    configToSave.Port = int(devHTTPPort)
    configToSave.P2PPort = int(devP2PPort)
}
```

## Testing Plan

1. **Manual Testing for Peer Node**:
   - Run the installer for a dev node
   - Verify that `dev_config.json` (lowercase) is created in the correct directory
   - Verify that the file contains the correct port settings and `InstallComplete: true`
   - Restart the application and confirm it loads the configuration correctly
   - Verify the node doesn't attempt to reinstall

2. **Testing for Other Roles**:
   - Run the installer for bootnode and client roles
   - Verify that the respective config files are created with lowercase names
   - Verify that `InstallComplete: true` is set in each config file
   - Restart each role and confirm they don't attempt to reinstall

3. **Error Handling Testing**:
   - Simulate a write failure (e.g., by making the directory read-only)
   - Verify that appropriate error messages are displayed
   - Verify that the application doesn't proceed as if the save was successful

## Expected Outcome

After implementing these changes:

1. The installer will save the configuration to `dev_config.json` (lowercase)
2. The configuration file will contain the correct dev port settings
3. The `InstallComplete` flag will be properly set and saved for all non-Root roles
4. Upon restart, Viper will find and load the configuration file
5. The dev node will start successfully with the configured ports
6. Nodes will correctly recognize they've been installed and won't trigger unnecessary reinstallation

This fix addresses the core issues by ensuring:
- Consistent filename casing between saving and loading configurations
- Proper setting and saving of the InstallComplete flag for all roles
- Improved error handling and verification steps to detect configuration save failures
- Explicit setting of port configurations before saving

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
