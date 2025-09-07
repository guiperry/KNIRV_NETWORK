package dverental

import (
	"encoding/json"
	"log"
	"time"

	"nexus-backend/internal/models"
	"github.com/tidwall/buntdb"
)

// Database operations

// loadFromDatabase loads rental data from the database
func (drs *DVERentalService) loadFromDatabase() error {
	return drs.db.View(func(tx *buntdb.Tx) error {
		// Load active rentals
		tx.Ascend("", func(key, value string) bool {
			if len(key) > 8 && key[:8] == "rental:" {
				var rental models.DVERental
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
				var plan models.RentalPlan
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
func (drs *DVERentalService) saveRentalToDatabase(rental *models.DVERental) error {
	return drs.db.Update(func(tx *buntdb.Tx) error {
		if data, err := json.Marshal(rental); err == nil {
			tx.Set("rental:"+rental.ID, string(data), nil)
		}
		return nil
	})
}

// Rental plan management

// createDefaultRentalPlans creates the default rental plans
func createDefaultRentalPlans() []*models.RentalPlan {
	return []*models.RentalPlan{
		{
			ID:          "basic",
			Name:        "Basic DVE",
			Description: "Basic DVE rental with limited resources",
			PricePerHour: 10, // 10 NRN per hour
			ResourceLimits: models.ResourceLimits{
				MaxCPU:       1.0,
				MaxMemory:    1024 * 1024 * 1024, // 1GB
				MaxDisk:      5 * 1024 * 1024 * 1024, // 5GB
				MaxBandwidth: 100 * 1024 * 1024, // 100MB/s
			},
			MaxDuration: 24 * 60 * 60, // 24 hours
			MinDuration: 60 * 60,      // 1 hour
			Features:    []string{"Basic CDE", "SSH Access", "Web Terminal"},
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "standard",
			Name:        "Standard DVE",
			Description: "Standard DVE rental with moderate resources",
			PricePerHour: 25, // 25 NRN per hour
			ResourceLimits: models.ResourceLimits{
				MaxCPU:       2.0,
				MaxMemory:    4 * 1024 * 1024 * 1024, // 4GB
				MaxDisk:      20 * 1024 * 1024 * 1024, // 20GB
				MaxBandwidth: 500 * 1024 * 1024, // 500MB/s
			},
			MaxDuration: 7 * 24 * 60 * 60, // 7 days
			MinDuration: 60 * 60,          // 1 hour
			Features:    []string{"Enhanced CDE", "SSH Access", "Web Terminal", "GPU Access", "Custom Images"},
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "premium",
			Name:        "Premium DVE",
			Description: "Premium DVE rental with high-performance resources",
			PricePerHour: 50, // 50 NRN per hour
			ResourceLimits: models.ResourceLimits{
				MaxCPU:       8.0,
				MaxMemory:    16 * 1024 * 1024 * 1024, // 16GB
				MaxDisk:      100 * 1024 * 1024 * 1024, // 100GB
				MaxBandwidth: 1024 * 1024 * 1024, // 1GB/s
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
	for _, plan := range drs.defaultPlans {
		drs.rentalPlans[plan.ID] = plan
	}
	
	// Save to database
	if err := drs.saveToDatabase(); err != nil {
		log.Printf("Warning: Failed to save default rental plans to database: %v", err)
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

// cleanupExpiredRentals removes expired rentals
func (drs *DVERentalService) cleanupExpiredRentals() {
	drs.mu.Lock()
	defer drs.mu.Unlock()
	
	now := time.Now()
	expiredRentals := make([]string, 0)
	
	for id, rental := range drs.activeRentals {
		if rental.Status == "active" && rental.EndTime.Before(now) {
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
}
