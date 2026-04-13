package query

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	coll "github.com/knirvcorp/knirvbase/go/internal/collection"
	db "github.com/knirvcorp/knirvbase/go/internal/database"
	stor "github.com/knirvcorp/knirvbase/go/internal/storage"
	typ "github.com/knirvcorp/knirvbase/go/internal/types"
)

type KNIRVQLParser struct{}

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
	case "CREATE":
		return p.parseCreate(parts[1:])
	case "DROP":
		return p.parseDrop(parts[1:])
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd)
	}
}

func (p *KNIRVQLParser) parseGet(parts []string) (*Query, error) {
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid GET query")
	}

	entryTypeOriginal := parts[0]
	entryType := strings.ToUpper(parts[0])
	var collection string
	var filters []Filter
	var similarTo []float64
	var limit int
	var modalityType string
	var bracketField string

	if strings.HasPrefix(entryTypeOriginal, "MEMORY.MODALITY(") && strings.HasSuffix(entryTypeOriginal, ")") {
		modalityName := entryTypeOriginal[len("MEMORY.MODALITY(") : len(entryTypeOriginal)-1]
		modalityType = modalityName
		entryType = "MEMORY"
	}

	if strings.HasPrefix(entryTypeOriginal, "MEMORY.BRACKET(") && strings.HasSuffix(entryTypeOriginal, ")") {
		bracketField = entryTypeOriginal[len("MEMORY.BRACKET(") : len(entryTypeOriginal)-1]
		entryType = "MEMORY"
	}

	var queryType QueryType
	if bracketField != "" {
		queryType = QueryGetBracketField
	} else if modalityType != "" {
		queryType = QueryGetModality
	} else {
		queryType = QueryGet
	}

	i := 1
	if i < len(parts) && parts[i] == "WHERE" {
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
			} else if parts[i] == "WITH" && i+1 < len(parts) && parts[i+1] == "TAGS" {
				i += 2
				if i < len(parts) && strings.HasPrefix(parts[i], "(") {
					tagStr := strings.Trim(parts[i], "()")
					tagParts := strings.Split(tagStr, ",")
					for _, t := range tagParts {
						filters = append(filters, Filter{Key: "tags", Operator: "CONTAINS", Value: strings.TrimSpace(t)})
					}
					i++
				}
			} else if parts[i] == "LIMIT" {
				i++
				if i < len(parts) {
					if l, err := strconv.Atoi(parts[i]); err == nil {
						limit = l
					}
				}
				break
			} else if parts[i] == "AND" || parts[i] == "OR" {
				i++
				continue
			} else {
				if i+2 < len(parts) {
					key := parts[i]
					operator := parts[i+1]
					valueStr := strings.Trim(parts[i+2], "\"")
					var value interface{}
					if valueStr == "true" {
						value = true
					} else if valueStr == "false" {
						value = false
					} else if f, err := strconv.ParseFloat(valueStr, 64); err == nil {
						value = f
					} else if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
						value = val
					} else if ls := strings.ToLower(valueStr); strings.HasPrefix(ls, "0x") {
						if u, err := strconv.ParseUint(ls[2:], 16, 64); err == nil {
							value = u
						} else {
							value = valueStr
						}
					} else {
						value = valueStr
					}
					filters = append(filters, Filter{Key: key, Operator: operator, Value: value})
					i += 3
				} else {
					break
				}
			}
		}
	}

	return &Query{
		Type:         queryType,
		EntryType:    typ.EntryType(entryType),
		Collection:   collection,
		Filters:      filters,
		SimilarTo:    similarTo,
		Limit:        limit,
		ModalityType: modalityType,
		BracketField: bracketField,
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

func (p *KNIRVQLParser) parseCreate(parts []string) (*Query, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid CREATE command")
	}

	subCmd := strings.ToUpper(parts[0])
	switch subCmd {
	case "INDEX":
		return p.parseCreateIndex(parts[1:])
	case "COLLECTION":
		return p.parseCreateCollection(parts[1:])
	default:
		return nil, fmt.Errorf("unknown CREATE command: %s", subCmd)
	}
}

func (p *KNIRVQLParser) parseCreateIndex(parts []string) (*Query, error) {
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid CREATE INDEX command")
	}

	indexRef := parts[0]
	indexParts := strings.Split(indexRef, ":")
	if len(indexParts) != 2 {
		return nil, fmt.Errorf("invalid index reference, expected collection:index")
	}
	collection := indexParts[0]
	indexName := indexParts[1]

	if parts[1] != "ON" {
		return nil, fmt.Errorf("expected ON after index name")
	}

	if parts[2] != collection {
		return nil, fmt.Errorf("collection mismatch in index definition")
	}

	fields := []string{}

	i := 3
	if i < len(parts) && strings.HasPrefix(parts[i], "(") {
		fieldStr := strings.Trim(parts[i], "()")
		if fieldStr != "" {
			fieldParts := strings.Split(fieldStr, ",")
			for _, f := range fieldParts {
				f = strings.TrimSpace(f)
				if f != "" {
					fields = append(fields, f)
				}
			}
		}
		i++
	}

	unique := false
	if i < len(parts) && strings.ToUpper(parts[i]) == "UNIQUE" {
		unique = true
		i++
	}

	return &Query{
		Type:       QueryCreateIndex,
		IndexName:  indexName,
		Collection: collection,
		Fields:     fields,
		Unique:     unique,
	}, nil
}

func (p *KNIRVQLParser) parseCreateCollection(parts []string) (*Query, error) {
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid CREATE COLLECTION command")
	}

	collectionName := parts[0]

	return &Query{
		Type:       QueryCreateCollection,
		Collection: collectionName,
	}, nil
}

func (p *KNIRVQLParser) parseDrop(parts []string) (*Query, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid DROP command")
	}

	subCmd := strings.ToUpper(parts[0])
	switch subCmd {
	case "INDEX":
		return p.parseDropIndex(parts[1:])
	case "COLLECTION":
		return p.parseDropCollection(parts[1:])
	default:
		return nil, fmt.Errorf("unknown DROP command: %s", subCmd)
	}
}

func (p *KNIRVQLParser) parseDropIndex(parts []string) (*Query, error) {
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid DROP INDEX command")
	}

	indexRef := parts[0]
	indexParts := strings.Split(indexRef, ":")
	if len(indexParts) != 2 {
		return nil, fmt.Errorf("invalid index reference, expected collection:index")
	}

	return &Query{
		Type:       QueryDropIndex,
		Collection: indexParts[0],
		IndexName:  indexParts[1],
	}, nil
}

func (p *KNIRVQLParser) parseDropCollection(parts []string) (*Query, error) {
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid DROP COLLECTION command")
	}

	collectionName := parts[0]

	return &Query{
		Type:       QueryDropCollection,
		Collection: collectionName,
	}, nil
}

type Query struct {
	Type         QueryType
	EntryType    typ.EntryType
	Collection   string
	ID           string
	Key          string
	Value        string
	Filters      []Filter
	SimilarTo    []float64
	Limit        int
	IndexName    string
	Fields       []string
	Unique       bool
	ModalityType string
	BracketField string
}

type QueryType int

const (
	QueryGet QueryType = iota
	QuerySet
	QueryDelete
	QueryCreateIndex
	QueryCreateCollection
	QueryDropIndex
	QueryDropCollection
	QueryGetModality
	QueryGetBracketField
)

type Filter struct {
	Key      string
	Operator string
	Value    interface{}
}

func (q *Query) Execute(db *db.DistributedDatabase, collection *coll.DistributedCollection) (interface{}, error) {
	ctx := context.TODO()
	switch q.Type {
	case QueryGet, QueryGetBracketField:
		return q.executeGet(db, collection)
	case QuerySet:
		doc := map[string]interface{}{
			"id":        q.Key,
			"entryType": typ.EntryTypeAuth,
			"payload": map[string]interface{}{
				"key":   q.Key,
				"value": q.Value,
			},
		}
		_, err := collection.Insert(ctx, doc)
		return nil, err
	case QueryDelete:
		_, err := collection.Delete(ctx, q.ID)
		return nil, err
	case QueryCreateIndex:
		indexType := stor.IndexTypeBTree
		if q.IndexName == "vector" {
			indexType = stor.IndexTypeHNSW
		} else if q.IndexName == "tag" {
			indexType = stor.IndexTypeTag
		}
		return nil, db.CreateIndex(ctx, q.Collection, q.IndexName, indexType, q.Fields, q.Unique, "", nil)
	case QueryCreateCollection:
		return nil, nil
	case QueryDropIndex:
		return nil, db.DropIndex(ctx, q.Collection, q.IndexName)
	case QueryDropCollection:
		return nil, nil
	case QueryGetModality:
		return q.executeGet(db, collection)
	default:
		return nil, fmt.Errorf("unsupported query type")
	}
}

func (q *Query) executeGet(db *db.DistributedDatabase, collection *coll.DistributedCollection) (interface{}, error) {
	collectionName := q.Collection
	if collectionName == "" {
		if q.EntryType == typ.EntryTypeAuth {
			collectionName = "auth"
		} else {
			collectionName = "memory"
		}
	}

	indexes := db.GetIndexesForCollection(context.TODO(), collectionName)

	optimizer := NewQueryOptimizer(collectionName, indexes, nil)

	plan, err := optimizer.Optimize(q)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize query: %w", err)
	}

	return q.executePlan(plan, db, collection)
}

func (q *Query) executePlan(plan *QueryPlan, db *db.DistributedDatabase, collection *coll.DistributedCollection) (interface{}, error) {
	switch plan.ScanType {
	case FullScan:
		return q.executeFullScan(plan, collection)
	case IndexScan:
		return q.executeIndexScan(plan, db, collection)
	case IndexOnlyScan:
		return q.executeIndexOnlyScan(plan, db)
	default:
		return q.executeFullScan(plan, collection)
	}
}

func (q *Query) executeFullScan(plan *QueryPlan, collection *coll.DistributedCollection) (interface{}, error) {
	ctx := context.TODO()
	docs, err := collection.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for _, doc := range docs {
		if q.matchesFiltersWithPlan(doc, plan.PostFilters) {
			results = append(results, doc)
		}
	}

	if plan.Limit > 0 && len(results) > plan.Limit {
		results = results[:plan.Limit]
	}

	return q.shapeMemoryResults(results)
}

func (q *Query) executeIndexScan(plan *QueryPlan, db *db.DistributedDatabase, collection *coll.DistributedCollection) (interface{}, error) {
	ctx := context.TODO()
	docIDs, err := db.QueryIndex(ctx, plan.IndexName, plan.IndexName, map[string]interface{}{
		"value": plan.IndexFilters[0].Value,
	})
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for _, docID := range docIDs {
		doc, err := collection.Find(ctx, docID)
		if err != nil {
			continue
		}
		if doc != nil && q.matchesFiltersWithPlan(doc, plan.PostFilters) {
			results = append(results, doc)
		}
	}

	if plan.Limit > 0 && len(results) > plan.Limit {
		results = results[:plan.Limit]
	}

	return q.shapeMemoryResults(results)
}

func (q *Query) executeIndexOnlyScan(plan *QueryPlan, db *db.DistributedDatabase) (interface{}, error) {
	docIDs, err := db.QueryIndex(context.TODO(), plan.IndexName, plan.IndexName, map[string]interface{}{
		"value": plan.IndexFilters[0].Value,
	})
	if err != nil {
		return nil, err
	}

	if plan.Limit > 0 && len(docIDs) > plan.Limit {
		docIDs = docIDs[:plan.Limit]
	}

	return docIDs, nil
}

func (q *Query) matchesFiltersWithPlan(doc map[string]interface{}, filters []Filter) bool {
	payload, ok := doc["payload"].(map[string]interface{})
	if !ok {
		return false
	}
	for _, f := range filters {
		if !q.matchesFilter(doc, payload, f) {
			return false
		}
	}
	return true
}

func (q *Query) matchesFilter(doc map[string]interface{}, payload map[string]interface{}, filter Filter) bool {
	switch filter.Key {
	case "z3_status":
		if z3, ok := payload["z3"].(map[string]interface{}); ok {
			return fmt.Sprintf("%v", z3["status"]) == fmt.Sprintf("%v", filter.Value)
		}
		return false
	case "avg_temp_c":
		if thermo, ok := payload["thermo"].(map[string]interface{}); ok {
			if tempC, ok := floatFromInterface(thermo["temp_celsius"]); ok {
				cmp := compareNumericValues(tempC, filter.Value)
				opFunc := compareOperator(filter.Operator)
				return opFunc(cmp)
			}
		}
		return false
	case "drift_score":
		if brackets, ok := payload["brackets"].(map[string]interface{}); ok {
			if maxDrift, ok := floatFromInterface(brackets["max_drift"]); ok {
				cmp := compareNumericValues(maxDrift, filter.Value)
				opFunc := compareOperator(filter.Operator)
				return opFunc(cmp)
			}
		}
		return false
	case "bracket_type":
		want := strings.ToUpper(fmt.Sprintf("%v", filter.Value))
		for _, br := range bracketsIndexFromPayload(payload) {
			if strings.ToUpper(fmt.Sprintf("%v", br["type"])) == want {
				return true
			}
		}
		if brackets, ok := payload["brackets"].(map[string]interface{}); ok {
			if types, ok := brackets["types"].([]string); ok {
				for _, t := range types {
					if strings.ToUpper(t) == want {
						return true
					}
				}
			}
			if typesAny, ok := brackets["types"].([]interface{}); ok {
				for _, t := range typesAny {
					if strings.ToUpper(fmt.Sprintf("%v", t)) == want {
						return true
					}
				}
			}
		}
		return false
	case "pos_tag":
		want := parsePOSToken(filter.Value)
		for _, br := range bracketsIndexFromPayload(payload) {
			if uint8FromInterface(br["pos_tag"]) == want {
				return true
			}
		}
		return false
	case "tense":
		want := parseTenseToken(filter.Value)
		for _, br := range bracketsIndexFromPayload(payload) {
			if uint8FromInterface(br["tense"]) == want {
				return true
			}
		}
		return false
	case "domain":
		want := parseDomainToken(filter.Value)
		for _, br := range bracketsIndexFromPayload(payload) {
			if uint16FromInterface(br["domain_sig"]) == want {
				return true
			}
		}
		return false
	case "intent_flags":
		mask := uint8(uintFromFilterValue(filter.Value))
		for _, br := range bracketsIndexFromPayload(payload) {
			flags := uint8FromInterface(br["intent_flags"])
			if filter.Operator == "&" {
				if (flags & mask) == mask {
					return true
				}
			} else if filter.Operator == "=" {
				if flags == mask {
					return true
				}
			}
		}
		return false
	case "lsh_salt":
		want := uint32FromInterface(filter.Value)
		for _, br := range bracketsIndexFromPayload(payload) {
			got := uint32FromInterface(br["lsh_salt"])
			switch filter.Operator {
			case "=":
				if got == want {
					return true
				}
			case "!=":
				if got != want {
					return true
				}
			case ">":
				if compareNumericValues(float64(got), want) > 0 {
					return true
				}
			case "<":
				if compareNumericValues(float64(got), want) < 0 {
					return true
				}
			case ">=":
				if compareNumericValues(float64(got), want) >= 0 {
					return true
				}
			case "<=":
				if compareNumericValues(float64(got), want) <= 0 {
					return true
				}
			}
		}
		return false
	default:
		val, ok := payload[filter.Key]
		if !ok {
			return false
		}
		switch filter.Operator {
		case "=":
			return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", filter.Value)
		case "!=":
			return fmt.Sprintf("%v", val) != fmt.Sprintf("%v", filter.Value)
		case ">":
			return compareValues(val, filter.Value) > 0
		case "<":
			return compareValues(val, filter.Value) < 0
		case ">=":
			return compareValues(val, filter.Value) >= 0
		case "<=":
			return compareValues(val, filter.Value) <= 0
		case "CONTAINS":
			valStr := fmt.Sprintf("%v", val)
			filterStr := fmt.Sprintf("%v", filter.Value)
			return strings.Contains(valStr, filterStr)
		case "STARTS_WITH":
			valStr := fmt.Sprintf("%v", val)
			filterStr := fmt.Sprintf("%v", filter.Value)
			return strings.HasPrefix(valStr, filterStr)
		default:
			return false
		}
	}
}

func compareOperator(op string) func(int) bool {
	switch op {
	case ">":
		return func(i int) bool { return i > 0 }
	case "<":
		return func(i int) bool { return i < 0 }
	case ">=":
		return func(i int) bool { return i >= 0 }
	case "<=":
		return func(i int) bool { return i <= 0 }
	default:
		return func(i int) bool { return i == 0 }
	}
}

func compareValues(a, b interface{}) int {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

func compareNumericValues(a float64, b interface{}) int {
	var bVal float64
	switch v := b.(type) {
	case float64:
		bVal = v
	case float32:
		bVal = float64(v)
	case int:
		bVal = float64(v)
	case int64:
		bVal = float64(v)
	case uint64:
		bVal = float64(v)
	case uint32:
		bVal = float64(v)
	default:
		bStr := fmt.Sprintf("%v", v)
		if parsed, err := strconv.ParseFloat(bStr, 64); err == nil {
			bVal = parsed
		}
	}
	if a < bVal {
		return -1
	} else if a > bVal {
		return 1
	}
	return 0
}

func (q *Query) shapeMemoryResults(docs []map[string]interface{}) (interface{}, error) {
	switch q.Type {
	case QueryGetBracketField:
		return q.projectBracketField(docs), nil
	case QueryGetModality:
		return q.projectModality(docs), nil
	default:
		return docs, nil
	}
}

func (q *Query) projectBracketField(docs []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0)
	field := strings.ToLower(strings.TrimSpace(q.BracketField))
	for _, doc := range docs {
		payload, _ := doc["payload"].(map[string]interface{})
		for _, br := range bracketsIndexFromPayload(payload) {
			v := extractBracketFieldValue(br, field)
			if v == nil {
				continue
			}
			row := map[string]interface{}{
				"frame_id":   doc["id"],
				"bracket_id": br["id"],
				field:        v,
			}
			out = append(out, row)
		}
	}
	return out
}

func (q *Query) projectModality(docs []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0)
	mod := strings.ToLower(strings.TrimSpace(q.ModalityType))
	for _, doc := range docs {
		payload, _ := doc["payload"].(map[string]interface{})
		idx := bracketsIndexFromPayload(payload)
		if len(idx) == 0 {
			continue
		}
		br := idx[0]
		row := map[string]interface{}{
			"frame_id": doc["id"],
			"modality": q.ModalityType,
		}
		switch mod {
		case "vector":
			if vec, ok := br["lsh_vector"].([]float64); ok {
				row["value"] = vec
			} else if vec, ok := br["lsh_vector"].([]interface{}); ok {
				row["value"] = vec
			}
		case "seed":
			row["value"] = uint32FromInterface(br["golden_seed"])
		case "thermo":
			if th, ok := payload["thermo"].(map[string]interface{}); ok {
				row["value"] = th
			}
		case "proof":
			row["value"] = nil
		default:
			if vec, ok := br["lsh_vector"].([]float64); ok {
				row["value"] = vec
			}
		}
		out = append(out, row)
	}
	return out
}

func extractBracketFieldValue(br map[string]interface{}, field string) interface{} {
	switch field {
	case "golden_seed":
		return uint32FromInterface(br["golden_seed"])
	case "subsecond_us":
		return uint32FromInterface(br["subsecond_us"])
	case "pos_tag":
		return uint8FromInterface(br["pos_tag"])
	case "tense":
		return uint8FromInterface(br["tense"])
	case "domain_sig", "domain":
		return uint16FromInterface(br["domain_sig"])
	case "intent_flags":
		return uint8FromInterface(br["intent_flags"])
	case "drift_score":
		if v, ok := floatFromInterface(br["drift_score"]); ok {
			return v
		}
		return nil
	case "lsh_salt":
		return uint32FromInterface(br["lsh_salt"])
	case "projections", "lsh_vector":
		return br["lsh_vector"]
	default:
		return br[field]
	}
}

func bracketsIndexFromPayload(payload map[string]interface{}) []map[string]interface{} {
	v, ok := payload["brackets_index"]
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case []map[string]interface{}:
		return x
	case []interface{}:
		var out []map[string]interface{}
		for _, e := range x {
			if m, ok := e.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func floatFromInterface(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint32:
		return float64(x), true
	default:
		return 0, false
	}
}

func uint8FromInterface(v interface{}) uint8 {
	switch x := v.(type) {
	case uint8:
		return x
	case int:
		return uint8(x)
	case int64:
		return uint8(x)
	case float64:
		return uint8(x)
	default:
		return 0
	}
}

func uint16FromInterface(v interface{}) uint16 {
	switch x := v.(type) {
	case uint16:
		return x
	case int:
		return uint16(x)
	case int64:
		return uint16(x)
	case float64:
		return uint16(x)
	default:
		return 0
	}
}

func uint32FromInterface(v interface{}) uint32 {
	switch x := v.(type) {
	case uint32:
		return x
	case int:
		return uint32(x)
	case int64:
		return uint32(x)
	case float64:
		return uint32(x)
	default:
		return 0
	}
}

func uintFromFilterValue(v interface{}) uint64 {
	switch x := v.(type) {
	case uint64:
		return x
	case uint32:
		return uint64(x)
	case int:
		return uint64(x)
	case int64:
		return uint64(x)
	case float64:
		return uint64(x)
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", x))
		if strings.HasPrefix(strings.ToLower(s), "0x") {
			u, err := strconv.ParseUint(s[2:], 16, 64)
			if err == nil {
				return u
			}
		}
		u, err := strconv.ParseUint(s, 10, 64)
		if err == nil {
			return u
		}
		return 0
	}
}

func parsePOSToken(v interface{}) uint8 {
	s := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", v)))
	switch s {
	case "NOUN":
		return 0x01
	case "VERB":
		return 0x02
	default:
		return uint8(uintFromFilterValue(v))
	}
}

func parseTenseToken(v interface{}) uint8 {
	s := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", v)))
	switch s {
	case "N/A", "NA":
		return 0
	case "PAST":
		return 1
	case "PRESENT":
		return 2
	case "FUTURE":
		return 3
	default:
		return uint8(uintFromFilterValue(v))
	}
}

func parseDomainToken(v interface{}) uint16 {
	s := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", v)))
	switch s {
	case "PROSE":
		return 0x1000
	case "MATH":
		return 0x2000
	case "CODE":
		return 0x3000
	default:
		return uint16(uintFromFilterValue(v))
	}
}
