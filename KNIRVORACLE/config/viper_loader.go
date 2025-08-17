package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// LoadConfigurationViper initializes Viper and loads the configuration
// for the specified role.
func LoadConfigurationViper(role Role, cliConfigPath string) (*Config, string, error) {
	// Special case for Root role - use constants and matrix, then resolve paths.
	if role == Root {
		log.Println("Viper: Root role detected. Constructing config from constants and matrix, then resolving paths.")

		// Create a config directly from constants and settings matrix
		rootCfg := CreateRootConfigFromMatrixAndConstants()

		// Resolve dynamic paths for the Root config
		if err := resolveDynamicPathsViper(rootCfg, Root, false); err != nil { // false as no config file was strictly "used" for root base
			return nil, "", fmt.Errorf("viper: failed to resolve dynamic paths for Root config: %w", err)
		}
		// Return the config with empty config path (indicating no file was used)
		return rootCfg, "", nil
	}

	// For all other roles, proceed with normal Viper configuration loading
	v := viper.New()

	// --- Configuration File Paths & Loading Strategy ---
	var baseConfigFileLoaded string
	var roleConfigFileLoaded string

	if cliConfigPath != "" { // Path from command line flag has highest priority for file location
		v.SetConfigFile(cliConfigPath) // If specific file is given
		// Default search paths
		v.AddConfigPath(".") // Current directory
		roleDataDir, err := GetDataDir(role)
		if err == nil {
			v.AddConfigPath(roleDataDir) // e.g., ~/.config/KNIRVORACLE/dev_data
		}
		log.Printf("Viper: Attempting to load specific config file from CLI: %s", cliConfigPath)
		if err := v.ReadInConfig(); err != nil {
			// If the specific file isn't found, it's an error because it was explicitly provided.
			return nil, "", fmt.Errorf("viper: error reading explicitly provided config file '%s': %w", cliConfigPath, err)
		}
		roleConfigFileLoaded = v.ConfigFileUsed()
		log.Printf("Viper: Using specific config file from CLI: %s", roleConfigFileLoaded)
	} else {
		// Strategy: Load default_config.json, then [role]_config.json
		v.SetConfigType("json")

		// Setup search paths
		v.AddConfigPath(".")
		userConfigDir, _ := os.UserConfigDir()
		if userConfigDir != "" {
			v.AddConfigPath(filepath.Join(userConfigDir, AppName))
		}
		v.AddConfigPath(filepath.Join("/etc", AppName))
		roleDataDir, _ := GetDataDir(role) // For role-specific directory
		if roleDataDir != "" {
			v.AddConfigPath(roleDataDir)
		}

		// Attempt to load default_config.json
		v.SetConfigName("default_config")
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				log.Printf("Viper: Warning: Error reading default_config.json: %v", err)
			} // else: default_config.json not found is acceptable
		} else {
			baseConfigFileLoaded = v.ConfigFileUsed()
			log.Printf("Viper: Loaded base defaults from: %s", baseConfigFileLoaded)
		}

		// Attempt to load [role]_config.json and merge it
		roleConfigFilename := fmt.Sprintf("%s_config", strings.ToLower(role.String()))
		v.SetConfigName(roleConfigFilename)
		if err := v.MergeInConfig(); err != nil { // MergeInConfig overlays on top of existing config
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				log.Printf("Viper: Role-specific config file '%s.json' not found. Using defaults.", roleConfigFilename)
			} else {
				log.Printf("Viper: Warning: Error reading role-specific config file '%s.json': %v", roleConfigFilename, err)
			}
		} else {
			roleConfigFileLoaded = v.ConfigFileUsed() // This will be the role-specific file path
			log.Printf("Viper: Loaded and merged role-specific config from: %s", roleConfigFileLoaded)
		}
	}

	// 3. Environment Variable Binding
	v.SetEnvPrefix("agent") // e.g., agent_HTTPPORT=5001
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // e.g., roles.Root.httpPort -> agent_ROLES_CREATOR_HTTPPORT

	// Explicitly bind environment variables for Cerebras configuration
	// These will be read from the environment after godotenv.Load() in main.go
	v.BindEnv("chromem.cerebras_config.api_key", "DEFAULT_CEREBRAS_API_KEY")
	v.BindEnv("chromem.cerebras_config.base_url", "DEFAULT_CEREBRAS_BASE_URL")

	// Determine which config file path to report back (role-specific takes precedence if loaded)
	finalConfigFileUsed := roleConfigFileLoaded
	if finalConfigFileUsed == "" {
		finalConfigFileUsed = baseConfigFileLoaded
	}

	// 5. Create the final Config struct
	var cfg Config
	baseDefault := DefaultConfig()
	cfg = *baseDefault

	// Log the current values before any operations
	log.Printf("Viper: Initial config values - ChainID: '%s', HTTPPort: %d, P2PPort: %d",
		cfg.ChainID, cfg.Port, cfg.P2PPort)

	// Unmarshal the entire Viper configuration into the cfg struct FIRST.
	// Viper has already merged default_config and [role]_config internally if they were found.
	// Environment variables will also override file values at this point.
	if err := v.Unmarshal(&cfg); err != nil {
		log.Printf("Viper: Warning: Could not unmarshal final config: %v. Using struct defaults.", err)
		// Keep using the baseDefault if unmarshal fails completely
	} else {
		log.Printf("Viper: After unmarshalling main config - ChainID: '%s', HTTPPort: %d, P2PPort: %d", cfg.ChainID, cfg.Port, cfg.P2PPort)
	}

	// Only apply role-specific defaults for fields that are still at their zero values
	// This ensures we don't overwrite values from config files or command-line flags
	log.Printf("Viper: Before applying role defaults - ChainID: '%s', HTTPPort: %d, P2PPort: %d",
		cfg.ChainID, cfg.Port, cfg.P2PPort)
	ApplyRoleDefaults(&cfg, role)
	log.Printf("Viper: After applying role defaults - ChainID: '%s', HTTPPort: %d, P2PPort: %d",
		cfg.ChainID, cfg.Port, cfg.P2PPort)

	// Explicitly set critical paths from Viper if a config file was loaded.
	// This ensures that paths specified in the config file override any defaults
	// that might have been set or would be set during Unmarshal.
	// Doing this AFTER Unmarshal ensures these values take precedence.
	if finalConfigFileUsed != "" {
		log.Printf("LoadConfig: Config file '%s' was specified and merged. Explicitly checking path keys.", finalConfigFileUsed)
		if v.IsSet("paths.blockchain_database_path") {
			cfg.BlockchainDatabasePath = v.GetString("paths.blockchain_database_path")
			log.Printf("Viper: After unmarshalling paths - BlockchainDB: '%s', SearchableDB: '%s'", cfg.BlockchainDatabasePath, cfg.SearchableDatabasePath)
		}
		if v.IsSet("paths.searchable_database_path") {
			cfg.SearchableDatabasePath = v.GetString("paths.searchable_database_path")
			log.Printf("LoadConfig: Set SearchableDatabasePath directly from loaded config file: %s", cfg.SearchableDatabasePath)
		}
		// Note: The main Config struct doesn't have a direct WalletPath field
		// It's handled through GetWalletPath function, so we don't set it directly here
		if v.IsSet("paths.wallet_path") {
			log.Printf("LoadConfig: Found wallet_path in config file, but it's handled through GetWalletPath function")
		}
		if v.IsSet("paths.reflection_database_path") {
			cfg.ReflectionDatabasePath = v.GetString("paths.reflection_database_path")
			log.Printf("LoadConfig: Set ReflectionDatabasePath directly from loaded config file: %s", cfg.ReflectionDatabasePath)
		}
	}

	// 6. Apply direct Viper Get overrides for specific keys if needed (after unmarshalling)

	// 8. Handle dynamic paths (BlockchainDatabasePath, etc.)
	// This logic remains important as paths often depend on the role and runtime environment.
	if err := resolveDynamicPathsViper(&cfg, role, finalConfigFileUsed != ""); err != nil { // Pass if a config file was used
		return nil, finalConfigFileUsed, fmt.Errorf("viper: failed to resolve dynamic paths: %w", err)
	}

	// 9. Ensure ChainID is set (especially for roles that depend on it)
	// If not set anywhere, it will be the zero value (false), correctly triggering install.
	// The 'Root' role in the example YAML/JSON sets it to true.
	log.Printf("Viper: Final config values before return - ChainID: '%s', HTTPPort: %d, P2PPort: %d",
		cfg.ChainID, cfg.Port, cfg.P2PPort)

	return &cfg, finalConfigFileUsed, nil
}

// resolveDynamicPathsViper is similar to your existing resolveDynamicPaths,
// but adapted to be called after Viper has loaded the initial config.
func resolveDynamicPathsViper(cfg *Config, role Role, _ bool) error {
	// This function ensures cfg.BlockchainDatabasePath (and SearchableDatabasePath if applicable) are absolute.

	// Resolve BlockchainDatabasePath
	if cfg.BlockchainDatabasePath != "" { // Path was set by Viper (config file, env, or flag)
		if !filepath.IsAbs(cfg.BlockchainDatabasePath) {
			// Path is relative, make it absolute from CWD
			absPath, err := filepath.Abs(cfg.BlockchainDatabasePath)
			if err != nil {
				return fmt.Errorf("failed to make BlockchainDatabasePath absolute %s: %w", cfg.BlockchainDatabasePath, err)
			}
			cfg.BlockchainDatabasePath = absPath
			log.Printf("Viper: Resolved relative BlockchainDatabasePath from config/env/flag to absolute: %s", cfg.BlockchainDatabasePath)
		}
		// If it was already absolute, it's used as is.
	} else {
		// Path was not set by Viper, so derive default based on role.
		var dbFilename string
		switch role {
		case RoleBootnode:
			bootnodeChainID := cfg.ChainID
			if bootnodeChainID == "" {
				bootnodeChainID = "default"
			}
			dbFilename = fmt.Sprintf("bootnode_%s.db", bootnodeChainID)
		case RolePeer:
			devChainID := cfg.ChainID
			if devChainID == "" {
				devChainID = "default"
			}
			dbFilename = fmt.Sprintf("dev_%s.db", devChainID)
		case RoleClient:
			// For clients, ChainID might be their wallet address, set later by installer.
			// If empty here, use a generic name or role name.
			clientChainID := cfg.ChainID
			if clientChainID == "" {
				clientChainID = strings.ToLower(role.String())
			}
			dbFilename = fmt.Sprintf("client_%s.db", clientChainID)
		default:
			dbFilename = "agent.db"
		}
		baseDataDir, err := GetDataDir(role)
		if err != nil {
			return fmt.Errorf("failed to get data dir for role %s: %w", role, err)
		}
		cfg.BlockchainDatabasePath = filepath.Join(baseDataDir, dbFilename)
		log.Printf("Viper: Resolved default BlockchainDatabasePath for role %s to: %s", role, cfg.BlockchainDatabasePath)
	}

	// Similar logic for SearchableDatabasePath
	if cfg.SearchableDatabasePath != "" { // Path was set by Viper
		if !filepath.IsAbs(cfg.SearchableDatabasePath) {
			absPath, err := filepath.Abs(cfg.SearchableDatabasePath)
			if err != nil {
				return fmt.Errorf("failed to make SearchableDatabasePath absolute %s: %w", cfg.SearchableDatabasePath, err)
			}
			cfg.SearchableDatabasePath = absPath
			log.Printf("Viper: Resolved relative SearchableDatabasePath from config/env/flag to absolute: %s", cfg.SearchableDatabasePath)
		}
	} else {
		// Path was not set by Viper, derive default.
		chromemDirName := "chromem_data_store" // Default directory name for Chromem data
		baseDataDir, err := GetDataDir(role)
		if err != nil {
			return fmt.Errorf("failed to get data dir for role %s: %w", role, err)
		}
		cfg.SearchableDatabasePath = filepath.Join(baseDataDir, chromemDirName)
		log.Printf("Viper: Resolved default SearchableDatabasePath for role %s to: %s", role, cfg.SearchableDatabasePath)
	}
	return nil
}
