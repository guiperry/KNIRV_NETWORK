package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"KNIRVENGINE/desktop-client/database"
	"KNIRVENGINE/desktop-client/utils"
)

const defaultAgentImage = "/Agentify_logo_2.png"

func main() {
	log.Println("🔄 Starting agent image update process...")

	// Get database directory
	dbDir, err := utils.GetDatabaseDir()
	if err != nil {
		log.Fatalf("Failed to get database directory: %v", err)
	}
	domainDBPath := filepath.Join(dbDir, "domain.db")

	// Check if database exists
	if _, err := os.Stat(domainDBPath); os.IsNotExist(err) {
		log.Printf("⚠️  Database not found at %s. No agents to update.", domainDBPath)
		return
	}

	// Initialize domain database
	domainDB, err := database.NewSimpleDomainDB(domainDBPath)
	if err != nil {
		log.Fatalf("Failed to initialize domain database: %v", err)
	}
	defer domainDB.Close()

	// Get agent collection and create repository
	agentCollection, err := domainDB.GetOrCreateCollection("agents")
	if err != nil {
		log.Fatalf("Failed to get agent collection: %v", err)
	}

	agentRepo := database.NewSimpleAgentRepository(agentCollection)
	ctx := context.Background()

	// Get all agents for user ID 1 (assuming default user)
	agents, err := agentRepo.GetAgentsByOwner(ctx, 1)
	if err != nil {
		log.Fatalf("Failed to list agents: %v", err)
	}

	if len(agents) == 0 {
		log.Println("📭 No agents found in database.")
		return
	}

	log.Printf("📊 Found %d agents to potentially update", len(agents))

	updatedCount := 0
	skippedCount := 0

	// Update each agent
	for _, agent := range agents {
		log.Printf("🔍 Checking agent: %s (ID: %s)", agent.Name, agent.ID)

		// Check if agent already has the new default image
		if agent.ImageURL == defaultAgentImage {
			log.Printf("✅ Agent %s already has the new default image, skipping", agent.Name)
			skippedCount++
			continue
		}

		// Check if agent has a custom image (not one of the old default images)
		if isCustomImage(agent.ImageURL) {
			log.Printf("🎨 Agent %s has a custom image (%s), skipping", agent.Name, agent.ImageURL)
			skippedCount++
			continue
		}

		// Update the agent's image
		log.Printf("🔄 Updating agent %s image from %s to %s", agent.Name, agent.ImageURL, defaultAgentImage)

		agent.ImageURL = defaultAgentImage

		if err := agentRepo.UpdateAgent(ctx, agent); err != nil {
			log.Printf("❌ Failed to update agent %s: %v", agent.Name, err)
			continue
		}

		log.Printf("✅ Successfully updated agent %s", agent.Name)
		updatedCount++
	}

	// Note: API-based agents (with JSON config) will be updated automatically
	// by the frontend when they are loaded, as it now defaults to the new image

	// Summary
	log.Println("\n📋 Update Summary:")
	log.Printf("✅ Updated agents: %d", updatedCount)
	log.Printf("⏭️  Skipped agents: %d", skippedCount)
	log.Printf("📊 Total agents processed: %d", updatedCount+skippedCount)

	if updatedCount > 0 {
		log.Println("\n🎉 Agent image update completed successfully!")
		log.Println("💡 All updated agents will now use the Agentify logo as their default image.")
	} else {
		log.Println("\n✨ No agents needed updating. All agents already have appropriate images.")
	}
}

// isCustomImage checks if the image URL is a custom image (not a default Pexels image)
func isCustomImage(imageURL string) bool {
	// List of old default images that should be updated
	oldDefaultImages := []string{
		"https://images.pexels.com/photos/5380664/pexels-photo-5380664.jpeg?auto=compress&cs=tinysrgb&w=400",
		"https://images.pexels.com/photos/5380617/pexels-photo-5380617.jpeg?auto=compress&cs=tinysrgb&w=400",
		"https://images.pexels.com/photos/5380613/pexels-photo-5380613.jpeg?auto=compress&cs=tinysrgb&w=400",
		"https://images.pexels.com/photos/5380665/pexels-photo-5380665.jpeg?auto=compress&cs=tinysrgb&w=400",
		"https://images.pexels.com/photos/5380668/pexels-photo-5380668.jpeg?auto=compress&cs=tinysrgb&w=400",
		"https://images.pexels.com/photos/5380671/pexels-photo-5380671.jpeg?auto=compress&cs=tinysrgb&w=400",
		"https://example.com/alpha.png",
		"https://example.com/beta.png",
		"https://example.com/gamma.png",
		"", // Empty image URL should also be updated
	}

	// If the image URL is in the list of old defaults, it's not a custom image
	for _, oldDefault := range oldDefaultImages {
		if imageURL == oldDefault {
			return false
		}
	}

	// If it's not an old default, consider it custom
	return true
}
