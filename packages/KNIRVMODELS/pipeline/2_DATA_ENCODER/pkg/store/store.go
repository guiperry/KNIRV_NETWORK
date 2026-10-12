// Package store defines the encoder's storage boundary.
package store

import "context"

type Collection interface {
	Insert(context.Context, map[string]interface{}) (map[string]interface{}, error)
	Update(context.Context, string, map[string]interface{}) (int, error)
	Delete(context.Context, string) (int, error)
	Find(context.Context, string) (map[string]interface{}, error)
	FindAll(context.Context) ([]map[string]interface{}, error)
	AttachToNetwork(string) error
	DetachFromNetwork() error
	ForceSync() error
}
