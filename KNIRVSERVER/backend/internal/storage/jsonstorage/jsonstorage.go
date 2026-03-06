package jsonstorage

// JSONStorage defines an interface for storing and retrieving JSON data.
type JSONStorage interface {
	// StoreJSON stores a JSON value for a given key.
	StoreJSON(key string, data []byte) error
	// GetJSON retrieves a JSON value for a given key into dest.
	GetJSON(key string, dest interface{}) error
}