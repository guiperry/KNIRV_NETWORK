package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Role represents a role in the system
type Role struct {
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// RoleRepository handles database operations for roles
type RoleRepository struct {
	db *sql.DB
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *sql.DB) *RoleRepository {
	return &RoleRepository{
		db: db,
	}
}

// GetRoleByName retrieves a role by name
func (r *RoleRepository) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	query := `SELECT name, description, created_at FROM roles WHERE name = ?`

	var role Role
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&role.Name,
		&role.Description,
		&role.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

// ListRoles retrieves all roles
func (r *RoleRepository) ListRoles(ctx context.Context) ([]*Role, error) {
	query := `SELECT name, description, created_at FROM roles`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		var role Role
		err := rows.Scan(
			&role.Name,
			&role.Description,
			&role.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, &role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating role rows: %w", err)
	}

	return roles, nil
}

// CreateRole creates a new role
func (r *RoleRepository) CreateRole(ctx context.Context, role *Role) error {
	// Set creation timestamp
	role.CreatedAt = time.Now()

	query := `
		INSERT INTO roles (name, description, created_at)
		VALUES (?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, role.Name, role.Description, role.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// UpdateRole updates an existing role
func (r *RoleRepository) UpdateRole(ctx context.Context, role *Role) error {
	query := `
		UPDATE roles
		SET description = ?
		WHERE name = ?
	`

	_, err := r.db.ExecContext(ctx, query, role.Description, role.Name)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

// DeleteRole deletes a role
func (r *RoleRepository) DeleteRole(ctx context.Context, name string) error {
	query := `DELETE FROM roles WHERE name = ?`

	_, err := r.db.ExecContext(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}

// AssignPermissionToRole assigns a permission to a role
func (r *RoleRepository) AssignPermissionToRole(ctx context.Context, roleName string, permissionID int64) error {
	query := `
		INSERT OR IGNORE INTO role_permissions (role_name, permission_id)
		VALUES (?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, roleName, permissionID)
	if err != nil {
		return fmt.Errorf("failed to assign permission to role: %w", err)
	}

	return nil
}

// RemovePermissionFromRole removes a permission from a role
func (r *RoleRepository) RemovePermissionFromRole(ctx context.Context, roleName string, permissionID int64) error {
	query := `
		DELETE FROM role_permissions
		WHERE role_name = ? AND permission_id = ?
	`

	_, err := r.db.ExecContext(ctx, query, roleName, permissionID)
	if err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}

	return nil
}

// GetRolePermissions retrieves all permissions for a role
func (r *RoleRepository) GetRolePermissions(ctx context.Context, roleName string) ([]int64, error) {
	query := `
		SELECT permission_id
		FROM role_permissions
		WHERE role_name = ?
	`

	rows, err := r.db.QueryContext(ctx, query, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to query role permissions: %w", err)
	}
	defer rows.Close()

	var permissionIDs []int64
	for rows.Next() {
		var id int64
		err := rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission ID: %w", err)
		}
		permissionIDs = append(permissionIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating permission ID rows: %w", err)
	}

	return permissionIDs, nil
}
