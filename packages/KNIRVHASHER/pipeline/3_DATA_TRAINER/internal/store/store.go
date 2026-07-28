// Package store defines the trainer's persistence boundary.
package store

import "context"

type Collection interface {
	Insert(context.Context, map[string]interface{}) (map[string]interface{}, error)
}
