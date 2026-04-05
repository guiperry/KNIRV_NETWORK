package query

import (
	"context"
	"os"
	"testing"

	dbpkg "github.com/knirvcorp/knirvbase/go/internal/database"
	stor "github.com/knirvcorp/knirvbase/go/internal/storage"
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
