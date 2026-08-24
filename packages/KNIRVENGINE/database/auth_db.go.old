package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// AuthDB manages the authentication database
type AuthDB struct {
	db *sql.DB
}

// NewAuthDB creates a new authentication database connection
func NewAuthDB(dbPath string) (*AuthDB, error) {
	// Ensure the directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create auth database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open auth database: %w", err)
	}

	// Initialize schema if needed
	if err := initAuthSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &AuthDB{db: db}, nil
}

// GetDB returns the underlying database connection
func (a *AuthDB) GetDB() *sql.DB {
	return a.db
}

// Close closes the database connection
func (a *AuthDB) Close() error {
	return a.db.Close()
}

// initAuthSchema initializes the authentication database schema
func initAuthSchema(db *sql.DB) error {
	// Create users table
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		role TEXT NOT NULL DEFAULT 'user',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	// Create user_passwords table
	passwordsTable := `
	CREATE TABLE IF NOT EXISTS user_passwords (
		user_id INTEGER PRIMARY KEY,
		password_hash TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	// Create roles table
	rolesTable := `
	CREATE TABLE IF NOT EXISTS roles (
		name TEXT PRIMARY KEY,
		description TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	// Create permissions table
	permissionsTable := `
	CREATE TABLE IF NOT EXISTS permissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		description TEXT NOT NULL
	);`

	// Create role_permissions junction table
	rolePermissionsTable := `
	CREATE TABLE IF NOT EXISTS role_permissions (
		role_name TEXT NOT NULL,
		permission_id INTEGER NOT NULL,
		PRIMARY KEY (role_name, permission_id),
		FOREIGN KEY (role_name) REFERENCES roles(name) ON DELETE CASCADE,
		FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
	);`

	// Create API tokens table
	apiTokensTable := `
	CREATE TABLE IF NOT EXISTS api_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL UNIQUE,
		description TEXT,
		expires_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	// Execute all schema creation statements
	for _, schema := range []string{
		usersTable,
		passwordsTable,
		rolesTable,
		permissionsTable,
		rolePermissionsTable,
		apiTokensTable,
	} {
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
	}

	// Insert default roles if they don't exist
	defaultRoles := []struct {
		name        string
		description string
	}{
		{"admin", "Administrator with full access"},
		{"user", "Standard user with limited access"},
	}

	for _, role := range defaultRoles {
		_, err := db.Exec(
			"INSERT OR IGNORE INTO roles (name, description) VALUES (?, ?)",
			role.name, role.description,
		)
		if err != nil {
			return fmt.Errorf("failed to insert default role %s: %w", role.name, err)
		}
	}

	// Insert default permissions if they don't exist
	defaultPermissions := []struct {
		name        string
		description string
	}{
		{"agent:read", "Read agent information"},
		{"agent:create", "Create new agents"},
		{"agent:update", "Update existing agents"},
		{"agent:delete", "Delete agents"},
		{"target:read", "Read target information"},
		{"target:create", "Create new targets"},
		{"target:update", "Update existing targets"},
		{"target:delete", "Delete targets"},
		{"capability:read", "Read capability information"},
		{"capability:create", "Create new capabilities"},
		{"capability:update", "Update existing capabilities"},
		{"capability:delete", "Delete capabilities"},
		{"workflow:read", "Read workflow information"},
		{"workflow:execute", "Execute workflows"},
		{"workflow:cancel", "Cancel workflows"},
		{"analytics:read", "View analytics data"},
		{"settings:read", "Read settings"},
		{"settings:update", "Update settings"},
		{"user:read", "Read user information"},
		{"user:create", "Create new users"},
		{"user:update", "Update existing users"},
		{"user:delete", "Delete users"},
	}

	for _, perm := range defaultPermissions {
		_, err := db.Exec(
			"INSERT OR IGNORE INTO permissions (name, description) VALUES (?, ?)",
			perm.name, perm.description,
		)
		if err != nil {
			return fmt.Errorf("failed to insert default permission %s: %w", perm.name, err)
		}
	}

	// Assign all permissions to admin role
	var permIDs []int
	rows, err := db.Query("SELECT id FROM permissions")
	if err != nil {
		return fmt.Errorf("failed to query permissions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan permission ID: %w", err)
		}
		permIDs = append(permIDs, id)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating permission rows: %w", err)
	}

	// Assign all permissions to admin role
	for _, permID := range permIDs {
		_, err := db.Exec(
			"INSERT OR IGNORE INTO role_permissions (role_name, permission_id) VALUES (?, ?)",
			"admin", permID,
		)
		if err != nil {
			return fmt.Errorf("failed to assign permission %d to admin role: %w", permID, err)
		}
	}

	// Assign limited permissions to user role
	userPermissions := []string{
		"agent:read", "target:read", "capability:read", "workflow:read",
		"workflow:execute", "analytics:read", "settings:read",
	}

	for _, permName := range userPermissions {
		var permID int
		err := db.QueryRow("SELECT id FROM permissions WHERE name = ?", permName).Scan(&permID)
		if err != nil {
			return fmt.Errorf("failed to get permission ID for %s: %w", permName, err)
		}

		_, err = db.Exec(
			"INSERT OR IGNORE INTO role_permissions (role_name, permission_id) VALUES (?, ?)",
			"user", permID,
		)
		if err != nil {
			return fmt.Errorf("failed to assign permission %s to user role: %w", permName, err)
		}
	}

	// Create default admin user if no users exist
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if userCount == 0 {
		log.Println("Creating default admin user...")

		// Create a user repository
		userRepo := NewUserRepository(db)

		// Create default admin user
		adminUser := &User{
			Username: "admin",
			Email:    "admin@example.com",
			Role:     "admin",
		}

		if err := userRepo.CreateUser(adminUser, "admin123"); err != nil {
			return fmt.Errorf("failed to create default admin user: %w", err)
		}

		log.Println("Default admin user created successfully.")
	}

	return nil
}
