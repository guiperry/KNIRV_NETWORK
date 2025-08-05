

---

**Source**: KNIRVROOT/docs/Troubleshooting/completedFixes/peernode_fix.md

Okay, I've analyzed the devnode_log.md file. It seems like the KNIRVCHAIN node is crashing on startup after the installation process due to missing dev port configurations. Let's break down what's happening and how to address it.

Analysis of the Log:

Initial Start & Forced Installation:

The application starts, identifies itself as a "Peer" node.
It doesn't find dev_config.json, so it uses defaults.
Crucially, it logs: WARNING: No wallet file found and MinersAddress not configured. and then This node requires reinstallation. Forcing installer.
Installation Process:

The user selects "Peer" role.
The installer prompts for and receives:
KNIRVCHAIN Root endpoint (default used).
Desired URI (devs).
Peer HTTP Port (default 1000 used).
Peer P2P (Libp2p) Port (1001 entered by user).
A wallet is generated.
The installer logs: Configuration updated and saved to /home/gperry/.config/KNIRVCHAIN/dev_data/dev_config.json. This is supposed to be the file containing the port settings and InstallComplete: true.
The application then attempts to restart with [-dev --skip-install].
Restart and Crash:

The application restarts.
It again logs: Viper: Role-specific config file 'dev_config.json' not found. Using defaults. This is the core problem. If the installer saved dev_config.json, why is it not found now?
Because dev_config.json is supposedly not found, the application loads default settings, which presumably do not include the specific dev ports configured during installation.
The --skip-install flag prevents the installer from running again.
The application then proceeds with initialization but hits a fatal error: FATAL: Peer ports not set in config.json. Run installation or set manually. This happens because the loaded configuration (the defaults) doesn't have the necessary port information.
Root Cause Hypothesis:

The primary issue seems to be that the dev_config.json file, which the installer claims to save with the necessary port configurations, is either:

Not being saved correctly by the installer (e.g., an error during the write, or it's written empty/partially).
Being saved correctly but is then inaccessible or considered "not found" by Viper immediately upon restart.
The log message Viper: Role-specific config file 'dev_config.json' not found. Using defaults. on the second run is the key indicator. If this file was correctly written by the installer and was readable, Viper should have loaded it, and the port information would be available.

Proposed Fix Strategy:

The fix needs to ensure that the dev port settings (and other essential configurations like InstallComplete: true) gathered during the installation are reliably written to dev_config.json and that this file is correctly read upon restart.

You'll need to look into the Go code responsible for the installer, specifically the part that saves the configuration for a "Peer" node.

Code Areas to Inspect and Potentially Modify:

Installer's Configuration Saving Logic:

Locate the function where the installer prompts for Peer HTTP Port and Peer P2P (Libp2p) Port.
Verify that these collected port values are correctly assigned to the configuration struct/map that is intended to be saved.
Examine the code that writes this configuration to dev_config.json. This might involve viper.WriteConfig(), viper.WriteConfigAs(), or a manual JSON marshal and file write.
Ensure the correct file path is used: /home/gperry/.config/KNIRVCHAIN/dev_data/dev_config.json.
Ensure the data being written actually includes the fields for HTTP and P2P ports with the values provided by the user.
Crucially, check for error handling after the file write operation. The log "Configuration updated and saved..." might be optimistic. If an error occurs during the write, it must be handled appropriately, and the application should not proceed as if the save was successful.
Example (conceptual - actual Viper usage might vary): Let's say you have a configuration struct:

go
type PeerNodeConfig struct {
    HTTPPort        int    `mapstructure:"HTTPPort"`
    P2PPort         int    `mapstructure:"P2PPort"`
    InstallComplete bool   `mapstructure:"InstallComplete"`
    ChainID         string `mapstructure:"ChainID"`
    // ... other fields
}
When saving in the installer:

go
// Assume 'devSettings' is an instance of PeerNodeConfig or similar map/struct
// populated with user input and other installer-determined values.
// e.g., devSettings.HTTPPort = 1000
//       devSettings.P2PPort = 1001
//       devSettings.InstallComplete = true
//       devSettings.ChainID = "devs"

// If using Viper to write:
// Make sure Viper's internal state reflects these settings before writing.
// For example, by unmarshalling devSettings into Viper or setting values directly:
v := viper.New() // Or use an existing relevant Viper instance
v.Set("HTTPPort", devSettings.HTTPPort)
v.Set("P2PPort", devSettings.P2PPort)
v.Set("InstallComplete", devSettings.InstallComplete)
v.Set("ChainID", devSettings.ChainID)
// ... set other necessary config values

configFilePath := filepath.Join(roleSpecificDataDir, "dev_config.json")
v.SetConfigFile(configFilePath) // Important: tell Viper where to save

// Use WriteConfig or SafeWriteConfig. WriteConfigAs might also be used if you're managing the path differently.
if err := v.WriteConfig(); err != nil {
    // Log the actual error here! This is critical.
    log.Fatalf("FATAL: Failed to write dev configuration to %s: %v", configFilePath, err)
    // Or handle it in a way that prevents the installer from claiming success.
} else {
    log.Printf("Configuration successfully saved to %s", configFilePath)
}
Configuration Loading on Restart:

While the primary issue seems to be the saving part, double-check that the loading logic (which logs Viper: Role-specific config file 'dev_config.json' not found) is correctly specifying the path and filename for Viper to read. The logs indicate the path is correct, so this is less likely the issue if the file was indeed saved properly.
Debugging Steps:

Add more detailed logging around the file saving operation in the installer. Log the actual data being written and any errors encountered.
After the installer claims to have saved dev_config.json and before it restarts, manually inspect the file system to see if /home/gperry/.config/KNIRVCHAIN/dev_data/dev_config.json exists and what its contents are. This will tell you if the file is being written at all and if it contains the port numbers.
If the file exists and has the correct content, then the "not found" message from Viper on restart becomes more mysterious and might point to a more subtle issue (permissions, very rapid restart timing with filesystem caches, or a bug in Viper's file detection in this specific context). However, a failure to write correctly is more common.
By ensuring the installer robustly saves the complete configuration, including the user-defined ports, to dev_config.json, the subsequent restart should allow Viper to find and load this configuration, resolving the fatal error.



---

<div class="footer-links">


© 2025 KNIRV Network
</div>
