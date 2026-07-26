package query

import (
	"context"
	"os"
	"testing"

	dbpkg "github.com/knirvcorp/knirvbase/internal/database"
	stor "github.com/knirvcorp/knirvbase/internal/storage"
)

func TestKNIRVQLIndexCommands(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "knirvql_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage and database
	store := stor.NewFileStorage(tmpDir)
	database, err := dbpkg.NewDistributedDatabase(context.Background(), dbpkg.DistributedDbOptions{}, store)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Shutdown()

	// Get collection
	collection := database.Collection("users", store)

	// Test CREATE INDEX command
	parser := &KNIRVQLParser{}
	query, err := parser.Parse("CREATE INDEX users:username ON users (username) UNIQUE")
	if err != nil {
		t.Fatalf("Failed to parse CREATE INDEX: %v", err)
	}

	_, err = query.Execute(database, collection)
	if err != nil {
		t.Fatalf("Failed to execute CREATE INDEX: %v", err)
	}

	// Verify index was created
	indexes := database.GetIndexesForCollection(context.TODO(), "users")
	if len(indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(indexes))
	}

	if indexes[0].Name != "username" || !indexes[0].Unique || indexes[0].Type != stor.IndexTypeBTree {
		t.Fatal("Index properties incorrect")
	}

	// Insert some test data
	doc1 := map[string]interface{}{
		"id":       "user1",
		"username": "alice",
		"email":    "alice@example.com",
	}

	doc2 := map[string]interface{}{
		"id":       "user2",
		"username": "bob",
		"email":    "bob@example.com",
	}

	_, err = collection.Insert(context.Background(), doc1)
	if err != nil {
		t.Fatalf("Failed to insert user1: %v", err)
	}

	_, err = collection.Insert(context.Background(), doc2)
	if err != nil {
		t.Fatalf("Failed to insert user2: %v", err)
	}

	// Test DROP INDEX command
	query, err = parser.Parse("DROP INDEX users:username")
	if err != nil {
		t.Fatalf("Failed to parse DROP INDEX: %v", err)
	}

	_, err = query.Execute(database, collection)
	if err != nil {
		t.Fatalf("Failed to execute DROP INDEX: %v", err)
	}

	// Verify index was dropped
	indexes = database.GetIndexesForCollection(context.TODO(), "users")
	if len(indexes) != 0 {
		t.Fatalf("Expected 0 indexes after drop, got %d", len(indexes))
	}
}

func TestKNIRVQLInsertWithIndex(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "knirvql_insert_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage and database
	store := stor.NewFileStorage(tmpDir)
	database, err := dbpkg.NewDistributedDatabase(context.Background(), dbpkg.DistributedDbOptions{}, store)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Shutdown()

	collection := database.Collection("credentials", store)

	// Create index
	err = database.CreateIndex(context.TODO(), "credentials", "username", stor.IndexTypeBTree, []string{"username"}, true, "", nil)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Insert document
	doc := map[string]interface{}{
		"id":       "cred1",
		"username": "alice@example.com",
		"hash":     "hashed_password",
		"status":   "active",
	}

	_, err = collection.Insert(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Query index
	results, err := database.QueryIndex(context.TODO(), "credentials", "username", map[string]interface{}{
		"value": "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Failed to query index: %v", err)
	}

	if len(results) != 1 || results[0] != "cred1" {
		t.Fatalf("Expected [cred1], got %v", results)
	}
}

func TestQueryOptimizer(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "optimizer_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage and database
	store := stor.NewFileStorage(tmpDir)
	database, err := dbpkg.NewDistributedDatabase(context.Background(), dbpkg.DistributedDbOptions{}, store)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Shutdown()

	// Create index
	err = database.CreateIndex(context.TODO(), "test", "name", stor.IndexTypeBTree, []string{"name"}, false, "", nil)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Create optimizer
	indexes := database.GetIndexesForCollection(context.TODO(), "test")
	optimizer := NewQueryOptimizer("test", indexes, nil)

	// Create query
	parser := &KNIRVQLParser{}
	query, err := parser.Parse("GET MEMORY WHERE name = \"alice\"")
	if err != nil {
		t.Fatalf("Failed to parse query: %v", err)
	}

	// Optimize
	plan, err := optimizer.Optimize(query)
	if err != nil {
		t.Fatalf("Failed to optimize: %v", err)
	}

	// Check plan
	if !plan.UseIndex {
		t.Fatal("Expected to use index")
	}
	if plan.IndexName != "name" {
		t.Fatalf("Expected index name 'name', got %s", plan.IndexName)
	}
	if plan.ScanType != IndexOnlyScan {
		t.Fatalf("Expected IndexOnlyScan, got %v", plan.ScanType)
	}
}

func TestKNIRVQLModalityQuery(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "knirvql_modality_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage and database
	store := stor.NewFileStorage(tmpDir)
	database, err := dbpkg.NewDistributedDatabase(context.Background(), dbpkg.DistributedDbOptions{}, store)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Shutdown()

	// Parse modality query
	parser := &KNIRVQLParser{}
	query, err := parser.Parse("GET MEMORY.MODALITY(vector) WHERE dataset = \"test\" AND verified = true")
	if err != nil {
		t.Fatalf("Failed to parse modality query: %v", err)
	}

	// Check query type
	if query.Type != QueryGetModality {
		t.Fatalf("Expected QueryGetModality, got %d", query.Type)
	}

	// Check modality type
	if query.ModalityType != "vector" {
		t.Fatalf("Expected modality type 'vector', got '%s'", query.ModalityType)
	}

	// Check filters
	if len(query.Filters) != 2 {
		t.Fatalf("Expected 2 filters, got %d", len(query.Filters))
	}

	// Check dataset filter
	foundDataset := false
	for _, f := range query.Filters {
		if f.Key == "dataset" && f.Value == "test" {
			foundDataset = true
			break
		}
	}
	if !foundDataset {
		t.Error("Expected dataset filter with value 'test'")
	}

	// Check verified filter
	foundVerified := false
	for _, f := range query.Filters {
		if f.Key == "verified" && f.Value == true {
			foundVerified = true
			break
		}
	}
	if !foundVerified {
		t.Error("Expected verified filter with value true")
	}
}

func TestKNIRVQLModalityQueryDifferentTypes(t *testing.T) {
	parser := &KNIRVQLParser{}

	testCases := []struct {
		queryString      string
		expectedModality string
	}{
		{"GET MEMORY.MODALITY(seed)", "seed"},
		{"GET MEMORY.MODALITY(thermo)", "thermo"},
		{"GET MEMORY.MODALITY(proof)", "proof"},
	}

	for _, tc := range testCases {
		query, err := parser.Parse(tc.queryString)
		if err != nil {
			t.Fatalf("Failed to parse query '%s': %v", tc.queryString, err)
		}

		if query.Type != QueryGetModality {
			t.Fatalf("Expected QueryGetModality for '%s', got %d", tc.queryString, query.Type)
		}

		if query.ModalityType != tc.expectedModality {
			t.Fatalf("Expected modality '%s' for query '%s', got '%s'", tc.expectedModality, tc.queryString, query.ModalityType)
		}
	}
}

func TestKNIRVQLModalityQueryWithLimit(t *testing.T) {
	parser := &KNIRVQLParser{}
	query, err := parser.Parse("GET MEMORY.MODALITY(vector) WHERE verified = true LIMIT 100")
	if err != nil {
		t.Fatalf("Failed to parse modality query with limit: %v", err)
	}

	if query.Type != QueryGetModality {
		t.Fatalf("Expected QueryGetModality, got %d", query.Type)
	}

	if query.ModalityType != "vector" {
		t.Fatalf("Expected modality type 'vector', got '%s'", query.ModalityType)
	}

	if query.Limit != 100 {
		t.Fatalf("Expected limit 100, got %d", query.Limit)
	}
}

func TestKNIRVQL_Z3StatusFilter(t *testing.T) {
	parser := &KNIRVQLParser{}
	query, err := parser.Parse("GET MEMORY WHERE z3_status = VALID")
	if err != nil {
		t.Fatalf("Failed to parse query: %v", err)
	}

	if len(query.Filters) != 1 {
		t.Fatalf("Expected 1 filter, got %d", len(query.Filters))
	}

	if query.Filters[0].Key != "z3_status" {
		t.Fatalf("Expected filter key 'z3_status', got '%s'", query.Filters[0].Key)
	}

	if query.Filters[0].Operator != "=" {
		t.Fatalf("Expected operator '=', got '%s'", query.Filters[0].Operator)
	}

	if query.Filters[0].Value != "VALID" {
		t.Fatalf("Expected filter value 'VALID', got '%v'", query.Filters[0].Value)
	}
}

func TestKNIRVQL_ThermoFilter(t *testing.T) {
	parser := &KNIRVQLParser{}
	query, err := parser.Parse("GET MEMORY WHERE avg_temp_c < 85")
	if err != nil {
		t.Fatalf("Failed to parse query: %v", err)
	}

	if len(query.Filters) != 1 {
		t.Fatalf("Expected 1 filter, got %d", len(query.Filters))
	}

	if query.Filters[0].Key != "avg_temp_c" {
		t.Fatalf("Expected filter key 'avg_temp_c', got '%s'", query.Filters[0].Key)
	}

	if query.Filters[0].Operator != "<" {
		t.Fatalf("Expected operator '<', got '%s'", query.Filters[0].Operator)
	}

	if query.Filters[0].Value != 85.0 {
		t.Fatalf("Expected filter value 85.0, got %v", query.Filters[0].Value)
	}
}

func TestKNIRVQL_BracketFieldQuery(t *testing.T) {
	parser := &KNIRVQLParser{}
	query, err := parser.Parse("GET MEMORY.BRACKET(golden_seed)")
	if err != nil {
		t.Fatalf("Failed to parse query: %v", err)
	}

	if query.Type != QueryGetBracketField {
		t.Fatalf("Expected QueryGetBracketField, got %d", query.Type)
	}

	if query.BracketField != "golden_seed" {
		t.Fatalf("Expected bracket field 'golden_seed', got '%s'", query.BracketField)
	}
}

func TestKNIRVQL_SemanticFiltersMatch(t *testing.T) {
	parser := &KNIRVQLParser{}
	query, err := parser.Parse("GET MEMORY WHERE pos_tag = VERB AND domain = MATH")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	doc := map[string]interface{}{
		"payload": map[string]interface{}{
			"thermo": map[string]interface{}{"temp_celsius": float64(72)},
			"z3":     map[string]interface{}{"status": "VALID"},
			"brackets_index": []map[string]interface{}{
				{
					"id":         "b1",
					"type":       "I",
					"pos_tag":    float64(0x02),
					"domain_sig": float64(0x2000),
					"tense":      float64(2),
				},
			},
		},
	}

	if !query.matchesFiltersWithPlan(doc, query.Filters) {
		t.Fatal("expected semantic filters to match VERB + MATH bracket")
	}
}

func TestKNIRVQL_IntentFlagsBitFilter(t *testing.T) {
	parser := &KNIRVQLParser{}
	query, err := parser.Parse("GET MEMORY WHERE intent_flags & 0x1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(query.Filters) != 1 || query.Filters[0].Operator != "&" {
		t.Fatalf("unexpected filters: %+v", query.Filters)
	}

	doc := map[string]interface{}{
		"payload": map[string]interface{}{
			"brackets_index": []map[string]interface{}{
				{"intent_flags": float64(0x3)},
			},
		},
	}
	if !query.matchesFiltersWithPlan(doc, query.Filters) {
		t.Fatal("expected intent_flags mask to match")
	}
}
