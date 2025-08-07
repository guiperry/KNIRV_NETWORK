package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tidwall/buntdb"
)

// BuntDBManager manages BuntDB operations with custom indexes
type BuntDBManager struct {
	db *buntdb.DB
}

// NewBuntDB creates a new BuntDB instance with custom indexes
func NewBuntDB(path string) (*BuntDBManager, error) {
	db, err := buntdb.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open BuntDB: %w", err)
	}

	manager := &BuntDBManager{db: db}

	// Create custom indexes
	if err := manager.createIndexes(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return manager, nil
}

// createIndexes creates all necessary indexes for DVE operations
func (bm *BuntDBManager) createIndexes() error {
	return bm.db.Update(func(tx *buntdb.Tx) error {
		// DVE Node indexes
		tx.CreateIndex("nodes_by_status", "dve:nodes:*", buntdb.IndexJSON("status"))
		tx.CreateIndex("nodes_by_tee_type", "dve:nodes:*", buntdb.IndexJSON("tee_type"))
		tx.CreateIndex("nodes_by_reputation", "dve:nodes:*", buntdb.IndexJSON("reputation_score"))
		tx.CreateIndex("nodes_by_stake", "dve:nodes:*", buntdb.IndexJSON("stake_amount"))
		tx.CreateIndex("nodes_by_location", "dve:nodes:*", buntdb.IndexJSON("location"))
		tx.CreateIndex("nodes_by_created_at", "dve:nodes:*", buntdb.IndexJSON("created_at"))

		// Validation Task indexes
		tx.CreateIndex("tasks_by_status", "validation:tasks:*", buntdb.IndexJSON("status"))
		tx.CreateIndex("tasks_by_priority", "validation:tasks:*", buntdb.IndexJSON("priority"))
		tx.CreateIndex("tasks_by_type", "validation:tasks:*", buntdb.IndexJSON("type"))
		tx.CreateIndex("tasks_by_created_at", "validation:tasks:*", buntdb.IndexJSON("created_at"))
		tx.CreateIndex("tasks_by_assigned_node", "validation:tasks:*", buntdb.IndexJSON("assigned_node_id"))

		// Validation Proof indexes
		tx.CreateIndex("proofs_by_task_id", "validation:proofs:*", buntdb.IndexJSON("task_id"))
		tx.CreateIndex("proofs_by_validator", "validation:proofs:*", buntdb.IndexJSON("validator_node_id"))
		tx.CreateIndex("proofs_by_status", "validation:proofs:*", buntdb.IndexJSON("status"))

		// TEE Attestation indexes
		tx.CreateIndex("attestations_by_node", "tee:attestations:*", buntdb.IndexJSON("node_id"))
		tx.CreateIndex("attestations_by_type", "tee:attestations:*", buntdb.IndexJSON("tee_type"))
		tx.CreateIndex("attestations_by_status", "tee:attestations:*", buntdb.IndexJSON("status"))

		// User Management indexes
		tx.CreateIndex("users_by_email", "users:profiles:*", buntdb.IndexJSON("email"))
		tx.CreateIndex("users_by_username", "users:profiles:*", buntdb.IndexJSON("username"))
		tx.CreateIndex("users_by_role", "users:profiles:*", buntdb.IndexJSON("role"))
		tx.CreateIndex("users_by_created_at", "users:profiles:*", buntdb.IndexJSON("created_at"))

		// Session indexes
		tx.CreateIndex("sessions_by_user_id", "users:sessions:*", buntdb.IndexJSON("user_id"))
		tx.CreateIndex("sessions_by_expires_at", "users:sessions:*", buntdb.IndexJSON("expires_at"))

		// Report indexes
		tx.CreateIndex("reports_by_type", "reports:*", buntdb.IndexJSON("type"))
		tx.CreateIndex("reports_by_created_at", "reports:*", buntdb.IndexJSON("created_at"))
		tx.CreateIndex("reports_by_user", "reports:*", buntdb.IndexJSON("generated_by"))
		tx.CreateIndex("reports_by_format", "reports:*", buntdb.IndexJSON("format"))

		// Report Template indexes
		tx.CreateIndex("templates_by_type", "report:templates:*", buntdb.IndexJSON("type"))
		tx.CreateIndex("templates_by_created_by", "report:templates:*", buntdb.IndexJSON("created_by"))
		tx.CreateIndex("templates_by_public", "report:templates:*", buntdb.IndexJSON("is_public"))

		// Metrics indexes
		tx.CreateIndex("metrics_by_timestamp", "metrics:historical:*", buntdb.IndexJSON("timestamp"))
		tx.CreateIndex("metrics_by_type", "metrics:historical:*", buntdb.IndexJSON("type"))
		tx.CreateIndex("metrics_by_node_id", "metrics:historical:*", buntdb.IndexJSON("node_id"))

		// Spatial index for geographic node distribution
		tx.CreateSpatialIndex("nodes_by_coordinates", "dve:nodes:*", buntdb.IndexRect)

		return nil
	})
}

// GetDB returns the underlying BuntDB instance
func (bm *BuntDBManager) GetDB() *buntdb.DB {
	return bm.db
}

// Close closes the database connection
func (bm *BuntDBManager) Close() error {
	return bm.db.Close()
}

// StoreJSON stores a JSON object with the given key
func (bm *BuntDBManager) StoreJSON(key string, data interface{}) error {
	return bm.db.Update(func(tx *buntdb.Tx) error {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(key, string(jsonData), nil)
		return err
	})
}

// GetJSON retrieves and unmarshals a JSON object
func (bm *BuntDBManager) GetJSON(key string, dest interface{}) error {
	return bm.db.View(func(tx *buntdb.Tx) error {
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), dest)
	})
}

// DeleteKey deletes a key from the database
func (bm *BuntDBManager) DeleteKey(key string) error {
	return bm.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(key)
		return err
	})
}

// ListKeys returns all keys matching a pattern
func (bm *BuntDBManager) ListKeys(pattern string) ([]string, error) {
	var keys []string
	err := bm.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend(pattern, func(key, value string) bool {
			keys = append(keys, key)
			return true
		})
	})
	return keys, err
}

// CountKeys returns the count of keys matching a pattern
func (bm *BuntDBManager) CountKeys(pattern string) (int, error) {
	count := 0
	err := bm.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend(pattern, func(key, value string) bool {
			count++
			return true
		})
	})
	return count, err
}

// SetWithTTL stores a key-value pair with a time-to-live
func (bm *BuntDBManager) SetWithTTL(key, value string, ttl time.Duration) error {
	return bm.db.Update(func(tx *buntdb.Tx) error {
		opts := &buntdb.SetOptions{Expires: true, TTL: ttl}
		_, _, err := tx.Set(key, value, opts)
		return err
	})
}

// Transaction executes a function within a database transaction
func (bm *BuntDBManager) Transaction(fn func(*buntdb.Tx) error) error {
	return bm.db.Update(fn)
}

// ViewTransaction executes a read-only function within a database transaction
func (bm *BuntDBManager) ViewTransaction(fn func(*buntdb.Tx) error) error {
	return bm.db.View(fn)
}
