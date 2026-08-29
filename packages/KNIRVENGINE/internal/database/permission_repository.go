package database

import (
	"context"
	"database/sql"
	"fmt"
)

// Permission represents a permission in the system
type Permission struct {
	ID          int64  `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
}

// PermissionRepository handles database operations for permissions
type PermissionRepository struct {
	db *sql.DB
}

// NewPermissionRepository creates a new permission repository
func NewPermissionRepository(db *sql.DB) *PermissionRepository {
	return &PermissionRepository{
		db: db,
	}
}

// GetPermissionByID retrieves a permission by ID
func (r *PermissionRepository) GetPermissionByID(ctx context.Context, id int64) (*Permission, error) {
	query := `SELECT id, name, description FROM permissions WHERE id = ?`

	var permission Permission
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&permission.ID,
		&permission.Name,
		&permission.Description,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("permission not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return &permission, nil
}

// GetPermissionByName retrieves a permission by name
func (r *PermissionRepository) GetPermissionByName(ctx context.Context, name string) (*Permission, error) {
	query := `SELECT id, name, description FROM permissions WHERE name = ?`

	var permission Permission
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&permission.ID,
		&permission.Name,
		&permission.Description,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("permission not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return &permission, nil
}

// ListPermissions retrieves all permissions
func (r *PermissionRepository) ListPermissions(ctx context.Context) ([]*Permission, error) {
	query := `SELECT id, name, description FROM permissions`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*Permission
	for rows.Next() {
		var permission Permission
		err := rows.Scan(
			&permission.ID,
			&permission.Name,
			&permission.Description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, &permission)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating permission rows: %w", err)
	}

	return permissions, nil
}

// CreatePermission creates a new permission
func (r *PermissionRepository) CreatePermission(ctx context.Context, permission *Permission) error {
	query := `
		INSERT INTO permissions (name, description)
		VALUES (?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, permission.Name, permission.Description)
	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get permission ID: %w", err)
	}

	permission.ID = id
	return nil
}

// UpdatePermission updates an existing permission
func (r *PermissionRepository) UpdatePermission(ctx context.Context, permission *Permission) error {
	query := `
		UPDATE permissions
		SET name = ?, description = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query, permission.Name, permission.Description, permission.ID)
	if err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	return nil
}

// DeletePermission deletes a permission
func (r *PermissionRepository) DeletePermission(ctx context.Context, id int64) error {
	query := `DELETE FROM permissions WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	return nil
}

// GetPermissionsForRole retrieves all permissions for a role
func (r *PermissionRepository) GetPermissionsForRole(ctx context.Context, roleName string) ([]*Permission, error) {
	query := `
		SELECT p.id, p.name, p.description
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_name = ?
	`

	rows, err := r.db.QueryContext(ctx, query, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to query permissions for role: %w", err)
	}
	defer rows.Close()

	var permissions []*Permission
	for rows.Next() {
		var permission Permission
		err := rows.Scan(
			&permission.ID,
			&permission.Name,
			&permission.Description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, &permission)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating permission rows: %w", err)
	}

	return permissions, nil
}

// GetPermissionsForUser retrieves all permissions for a user based on their role
func (r *PermissionRepository) GetPermissionsForUser(ctx context.Context, userID int64) ([]*Permission, error) {
	query := `
		SELECT p.id, p.name, p.description
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN users u ON rp.role_name = u.role
		WHERE u.id = ?
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query permissions for user: %w", err)
	}
	defer rows.Close()

	var permissions []*Permission
	for rows.Next() {
		var permission Permission
		err := rows.Scan(
			&permission.ID,
			&permission.Name,
			&permission.Description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, &permission)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating permission rows: %w", err)
	}

	return permissions, nil
}

// HasPermission checks if a user has a specific permission
func (r *PermissionRepository) HasPermission(ctx context.Context, userID int64, permissionName string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM users u
		JOIN role_permissions rp ON u.role = rp.role_name
		JOIN permissions p ON rp.permission_id = p.id
		WHERE u.id = ? AND p.name = ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, permissionName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return count > 0, nil
}
