package knirvbase

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	// Test with valid options
	tmpDir := t.TempDir()
	opts := Options{
		DataDir: tmpDir,
	}
	ctx := context.Background()
	db, err := New(ctx, opts)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if db == nil {
		t.Fatal("New() returned nil DB")
	}
	defer db.Shutdown()

	// Test with empty DataDir
	_, err = New(ctx, Options{DataDir: ""})
	if err == nil {
		t.Fatal("New() should fail with empty DataDir")
	}

	// Test with nil context (use context.TODO instead of nil)
	// The New function should accept context.TODO, not fail
	_, err = New(context.TODO(), opts)
	if err != nil {
		t.Fatalf("New() with context.TODO should succeed, got: %v", err)
	}
}

func TestDB_Collection(t *testing.T) {
	tmpDir := t.TempDir()
	opts := Options{
		DataDir: tmpDir,
	}
	ctx := context.Background()
	db, err := New(ctx, opts)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer db.Shutdown()

	// Test valid collection
	coll := db.Collection("test")
	if coll == nil {
		t.Fatal("Collection() returned nil")
	}

	// Test empty name (should panic, but we'll catch it)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Collection() should panic with empty name")
		}
	}()
	db.Collection("")
}

func TestCollection_Insert(t *testing.T) {
	tmpDir := t.TempDir()
	opts := Options{
		DataDir: tmpDir,
	}
	ctx := context.Background()
	db, err := New(ctx, opts)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer db.Shutdown()

	coll := db.Collection("test")

	// Test valid insert
	doc := map[string]interface{}{
		"id":   "test1",
		"data": "value",
	}
	result, err := coll.Insert(ctx, doc)
	if err != nil {
		t.Fatalf("Insert() failed: %v", err)
	}
	if result == nil {
		t.Fatal("Insert() returned nil")
	}

	// Test nil context (use context.TODO instead of nil)
	// The Insert method should accept context.TODO, not fail
	_, err = coll.Insert(context.TODO(), doc)
	if err != nil {
		t.Fatalf("Insert() with context.TODO should succeed, got: %v", err)
	}

	// Test nil doc
	_, err = coll.Insert(ctx, nil)
	if err == nil {
		t.Fatal("Insert() should fail with nil doc")
	}

	// Test doc without id
	_, err = coll.Insert(ctx, map[string]interface{}{"data": "value"})
	if err == nil {
		t.Fatal("Insert() should fail with doc without id")
	}

	// Test doc with empty id
	_, err = coll.Insert(ctx, map[string]interface{}{"id": ""})
	if err == nil {
		t.Fatal("Insert() should fail with empty id")
	}
}

func TestCollection_Find(t *testing.T) {
	tmpDir := t.TempDir()
	opts := Options{
		DataDir: tmpDir,
	}
	ctx := context.Background()
	db, err := New(ctx, opts)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer db.Shutdown()

	coll := db.Collection("test")

	// Insert a doc
	doc := map[string]interface{}{
		"id":   "test1",
		"data": "value",
	}
	_, err = coll.Insert(ctx, doc)
	if err != nil {
		t.Fatalf("Insert() failed: %v", err)
	}

	// Find it
	result, err := coll.Find(ctx, "test1")
	if err != nil {
		t.Fatalf("Find() failed: %v", err)
	}
	if result == nil {
		t.Fatal("Find() returned nil")
	}
	if result["id"] != "test1" {
		t.Errorf("Find() returned wrong id: %v", result["id"])
	}
}
