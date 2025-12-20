package distributed

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	coll "github.com/knirv/knirvbase/internal/collection"
	db "github.com/knirv/knirvbase/internal/database"
	typ "github.com/knirv/knirvbase/internal/types"
)

// KNIRVQLParser parses KNIRVQL queries
type KNIRVQLParser struct{}

// Parse parses a KNIRVQL query
func (p *KNIRVQLParser) Parse(query string) (*Query, error) {
	query = strings.TrimSpace(query)
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty query")
	}

	cmd := strings.ToUpper(parts[0])
	switch cmd {
	case "GET":
		return p.parseGet(parts[1:])
	case "SET":
		return p.parseSet(parts[1:])
	case "DELETE":
		return p.parseDelete(parts[1:])
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd)
	}
}

func (p *KNIRVQLParser) parseGet(parts []string) (*Query, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GET query")
	}

	entryType := strings.ToUpper(parts[0])
	var collection string
	var filters []Filter
	var similarTo []float64
	var limit int

	i := 1
	if parts[i] == "WHERE" {
		i++
		for i < len(parts) {
			if parts[i] == "SIMILAR" && i+1 < len(parts) && parts[i+1] == "TO" {
				i += 2
				if i < len(parts) {
					vecStr := parts[i]
					vecStr = strings.Trim(vecStr, "[]")
					vecParts := strings.Split(vecStr, ",")
					for _, vp := range vecParts {
						if v, err := strconv.ParseFloat(strings.TrimSpace(vp), 64); err == nil {
							similarTo = append(similarTo, v)
						}
					}
				}
				break
			} else if parts[i] == "LIMIT" {
				i++
				if i < len(parts) {
					if l, err := strconv.Atoi(parts[i]); err == nil {
						limit = l
					}
				}
				break
			} else {
				// Simple filter: key = value
				if i+2 < len(parts) && parts[i+1] == "=" {
					value := strings.Trim(parts[i+2], "\"")
					filters = append(filters, Filter{Key: parts[i], Value: value})
					i += 3
				} else {
					break
				}
			}
		}
	}

	return &Query{
		Type:       QueryGet,
		EntryType:  typ.EntryType(entryType),
		Collection: collection,
		Filters:    filters,
		SimilarTo:  similarTo,
		Limit:      limit,
	}, nil
}

func (p *KNIRVQLParser) parseSet(parts []string) (*Query, error) {
	if len(parts) < 3 || parts[1] != "=" {
		return nil, fmt.Errorf("invalid SET query")
	}

	key := parts[0]
	value := strings.Join(parts[2:], " ")
	value = strings.Trim(value, "\"")

	return &Query{
		Type:      QuerySet,
		Key:       key,
		Value:     value,
		EntryType: typ.EntryTypeAuth,
	}, nil
}

func (p *KNIRVQLParser) parseDelete(parts []string) (*Query, error) {
	if len(parts) < 2 || parts[0] != "WHERE" || parts[1] != "id" || parts[2] != "=" {
		return nil, fmt.Errorf("invalid DELETE query")
	}

	id := parts[3]

	return &Query{
		Type: QueryDelete,
		ID:   id,
	}, nil
}

// Query represents a parsed KNIRVQL query
type Query struct {
	Type       QueryType
	EntryType  typ.EntryType
	Collection string
	ID         string
	Key        string
	Value      string
	Filters    []Filter
	SimilarTo  []float64
	Limit      int
}

// QueryType enum
type QueryType int

const (
	QueryGet QueryType = iota
	QuerySet
	QueryDelete
)

// Filter for WHERE clauses
type Filter struct {
	Key   string
	Value string
}

// Execute executes the query on the database
func (q *Query) Execute(db *db.DistributedDatabase, collection *coll.DistributedCollection) (interface{}, error) {
	switch q.Type {
	case QueryGet:
		if q.EntryType == typ.EntryTypeAuth {
			// For AUTH, find by key from filters
			if len(q.Filters) > 0 && q.Filters[0].Key == "key" {
				doc, err := collection.Find(q.Filters[0].Value)
				if err != nil || doc == nil {
					return nil, err
				}
				if p, ok := doc["payload"].(map[string]interface{}); ok {
					return p["value"], nil
				}
			}
			return nil, fmt.Errorf("invalid AUTH query")
		} else {
			// For MEMORY, find with filters
			docs, err := collection.FindAll()
			if err != nil {
				return nil, err
			}
			var results []map[string]interface{}
			for _, doc := range docs {
				if p, ok := doc["payload"].(map[string]interface{}); ok {
					if q.matchesFilters(p) {
						results = append(results, doc)
					}
				}
			}
			if q.Limit > 0 && len(results) > q.Limit {
				results = results[:q.Limit]
			}
			return results, nil
		}
	case QuerySet:
		doc := map[string]interface{}{
			"id":        q.Key,
			"entryType": typ.EntryTypeAuth,
			"payload": map[string]interface{}{
				"key":   q.Key,
				"value": q.Value,
			},
		}
		_, err := collection.Insert(context.TODO(), doc)
		return nil, err
	case QueryDelete:
		_, err := collection.Delete(q.ID)
		return nil, err
	default:
		return nil, fmt.Errorf("unsupported query type")
	}
}

func (q *Query) matchesFilters(payload map[string]interface{}) bool {
	for _, f := range q.Filters {
		if val, ok := payload[f.Key]; !ok || fmt.Sprintf("%v", val) != f.Value {
			return false
		}
	}
	return true
}
