package inferencer

// DatabaseAccessor defines the interface for database operations
// that the InferenceService might need.
type DatabaseAccessor interface {
	// GetValue retrieves a value from the database by key.
	GetValue(key string) (string, error)

	// SetValue stores a value in the database with a given key.
	SetValue(key, value string) error

	// StoreJSON stores a JSON-serializable object with the given key,
	// dual-writing to KNIRVBASE when the bridge is enabled.
	StoreJSON(key string, data interface{}) error
}
