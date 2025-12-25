package dverental

import (
	"backend_server/internal/objects"
	"encoding/json"
	"log"
	"time"

	"github.com/tidwall/buntdb"
)

// Database operations

// loadFromDatabase loads rental data from the database
func (drs *DVERentalService) loadFromDatabase() error {
	return drs.db.View(func(tx *buntdb.Tx) error {
		// Load active rentals
		tx.Ascend("", func(key, value string) bool {
			if len(key) > 8 && key[:8] == "rental:" {
				var rental objects.DVERental
				if err := json.Unmarshal([]byte(value), &rental); err == nil {
					if rental.Status == "active" {
						drs.activeRentals[rental.ID] = &rental
					}
				}
			}
			return true
		})

		// Load rental plans
		tx.Ascend("", func(key, value string) bool {
			if len(key) > 5 && key[:5] == "plan:" {
				var plan objects.RentalPlan
				if err := json.Unmarshal([]byte(value), &plan); err == nil {
					drs.rentalPlans[plan.ID] = &plan
				}
			}
			return true
		})

		return nil
	})
}

// saveToDatabase saves all rental data to the database
func (drs *DVERentalService) saveToDatabase() error {
	return drs.db.Update(func(tx *buntdb.Tx) error {
		// Save active rentals
		for _, rental := range drs.activeRentals {
			if data, err := json.Marshal(rental); err == nil {
				tx.Set("rental:"+rental.ID, string(data), nil)
			}
		}

		// Save rental plans
		for _, plan := range drs.rentalPlans {
			if data, err := json.Marshal(plan); err == nil {
				tx.Set("plan:"+plan.ID, string(data), nil)
			}
		}

		return nil
	})
}

// saveRentalToDatabase saves a single rental to the database
func (drs *DVERentalService) saveRentalToDatabase(rental *objects.DVERental) error {
	return drs.db.Update(func(tx *buntdb.Tx) error {
		if data, err := json.Marshal(rental); err == nil {
			tx.Set("rental:"+rental.ID, string(data), nil)
		}
		return nil
	})
}

// Rental plan management

// createDefaultRentalPlans creates the default rental plans
func createDefaultRentalPlans() []*objects.RentalPlan {
	return []*objects.RentalPlan{
		{
			ID:           "basic",
			Name:         "Basic DVE",
			Description:  "Basic DVE rental with limited resources",
			PricePerHour: 10, // 10 NRN per hour
			ResourceLimits: objects.ResourceLimits{
				MaxCPU:       1.0,
				MaxMemory:    1024 * 1024 * 1024,     // 1GB
				MaxDisk:      5 * 1024 * 1024 * 1024, // 5GB
				MaxBandwidth: 100 * 1024 * 1024,      // 100MB/s
			},
			MaxDuration: 24 * 60 * 60, // 24 hours
			MinDuration: 60 * 60,      // 1 hour
			Features:    []string{"Basic CDE", "SSH Access", "Web Terminal"},
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:           "standard",
			Name:         "Standard DVE",
			Description:  "Standard DVE rental with moderate resources",
			PricePerHour: 25, // 25 NRN per hour
			ResourceLimits: objects.ResourceLimits{
				MaxCPU:       2.0,
				MaxMemory:    4 * 1024 * 1024 * 1024,  // 4GB
				MaxDisk:      20 * 1024 * 1024 * 1024, // 20GB
				MaxBandwidth: 500 * 1024 * 1024,       // 500MB/s
			},
			MaxDuration: 7 * 24 * 60 * 60, // 7 days
			MinDuration: 60 * 60,          // 1 hour
			Features:    []string{"Enhanced CDE", "SSH Access", "Web Terminal", "GPU Access", "Custom Images"},
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:           "premium",
			Name:         "Premium DVE",
			Description:  "Premium DVE rental with high-performance resources",
			PricePerHour: 50, // 50 NRN per hour
			ResourceLimits: objects.ResourceLimits{
				MaxCPU:       8.0,
				MaxMemory:    16 * 1024 * 1024 * 1024,  // 16GB
				MaxDisk:      100 * 1024 * 1024 * 1024, // 100GB
				MaxBandwidth: 1024 * 1024 * 1024,       // 1GB/s
			},
			MaxDuration: 30 * 24 * 60 * 60, // 30 days
			MinDuration: 60 * 60,           // 1 hour
			Features:    []string{"Premium CDE", "SSH Access", "Web Terminal", "GPU Access", "Custom Images", "Priority Support", "Dedicated Resources"},
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
}

// initializeDefaultPlans initializes the default rental plans
func (drs *DVERentalService) initializeDefaultPlans() {
	log.Printf("[DVE Rental] Initializing default plans, count: %d", len(drs.defaultPlans))
	for _, plan := range drs.defaultPlans {
		log.Printf("[DVE Rental] Adding plan: %s - %s", plan.ID, plan.Name)
		drs.rentalPlans[plan.ID] = plan
	}

	log.Printf("[DVE Rental] Total plans after initialization: %d", len(drs.rentalPlans))

	// Save to database
	if err := drs.saveToDatabase(); err != nil {
		log.Printf("Warning: Failed to save default rental plans to database: %v", err)
	} else {
		log.Printf("[DVE Rental] Successfully saved default plans to database")
	}
}

// Utility functions

// countActiveRentals counts currently active rentals
func (drs *DVERentalService) countActiveRentals() int64 {
	count := int64(0)
	now := time.Now()
	for _, rental := range drs.activeRentals {
		if rental.Status == "active" && rental.EndTime.After(now) {
			count++
		}
	}
	return count
}

// calculateTotalRevenue calculates total NRN revenue collected
func (drs *DVERentalService) calculateTotalRevenue() int64 {
	total := int64(0)
	for _, rental := range drs.activeRentals {
		total += rental.NRNAmount
	}
	return total
}

// calculateRevenueForPeriod calculates revenue for a specific time period
func (drs *DVERentalService) calculateRevenueForPeriod(period time.Duration) int64 {
	cutoff := time.Now().Add(-period)
	total := int64(0)
	for _, rental := range drs.activeRentals {
		if rental.CreatedAt.After(cutoff) {
			total += rental.NRNAmount
		}
	}
	return total
}

// cleanupRoutine periodically cleans up expired rentals
func (drs *DVERentalService) cleanupRoutine() {
	ticker := time.NewTicker(drs.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !drs.running {
				return
			}
			drs.cleanupExpiredRentals()
		}
	}
}

// cleanupExpiredRentals removes expired rentals and handles automatic renewal
func (drs *DVERentalService) cleanupExpiredRentals() {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	now := time.Now()
	expiredRentals := make([]string, 0)
	renewedRentals := make([]string, 0)

	for id, rental := range drs.activeRentals {
		if rental.Status == "active" && rental.EndTime.Before(now) {
			// Check if automatic renewal is enabled
			if rental.AutoRenewalEnabled && rental.RenewalPaymentTxHash != "" {
				// Attempt to renew the rental
				if drs.attemptAutomaticRenewal(rental) {
					renewedRentals = append(renewedRentals, id)
					log.Printf("Successfully auto-renewed rental %s", id)
					continue
				} else {
					log.Printf("Failed to auto-renew rental %s, expiring", id)
				}
			}

			// Rental has expired and cannot be renewed
			rental.Status = "expired"
			rental.UpdatedAt = now

			// Clean up CDE environment
			if err := drs.cleanupCDEEnvironment(rental.CDEEnvironmentID); err != nil {
				log.Printf("Warning: Failed to cleanup CDE environment for expired rental %s: %v", id, err)
			}

			expiredRentals = append(expiredRentals, id)

			// Save expired rental to database
			if err := drs.saveRentalToDatabase(rental); err != nil {
				log.Printf("Warning: Failed to save expired rental to database: %v", err)
			}
		}
	}

	// Remove expired rentals from active list
	for _, id := range expiredRentals {
		delete(drs.activeRentals, id)
	}

	if len(expiredRentals) > 0 {
		log.Printf("Cleaned up %d expired rentals", len(expiredRentals))
	}

	if len(renewedRentals) > 0 {
		log.Printf("Auto-renewed %d rentals", len(renewedRentals))
	}
}

// usageTrackingRoutine periodically tracks resource usage for active rentals
func (drs *DVERentalService) usageTrackingRoutine() {
	ticker := time.NewTicker(1 * time.Minute) // Track usage every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !drs.running {
				return
			}
			drs.trackUsageForActiveRentals()
		}
	}
}

// trackUsageForActiveRentals updates usage metrics for all active rentals
func (drs *DVERentalService) trackUsageForActiveRentals() {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	now := time.Now()

	for _, rental := range drs.activeRentals {
		if rental.Status == "active" {
			// Simulate usage tracking (in real implementation, this would query actual metrics)
			drs.updateRentalUsageMetrics(rental, now)

			// Save updated rental to database
			if err := drs.saveRentalToDatabase(rental); err != nil {
				log.Printf("Warning: Failed to save usage metrics for rental %s: %v", rental.ID, err)
			}
		}
	}
}

// updateRentalUsageMetrics updates the usage metrics for a rental
func (drs *DVERentalService) updateRentalUsageMetrics(rental *objects.DVERental, timestamp time.Time) {
	// Calculate time elapsed since last update
	timeElapsed := timestamp.Sub(rental.UsageMetrics.LastUpdated).Minutes()

	if timeElapsed <= 0 {
		return // No time has passed, skip update
	}

	// Simulate resource usage accumulation (in real implementation, query actual usage)
	// CPU usage: simulate 20-80% utilization
	rental.UsageMetrics.CPUUsage += float64(20+(timestamp.Unix()%60)) / 100.0
	if rental.UsageMetrics.CPUUsage > 1.0 {
		rental.UsageMetrics.CPUUsage = 1.0
	}

	// Memory usage: simulate 512MB to 2GB usage
	rental.UsageMetrics.MemoryUsage += int64(10 * 1024 * 1024) // Add 10MB per minute
	maxMemory := rental.ResourceLimits.MaxMemory
	if rental.UsageMetrics.MemoryUsage > maxMemory {
		rental.UsageMetrics.MemoryUsage = maxMemory
	}

	// Disk usage: simulate gradual increase
	rental.UsageMetrics.DiskUsage += int64(50 * 1024 * 1024) // Add 50MB per minute
	maxDisk := rental.ResourceLimits.MaxDisk
	if rental.UsageMetrics.DiskUsage > maxDisk {
		rental.UsageMetrics.DiskUsage = maxDisk
	}

	// Network usage: simulate data transfer
	rental.UsageMetrics.NetworkUsage += int64(5 * 1024 * 1024) // Add 5MB per minute

	// Update timestamp
	rental.UsageMetrics.LastUpdated = timestamp
	rental.UpdatedAt = timestamp

	// Calculate billing based on usage (optional - could be used for overage charges)
	drs.calculateUsageBasedBilling(rental)
}

// calculateUsageBasedBilling calculates any additional charges based on resource usage
func (drs *DVERentalService) calculateUsageBasedBilling(rental *objects.DVERental) {
	// This is a placeholder for usage-based billing logic
	// In a real implementation, this could calculate overage charges for:
	// - CPU usage above allocated limits
	// - Memory usage above allocated limits
	// - Network bandwidth usage
	// - Storage usage above allocated limits

	// For now, just log usage for monitoring
	if rental.UsageMetrics.CPUUsage > 0.9 {
		log.Printf("High CPU usage detected for rental %s: %.1f%%", rental.ID, rental.UsageMetrics.CPUUsage*100)
	}

	if float64(rental.UsageMetrics.MemoryUsage) > float64(rental.ResourceLimits.MaxMemory)*0.9 {
		log.Printf("High memory usage detected for rental %s: %d/%d bytes",
			rental.ID, rental.UsageMetrics.MemoryUsage, rental.ResourceLimits.MaxMemory)
	}
}

// attemptAutomaticRenewal attempts to automatically renew a rental using the renewal payment
func (drs *DVERentalService) attemptAutomaticRenewal(rental *objects.DVERental) bool {
	if drs.blockchainClient == nil {
		log.Printf("Cannot auto-renew rental %s: blockchain client not configured", rental.ID)
		return false
	}

	// Verify the renewal payment transaction
	systemWalletAddress := "knirv1system" // This should be configurable
	expectedAmount := rental.NRNAmount    // Same amount as original rental

	payment, err := drs.blockchainClient.VerifyPaymentTransaction(rental.RenewalPaymentTxHash, expectedAmount, systemWalletAddress)
	if err != nil {
		log.Printf("Renewal payment verification failed for rental %s: %v", rental.ID, err)
		return false
	}

	if payment.Status != "confirmed" {
		log.Printf("Renewal payment not confirmed for rental %s", rental.ID)
		return false
	}

	// Payment verified, extend the rental
	rental.EndTime = rental.EndTime.Add(time.Duration(rental.RentalDuration) * time.Second)
	rental.NRNAmount += expectedAmount // Add renewal payment to total
	rental.UpdatedAt = time.Now()

	// Clear the renewal payment hash (it was used)
	rental.RenewalPaymentTxHash = ""

	log.Printf("Successfully renewed rental %s for additional %d seconds, new end time: %s",
		rental.ID, rental.RentalDuration, rental.EndTime.Format(time.RFC3339))

	return true
}
