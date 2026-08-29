package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/philippgille/chromem-go"
	"golang.org/x/crypto/bcrypt"
)

// ChromemAuthDB manages authentication using chromem-go
type ChromemAuthDB struct {
	db               *chromem.DB
	usersCollection  *chromem.Collection
	rolesCollection  *chromem.Collection
	tokensCollection *chromem.Collection
}

// UserDocument represents a user document in chromem-go
type UserDocument struct {
	ID           string                 `json:"id"`
	Username     string                 `json:"username"`
	Email        string                 `json:"email"`
	PasswordHash string                 `json:"password_hash"`
	Role         string                 `json:"role"`
	Permissions  []string               `json:"permissions"`
	Sessions     []SessionInfo          `json:"sessions"`
	APITokens    []string               `json:"api_tokens"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// RoleDocument represents a role document in chromem-go
type RoleDocument struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// TokenDocument represents an API token document in chromem-go
type TokenDocument struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Token       string     `json:"token"`
	Description string     `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// SessionInfo represents session information
type SessionInfo struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

// NewChromemAuthDB creates a new chromem-go authentication database
func NewChromemAuthDB(dbPath string) (*ChromemAuthDB, error) {
	// Create or open the persistent database
	db, err := chromem.NewPersistentDB(dbPath, true) // true for compression
	if err != nil {
		return nil, fmt.Errorf("failed to create chromem-go auth database: %v", err)
	}

	// Create deterministic embedding function to avoid external API dependencies
	embeddingFunc := createDeterministicEmbeddingFunction()

	// Get or create collections
	usersCollection, err := db.GetOrCreateCollection("users", nil, embeddingFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create users collection: %v", err)
	}

	rolesCollection, err := db.GetOrCreateCollection("roles", nil, embeddingFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create roles collection: %v", err)
	}

	tokensCollection, err := db.GetOrCreateCollection("tokens", nil, embeddingFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokens collection: %v", err)
	}

	authDB := &ChromemAuthDB{
		db:               db,
		usersCollection:  usersCollection,
		rolesCollection:  rolesCollection,
		tokensCollection: tokensCollection,
	}

	// Initialize default data
	if err := authDB.initializeDefaultData(); err != nil {
		return nil, fmt.Errorf("failed to initialize default data: %v", err)
	}

	return authDB, nil
}

// createDeterministicEmbeddingFunction creates a simple deterministic embedding function
func createDeterministicEmbeddingFunction() chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		// Simple deterministic embedding based on text hash
		hash := 0
		for _, char := range text {
			hash = hash*31 + int(char)
		}

		// Create a 384-dimensional embedding (common size)
		embedding := make([]float32, 384)
		for i := range embedding {
			embedding[i] = float32((hash+i)%1000) / 1000.0
		}

		return embedding, nil
	}
}

// initializeDefaultData creates default roles and admin user if they don't exist
func (db *ChromemAuthDB) initializeDefaultData() error {
	ctx := context.Background()

	// Create default roles
	defaultRoles := []RoleDocument{
		{
			Name:        "admin",
			Description: "Administrator with full access",
			Permissions: []string{
				"agent:read", "agent:create", "agent:update", "agent:delete",
				"target:read", "target:create", "target:update", "target:delete",
				"capability:read", "capability:create", "capability:update", "capability:delete",
				"workflow:read", "workflow:execute", "workflow:cancel",
				"analytics:read", "settings:read", "settings:update",
				"user:read", "user:create", "user:update", "user:delete",
			},
			CreatedAt: time.Now(),
		},
		{
			Name:        "user",
			Description: "Standard user with limited access",
			Permissions: []string{
				"agent:read", "target:read", "capability:read",
				"workflow:read", "workflow:execute", "analytics:read", "settings:read",
			},
			CreatedAt: time.Now(),
		},
	}

	// Create roles if they don't exist
	for _, role := range defaultRoles {
		if err := db.createRoleIfNotExists(ctx, role); err != nil {
			return fmt.Errorf("failed to create default role %s: %v", role.Name, err)
		}
	}

	// Check if admin user exists
	_, err := db.GetUserByUsername(ctx, "admin")
	if err != nil {
		// Admin user doesn't exist, create it
		log.Println("Creating default admin user...")

		adminUser := &UserDocument{
			ID:          "admin-user-001",
			Username:    "admin",
			Email:       "admin@example.com",
			Role:        "admin",
			Permissions: defaultRoles[0].Permissions, // Admin permissions
			Sessions:    []SessionInfo{},
			APITokens:   []string{},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata:    make(map[string]interface{}),
		}

		if err := db.CreateUser(ctx, adminUser, "admin123"); err != nil {
			return fmt.Errorf("failed to create default admin user: %v", err)
		}

		log.Println("Default admin user created successfully.")
	}

	return nil
}

// createRoleIfNotExists creates a role if it doesn't already exist
func (db *ChromemAuthDB) createRoleIfNotExists(ctx context.Context, role RoleDocument) error {
	// Check if role exists
	_, err := db.rolesCollection.GetByID(ctx, role.Name)
	if err == nil {
		// Role already exists
		return nil
	}

	// Create role document
	roleJSON, err := json.Marshal(role)
	if err != nil {
		return fmt.Errorf("failed to marshal role: %v", err)
	}

	doc := chromem.Document{
		ID:      role.Name,
		Content: fmt.Sprintf("Role: %s - %s", role.Name, role.Description),
		Metadata: map[string]string{
			"type":        "role",
			"name":        role.Name,
			"description": role.Description,
			"created_at":  role.CreatedAt.Format(time.RFC3339),
			"data":        string(roleJSON),
		},
	}

	return db.rolesCollection.AddDocuments(ctx, []chromem.Document{doc}, 1)
}

// GetUserByUsername retrieves a user by username
func (db *ChromemAuthDB) GetUserByUsername(ctx context.Context, username string) (*UserDocument, error) {
	// Use progressive query fallback approach from query_fix.md
	// Try different search terms and limits to find all users
	searchTerms := []string{username, "User", "user", "admin"}
	limits := []int{5, 3, 1}

	for _, searchTerm := range searchTerms {
		for _, limit := range limits {
			results, err := db.usersCollection.Query(ctx, searchTerm, limit, nil, nil)
			if err == nil {
				// Success - now filter manually for exact username match
				log.Printf("Query succeeded with %d results for search term '%s', looking for username '%s'", len(results), searchTerm, username)
				for i, result := range results {
					log.Printf("Result %d: username='%s', type='%s'", i, result.Metadata["username"], result.Metadata["type"])
					if result.Metadata["username"] == username {
						// Get the user data from metadata
						userData := result.Metadata["data"]

						var user UserDocument
						if err := json.Unmarshal([]byte(userData), &user); err != nil {
							return nil, fmt.Errorf("failed to unmarshal user data: %v", err)
						}

						return &user, nil
					}
				}
				// If we found results but not the target user, continue searching
				if len(results) > 0 {
					continue
				}
			} else if strings.Contains(err.Error(), "nResults") {
				// Try smaller limit
				continue
			} else {
				// Other errors, try different search term
				break
			}
		}
	}

	return nil, fmt.Errorf("user not found: %s", username)
}

// GetUserByID retrieves a user by ID
func (db *ChromemAuthDB) GetUserByID(ctx context.Context, userID string) (*UserDocument, error) {
	result, err := db.usersCollection.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	// Get the user data from metadata
	userData := result.Metadata["data"]

	var user UserDocument
	if err := json.Unmarshal([]byte(userData), &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user data: %v", err)
	}

	return &user, nil
}

// CreateUser creates a new user with the given password
func (db *ChromemAuthDB) CreateUser(ctx context.Context, user *UserDocument, password string) error {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	user.PasswordHash = string(hashedPassword)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	// Get role permissions
	role, err := db.getRoleByName(ctx, user.Role)
	if err != nil {
		return fmt.Errorf("failed to get role permissions: %v", err)
	}
	user.Permissions = role.Permissions

	// Marshal user data
	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %v", err)
	}

	// Create document
	doc := chromem.Document{
		ID:      user.ID,
		Content: fmt.Sprintf("User: %s (%s) - %s", user.Username, user.Email, user.Role),
		Metadata: map[string]string{
			"type":       "user",
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"created_at": user.CreatedAt.Format(time.RFC3339),
			"updated_at": user.UpdatedAt.Format(time.RFC3339),
			"data":       string(userJSON),
		},
	}

	return db.usersCollection.AddDocuments(ctx, []chromem.Document{doc}, 1)
}

// VerifyPassword checks if the provided password matches the stored hash
func (db *ChromemAuthDB) VerifyPassword(ctx context.Context, username, password string) (bool, error) {
	user, err := db.GetUserByUsername(ctx, username)
	if err != nil {
		return false, nil // User not found, but not an error for security
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil, nil
}

// getRoleByName retrieves a role by name
func (db *ChromemAuthDB) getRoleByName(ctx context.Context, roleName string) (*RoleDocument, error) {
	result, err := db.rolesCollection.GetByID(ctx, roleName)
	if err != nil {
		return nil, fmt.Errorf("role not found: %s", roleName)
	}

	// Get the role data from metadata
	roleData := result.Metadata["data"]

	var role RoleDocument
	if err := json.Unmarshal([]byte(roleData), &role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal role data: %v", err)
	}

	return &role, nil
}

// Close closes the database connection
func (db *ChromemAuthDB) Close() error {
	// chromem-go handles persistence automatically
	return nil
}
