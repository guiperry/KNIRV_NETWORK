

---

**Source**: KNIRVROOT/docs/Troubleshooting/completedFixes/Init_Refactor_Plan.md

```markdown
# Implementation Plan: KNIRVCHAIN Role-Based Initialization and Config Refactor

**Objective:** To refactor the node initialization process to correctly handle configuration files based on node role (Root, Bootnode, Peer, Client), enforce wallet/config consistency for non-Root roles, and integrate role selection into the installation process.

**Key Concepts:**

*   **Roles:** Explicitly define node roles (Root, Bootnode, Peer, Client).
*   **Config Locations:** Differentiate config file locations based on role (App Root for Root, OS App Data Dir for others).
*   **Wallet Consistency:** For non-Root roles, enforce that if an address is in the config, a corresponding `wallet.dat` must exist and match. If not, trigger re-installation.
*   **Installer Role Selection:** Make the installer interactive, asking the user their intended role.

## Phase 1: Define Roles and Refactor Configuration Management

**Goal:** Introduce node roles and update the config package to handle role-specific config file locations and path helpers.

**Tasks:**

1.  **Define Roles:** Create an enum or set of constants (e.g., `Root`, `RoleBootnode`, `RolePeer`, `RoleClient`) to represent the node roles.

2.  **Update `config` Package:**

    *   **Add a new function `GetAppConfigDir()`:** This function returns the appropriate OS-specific application data directory path (e.g., `~/.config/KNIRVCHAIN` on Linux, `%APPDATA%\KNIRVCHAIN` on Windows, `~/Library/Application Support/KNIRVCHAIN` on macOS).

    *   **Modify `LoadConfig(configPath string, role Role)`:**

        *   If `configPath` is explicitly provided (via flag), use that path regardless of role.
        *   If `configPath` is empty:
            *   If `role == Root`, attempt to load `config.json` from the executable's directory.
            *   Otherwise (Bootnode, Peer, Client), attempt to load `config.json` from `GetAppConfigDir()`.
            *   Handle the case where the config file is not found by returning a default config and indicating it's a new config.

    *   **Modify `SaveConfig(configPath string, cfg *Config, role Role)`:**

        *   If `configPath` is explicitly provided, save to that path.
        *   If `configPath` is empty:
            *   If `role == Root`, save to the executable's directory.
            *   Otherwise, save to `GetAppConfigDir()`.
            *   Ensure the target directory exists before saving.

    *   **Update existing path helper functions:** (`GetWalletPath`, `GetPeerWalletPath`, `GetMasterWalletPath`, `GetBlockchainDatabasePath`, `GetSearchableDatabasePath`, `GetReflectionDatabasePath`)
        *   Accept a `Role` parameter.
        *   Use `GetAppConfigDir()` as the base directory for non-Root roles.
        *   The Root role paths should remain relative to the executable or use system-wide locations if appropriate (e.g., `/var/lib/KNIRVCHAIN`).

3.  **Update `main.go`:**

    *   Parse command-line flags (`--root`, `--bootnode`, `--dev`, `--client-only`) early to determine the node's `Role`. If multiple or none are specified, define clear default behavior (e.g., `--root` takes precedence, then `--bootnode`, then `--dev`, default to Client if no role flag).
    *   Call `config.LoadConfig` with the determined `Role`.
    *   The logic for applying flag overrides should be reviewed to ensure flags correctly override config values based on the primary node being run by this instance and its role.

## Phase 2: Centralize Wallet Management and Implement Consistency Checks

**Goal:** Create a dedicated module for wallet loading/saving and implement the wallet/config consistency checks for non-Root roles in `main.go`.

**Tasks:**

1.  **Create Wallet Manager:** Create a new file/package (e.g., `wallet_manager.go`).

2.  **Move Wallet I/O:** Move `loadWalletFromFile` and `saveWalletToFile` (currently scattered in `wallet_server.go` and `main.go`) into `wallet_manager.go`. These functions should:

    *   Accept the `Role`.
    *   Use the appropriate path helpers from the `config` package to determine the wallet file location (`wallet.dat`, `master_wallet.dat`).

3.  **Add Wallet Loading Functions:** Add functions like `wallet_manager.LoadWallet(address string, role Role)` and `wallet_manager.LoadMasterWallet(address string, role Role)` which use the file loading functions.  These should:

    *   Return the `*Wallet` and an `error` (e.g., `os.ErrNotExist` if the file is missing, or a specific error if the address in the file doesn't match the requested address).

4.  **Implement Consistency Checks in `main.go`:**

    *   After loading the config and determining the `Role`:
        *   If `Role != Root`:
            *   **Check `MinersAddress`:**
                *   If `cfg.MinersAddress` is not empty: Attempt to load the wallet using `wallet_manager.LoadWallet(cfg.MinersAddress, Role)`. If it returns an error (file missing, mismatch, etc.), print a clear error message indicating the issue and the need to re-install, then exit with a specific status code (e.g., 1). If successful, store the loaded wallet object.
                *   If `cfg.MinersAddress` is empty: Attempt to load any wallet from the default path using `wallet_manager.LoadWallet("", Role)`. If successful, update `cfg.MinersAddress` in the config and save the config. If it returns `os.ErrNotExist`, print an error indicating no wallet found and the need to re-install, then exit with status code 1.
            *   **Check `MasterAddress` (if `Role` is Bootnode):**
                *   If `cfg.MasterAddress` is not empty: Attempt to load the master wallet using `wallet_manager.LoadMasterWallet(cfg.MasterAddress, Role)`. If it returns an error, print an error and exit with status code 1. If successful, store the loaded master wallet object.
                *   If `cfg.MasterAddress` is empty: Attempt to load any master wallet from the default path. If successful, update `cfg.MasterAddress` and save the config. If it returns `os.ErrNotExist`, print an error and exit with status code 1.

    *   Modify the entry point logic in `main` to detect the exit code 1 and launch the installer (`install.go`) if needed.

    *   Update `getMasterWallet`: Modify `getMasterWallet` in `master_wallet.go` to use the new `wallet_manager` functions and be aware of the `Role` (only load/generate for Root/Bootnode).

## Phase 3: Refactor Installer with Role Selection and Role-Specific Setup

**Goal:** Make the installer interactive, asking for the user's role and performing setup steps specific to that role, including conditional wallet generation and config saving.

**Tasks:**

1.  **Add Role Prompt:** Add a step at the beginning of `install.go` to prompt the user to select their `Role` (Root, Bootnode, Peer, Client). Store the selected role.

2.  **Conditional URI Generation:**

    *   If the selected `Role` is `Root`, skip the step of connecting to a Root node to generate a URI. The Root node's URI is inherent or defined in its root config.
    *   For other roles, proceed with connecting to a Root node (prompting for its endpoint) to generate a unique chain URI.

3.  **Conditional Wallet Generation:**

    *   If `Root`, skip `wallet.dat` generation. The address comes from the root config file.
    *   If `RoleBootnode`, generate both a dev wallet (`wallet.dat`) and a master wallet (`master_wallet.dat`). Use `wallet_manager.SaveWallet` and `wallet_manager.SaveMasterWallet` with the `RoleBootnode`.
    *   If `RolePeer`, generate only a dev wallet (`wallet.dat`). Use `wallet_manager.SaveWallet` with the `RolePeer`.
    *   If `RoleClient`, skip wallet generation.

4.  **Update Config in Installer:**

    *   Load the existing config using `config.LoadConfig` with the selected `Role`.
    *   Update the config object (`cfg`) with role-specific details:
        *   `ChainID`: Set based on the generated URI (if not Root).
        *   `MinersAddress`: Set to the generated dev wallet address (if Bootnode or Peer).
        *   `MasterAddress`: Set to the generated master wallet address (if Bootnode).
        *   `InstallComplete`: Set to `true`.
        *   `IsBootnode`: Set based on the selected role.
        *   `ClientOnly`: Set based on the selected role.
        *   `UseGUI`: Prompt the user if they want a GUI, set accordingly.
        *   `PaymentProcessor.Enabled`: Set to `true` if `Root` or `RoleBootnode` (or prompt?).
    *   Save the updated config using `config.SaveConfig` with the selected `Role`, ensuring it goes to the correct location.

5.  **Configure Service:** Modify the `ConfigureSystemService` call in `install.go` to pass the selected `Role`.

6.  **Final Launch:** The final launch step should use the config file path determined earlier by `config.SaveConfig`.

## Phase 4: Service Configuration and Uninstall Updates

**Goal:** Ensure system services are configured correctly based on role and update uninstall logic to clean up role-specific files.

**Tasks:**

1.  **Update Service Configuration:** Modify `ConfigureSystemService` and its OS-specific helper functions (`configureLinuxService`, `configureWindowsService`, `configureMacOSService`) to accept the `Role` parameter. Use the `Role` to determine:

    *   The service name (e.g., `KNIRVCHAIN-root`, `KNIRVCHAIN-bootnode`, `KNIRVCHAIN-dev`, `KNIRVCHAIN-client`).
    *   The command-line arguments passed to the executable (e.g., `-config <path> -role <role>`).
    *   Potentially different service user, working directory, or log paths based on role (e.g., Root might use system-wide paths, others user-specific).

2.  **Update Uninstall:**

    *   Modify `Uninstall()` to determine the role of the installed node (perhaps by checking the config file location or a marker file).
    *   Modify `cleanupConfiguration()` to remove config files and wallet files (`wallet.dat`, `master_wallet.dat`) from the OS-specific app data directory based on the determined role.
    *   Modify `UnregisterURIHandlers()` and its OS-specific helpers to remove service configurations based on the determined role and service name.

## Phase 5: GUI Integration and Testing

**Goal:** Ensure the GUI correctly reflects the node's role and wallet status, and perform comprehensive testing of all scenarios.

**Tasks:**

1.  **GUI Wallet Status:** In `main.go`, pass the loaded wallet object (if any) and the node's `Role` to `NewGUI`.

2.  **GUI Role Display:** Update `gui.go` to display the node's `Role` on the dashboard.

3.  **GUI Feature Toggling:** Update `gui.go` to enable/disable or show/hide GUI elements (e.g., payment processor tabs, mining controls) based on the node's `Role` and the availability of the necessary wallet (e.g., mining requires `MinersAddress` and `wallet.dat`, payment processing requires `MasterAddress` and `master_wallet.dat`).

4.  **Comprehensive Testing:**

    *   Write/update unit tests for the refactored `config` and `wallet_manager` packages.
    *   Write/update integration tests covering:
        *   Running `main` for each role with valid config/wallet setups.
        *   Running `main` for non-Root roles with missing config/wallet files and verifying the re-install trigger.
        *   Running the `install.go` process for each role and verifying correct config/wallet generation and placement.
        *   Verifying service installation/uninstallation for each role.
        *   Testing multi-node scenarios with different roles interacting (e.g., Peer connecting to Bootnode, Peer getting URI from Root).
        *   Testing GUI behavior for different roles.

This plan provides a structured approach to implementing the requested features, ensuring that configuration, wallet management, and installation are tightly integrated and role-aware. It highlights the need for careful handling of file paths and consistency checks, especially for non-Root nodes.
```

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
