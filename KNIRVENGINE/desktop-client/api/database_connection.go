package api

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	// SQLite support removed - using chromem-go for local storage
)

// DatabaseConnection implements TargetSystemConnection for database access
type DatabaseConnection struct {
	target    *TargetSystem
	db        *sql.DB
	dbType    string
	connected bool
	startTime time.Time
	mutex     sync.RWMutex
}

// NewDatabaseConnection creates a new database connection
func NewDatabaseConnection(target *TargetSystem) (TargetSystemConnection, error) {
	conn := &DatabaseConnection{
		target: target,
	}

	// Get database type from config
	if dbType, ok := target.Config["dbType"].(string); ok {
		conn.dbType = dbType
	} else {
		conn.dbType = "sqlite3" // Default
	}

	return conn, nil
}

// Connect establishes the database connection
func (c *DatabaseConnection) Connect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		return nil
	}

	// Build connection string based on database type
	var dsn string
	var err error

	switch c.dbType {
	case "sqlite3":
		dsn = c.getSQLiteDSN()
	case "postgres", "postgresql":
		dsn = c.getPostgresDSN()
	case "mysql":
		dsn = c.getMySQLDSN()
	default:
		return fmt.Errorf("unsupported database type: %s", c.dbType)
	}

	// Open database connection
	c.db, err = sql.Open(c.dbType, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// Test connection
	if err := c.db.PingContext(ctx); err != nil {
		c.db.Close()
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Configure connection pool
	c.db.SetMaxOpenConns(10)
	c.db.SetMaxIdleConns(5)
	c.db.SetConnMaxLifetime(time.Hour)

	c.connected = true
	c.startTime = time.Now()

	return nil
}

// Disconnect closes the database connection
func (c *DatabaseConnection) Disconnect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected {
		return nil
	}

	if c.db != nil {
		c.db.Close()
	}

	c.connected = false
	return nil
}

// IsConnected returns the connection status
func (c *DatabaseConnection) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// GetCapabilities returns available database capabilities
func (c *DatabaseConnection) GetCapabilities() []string {
	return []string{
		"query",
		"execute",
		"transaction",
		"list_tables",
		"describe_table",
		"get_schema",
		"backup",
		"restore",
	}
}

// Execute executes a database operation
func (c *DatabaseConnection) Execute(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	switch operation {
	case "query":
		return c.query(ctx, params)
	case "execute":
		return c.execute(ctx, params)
	case "transaction":
		return c.transaction(ctx, params)
	case "list_tables":
		return c.listTables(ctx, params)
	case "describe_table":
		return c.describeTable(ctx, params)
	case "get_schema":
		return c.getSchema(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// GetStatus returns detailed database status
func (c *DatabaseConnection) GetStatus() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	status := map[string]interface{}{
		"connected": c.connected,
		"type":      "database",
		"dbType":    c.dbType,
		"uptime":    time.Since(c.startTime).String(),
	}

	if c.connected && c.db != nil {
		stats := c.db.Stats()
		status["openConnections"] = stats.OpenConnections
		status["inUse"] = stats.InUse
		status["idle"] = stats.Idle
	}

	return status
}

// GetType returns the target system type
func (c *DatabaseConnection) GetType() TargetSystemType {
	return TargetTypeDatabase
}

// Helper methods for building DSNs

func (c *DatabaseConnection) getSQLiteDSN() string {
	if path, ok := c.target.Config["path"].(string); ok {
		return path
	}
	return ":memory:" // Default to in-memory database
}

func (c *DatabaseConnection) getPostgresDSN() string {
	host := getStringFromConfig(c.target.Config, "host", "localhost")
	port := getStringFromConfig(c.target.Config, "port", "5432")
	user := getStringFromConfig(c.target.Config, "user", "postgres")
	password := getStringFromConfig(c.target.Config, "password", "")
	dbname := getStringFromConfig(c.target.Config, "dbname", "postgres")
	sslmode := getStringFromConfig(c.target.Config, "sslmode", "disable")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

func (c *DatabaseConnection) getMySQLDSN() string {
	host := getStringFromConfig(c.target.Config, "host", "localhost")
	port := getStringFromConfig(c.target.Config, "port", "3306")
	user := getStringFromConfig(c.target.Config, "user", "root")
	password := getStringFromConfig(c.target.Config, "password", "")
	dbname := getStringFromConfig(c.target.Config, "dbname", "mysql")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, dbname)
}

// Database operations

func (c *DatabaseConnection) query(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query := getStringParam(params, "query", "")
	if query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %v", err)
	}

	// Prepare result
	var results []map[string]interface{}

	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// Create result map
		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"rows":    results,
		"count":   len(results),
		"columns": columns,
	}, nil
}

func (c *DatabaseConnection) execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query := getStringParam(params, "query", "")
	if query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	result, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("execute failed: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	return map[string]interface{}{
		"success":      true,
		"rowsAffected": rowsAffected,
		"lastInsertId": lastInsertId,
	}, nil
}

func (c *DatabaseConnection) transaction(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	queries, ok := params["queries"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("queries parameter is required and must be an array")
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}

	var results []map[string]interface{}

	for i, queryInterface := range queries {
		query, ok := queryInterface.(string)
		if !ok {
			tx.Rollback()
			return nil, fmt.Errorf("query %d is not a string", i)
		}

		result, err := tx.ExecContext(ctx, query)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("query %d failed: %v", i, err)
		}

		rowsAffected, _ := result.RowsAffected()
		lastInsertId, _ := result.LastInsertId()

		results = append(results, map[string]interface{}{
			"query":        query,
			"rowsAffected": rowsAffected,
			"lastInsertId": lastInsertId,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"results": results,
		"count":   len(results),
	}, nil
}

func (c *DatabaseConnection) listTables(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = params // Parameters not used in current implementation
	var query string

	switch c.dbType {
	case "sqlite3":
		query = "SELECT name FROM sqlite_master WHERE type='table'"
	case "postgres", "postgresql":
		query = "SELECT tablename FROM pg_tables WHERE schemaname = 'public'"
	case "mysql":
		query = "SHOW TABLES"
	default:
		return nil, fmt.Errorf("list_tables not supported for database type: %s", c.dbType)
	}

	return c.query(ctx, map[string]interface{}{"query": query})
}

func (c *DatabaseConnection) describeTable(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	tableName := getStringParam(params, "table", "")
	if tableName == "" {
		return nil, fmt.Errorf("table parameter is required")
	}

	var query string

	switch c.dbType {
	case "sqlite3":
		query = fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	case "postgres", "postgresql":
		query = fmt.Sprintf("SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = '%s'", tableName)
	case "mysql":
		query = fmt.Sprintf("DESCRIBE %s", tableName)
	default:
		return nil, fmt.Errorf("describe_table not supported for database type: %s", c.dbType)
	}

	return c.query(ctx, map[string]interface{}{"query": query})
}

func (c *DatabaseConnection) getSchema(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Get all tables first
	tablesResult, err := c.listTables(ctx, params)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"schema":  tablesResult,
		"dbType":  c.dbType,
	}, nil
}
