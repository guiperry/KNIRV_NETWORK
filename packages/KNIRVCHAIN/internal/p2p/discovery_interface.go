package p2p

import (
	"KNIRVCHAIN/config"
	"KNIRVCHAIN/internal/types"
	"KNIRVCHAIN/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// DiscoveryService defines the interface for discovery operations needed by the server
type DiscoveryService interface {
	// For generic resources like "chain"
	FindGenericResource(id string, resourceType types.DiscoveryResourceType) ([]peer.AddrInfo, error)
	AnnounceGenericResource(id string, resourceType types.DiscoveryResourceType) error
	// For MCP capabilities using their specific type strings (e.g., "resource", "tool")
	FindMCPCapabilityProviders(ctx context.Context, id string, mcpTypeString string) ([]peer.AddrInfo, error)
	AnnounceMCPCapability(ctx context.Context, id string, mcpTypeString string) error
	// For specific resource types
	FindResource(ctx context.Context, id string, resourceType types.DiscoveryResourceType) ([]peer.AddrInfo, error)
	AnnounceMintedResource(ctx context.Context, id string, resourceType types.DiscoveryResourceType) error
	// Management methods
	Run(interval time.Duration)
	GetPeerID() string
	GetSelfMultiaddrs() []string
	Close()
	// Check if a node was recently registered
	IsRecentlyRegistered(nodeID string) bool
	// Connect to a specific node
	ConnectToPeer(nodeInfo peer.AddrInfo, ctx context.Context) error
}

// fetchAndStorePublicIPInfo fetches the public IP information and stores it in the config
func fetchAndStorePublicIPInfo(cfg *config.Config, role config.Role) (errCatch error) {
	log.Println("Attempting to fetch and store public IP information...")

	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in fetchAndStorePublicIPInfo: %v. Recovered. IP info might be stale or unavailable.", r)
			errCatch = fmt.Errorf("panic during public IP fetch: %v", r)
			// Ensure LastIPInfoResponse is nil if a panic occurred during its fetching/parsing
			utils.LastIPInfoResponse = nil
		}
	}()

	// fetchPublicIPFromIPInfo is in net_utils.go and populates net_utils.LastIPInfoResponse
	_, err := utils.FetchPublicIPFromIPInfo() // from net_utils.go
	if err != nil {
		log.Printf("Warning: Failed to fetch public IP information: %v. Will proceed without it or use previously cached data in config.", err)
		// If LastIPInfoResponse is nil here, it means the fetch failed and there was no prior successful fetch in this session.
		utils.LastIPInfoResponse = nil // Ensure LastIPInfoResponse is nil if fetch failed
		// The config might still hold outdated info if previously saved.
		// If IP info is critical for startup, you might return an error.
		// For now, we'll allow proceeding, and cfg.PublicIPInfo might remain nil or outdated.
	}

	if utils.LastIPInfoResponse == nil { // LastIPInfoResponse is from net_utils
		log.Println("No new public IP information fetched and no cached information from current session available.")
		// If cfg.PublicIPInfo is already populated from a previous run (loaded from file), we might not want to overwrite it with nil.
		// However, if the fetch failed, we probably shouldn't claim we have fresh IP info.
		// Let's only update cfg.PublicIPInfo if LastIPInfoResponse is not nil.
		if cfg.PublicIPInfo != nil {
			log.Println("Retaining previously configured PublicIPInfo as current fetch failed.")
		}
		return nil // Not returning an error, allowing startup with potentially stale/missing IP info
	}

	// Marshal the IPInfoResponse to JSON for storage
	ipInfoJSON, err := json.Marshal(utils.LastIPInfoResponse)
	if err != nil {
		log.Printf("Warning: Failed to marshal IPInfoResponse to JSON: %v", err)
		return fmt.Errorf("failed to marshal IPInfoResponse: %w", err)
	}

	// Handle storage differently based on role
	if role == config.Root {
		// For Root role, save to root-data/env.local
		log.Println("Root role detected: Saving IP info to user's root-data directory (env.local)")

		// Get the user-specific root_data directory. config.GetDataDir ensures the directory exists.
		rootDataDir, err := config.GetDataDir(config.Root)
		if err != nil {
			log.Printf("Warning: Failed to get root data directory: %v. IP info will not be saved to env.local.", err)
		} else {
			rootDataEnvPath := filepath.Join(rootDataDir, "env.local")
			log.Printf("Target path for env.local: %s", rootDataEnvPath)

			// UpdateEnvVariable will create the file if it doesn't exist.
			// It also handles creating parent directories if needed via its os.CreateTemp logic.
			// We now want to store only the IP address string, not the full JSON.
			ipAddressFromString := ""
			if utils.LastIPInfoResponse != nil { // Ensure LastIPInfoResponse is not nil
				ipAddressFromString = utils.LastIPInfoResponse.IP
			}
			if err := utils.UpdateEnvVariable(rootDataEnvPath, "NEXT_PUBLIC_IP_INFO", ipAddressFromString); err != nil {
				log.Printf("Warning: Failed to update IPInfo in %s: %v", rootDataEnvPath, err)
			} else {
				log.Printf("Successfully saved IPInfo to %s for Root role", rootDataEnvPath)
			}
		}
	} else {
		// For non-Root roles, save to config file
		// Convert the IPInfoResponse to map[string]interface{} for storage in config
		ipInfoMap := make(map[string]interface{})
		if err := json.Unmarshal(ipInfoJSON, &ipInfoMap); err != nil {
			log.Printf("Warning: Failed to unmarshal IPInfo JSON to map: %v", err)
			return fmt.Errorf("failed to unmarshal IPInfo to map: %w", err)
		}

		cfg.PublicIPInfo = ipInfoMap // Store in the main config struct
		// Save the updated configuration
		config.SaveMinimalConfigToUserDir(cfg, role)
		log.Printf("Successfully stored public IP information in config for role %s.", role.String())
	}

	// If this is the Root node, update ROOTCHAIN_URL
	if role == config.Root && utils.LastIPInfoResponse.IP != "" {
		newRootURL := fmt.Sprintf("http://%s:%d", utils.LastIPInfoResponse.IP, cfg.Port)
		utils.SetRootchainURL(newRootURL) // This function is in constants.go
		log.Printf("Updated ROOTCHAIN_URL to: %s based on fetched public IP.", newRootURL)
	}
	return nil
}
