package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	dbpkg "github.com/knirv/knirvbase/internal/database"
	qry "github.com/knirv/knirvbase/internal/query"
	stor "github.com/knirv/knirvbase/internal/storage"
	typ "github.com/knirv/knirvbase/internal/types"
)

func main() {
	ctx := context.Background()

	// Get app data directory
	appDataDir := os.Getenv("XDG_DATA_HOME")
	if appDataDir == "" {
		home, _ := os.UserHomeDir()
		appDataDir = filepath.Join(home, ".local", "share", "knirvbase")
	}
	os.MkdirAll(appDataDir, 0755)

	// Create storage
	store := stor.NewFileStorage(appDataDir)

	// Create database
	opts := dbpkg.DistributedDbOptions{}
	db, err := dbpkg.NewDistributedDatabase(ctx, opts, store)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Shutdown()

	// Create network
	networkID, err := db.CreateNetwork(typ.NetworkConfig{
		NetworkID: "consortium-1",
		Name:      "Consortium 1",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create collections
	authColl := db.Collection("auth", store)
	memoryColl := db.Collection("memory", store)

	// Attach to network
	if err := db.AddCollectionToNetwork(networkID, "auth"); err != nil {
		log.Fatal(err)
	}
	if err := db.AddCollectionToNetwork(networkID, "memory"); err != nil {
		log.Fatal(err)
	}

	// Example usage
	fmt.Println("KNIRVBASE Distributed Database Started")

	// Example KNIRVQL
	parser := &qry.KNIRVQLParser{}

	// Set auth key
	query, err := parser.Parse(`SET google_maps_api_key = "AIzaSy..."`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Execute(db, authColl)
	if err != nil {
		log.Fatal(err)
	}

	// Get auth key
	query, err = parser.Parse(`GET AUTH WHERE key = "google_maps_api_key"`)
	if err != nil {
		log.Fatal(err)
	}
	result, err := query.Execute(db, authColl)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Auth key: %v\n", result)

	// Insert memory entry
	doc := map[string]interface{}{
		"id":        "mem1",
		"entryType": typ.EntryTypeMemory,
		"payload": map[string]interface{}{
			"source": "web-scrape",
			"data":   "some data",
			"vector": []float64{0.45, 0.12},
		},
	}
	_, err = memoryColl.Insert(ctx, doc)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Inserted memory entry")

	// Query memory
	query, err = parser.Parse(`GET MEMORY WHERE source = "web-scrape" SIMILAR TO [0.45, 0.12] LIMIT 10`)
	if err != nil {
		log.Fatal(err)
	}
	results, err := query.Execute(db, memoryColl)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Memory results: %v\n", results)

	fmt.Println("KNIRVBASE running. Press Ctrl+C to exit.")
	select {}
}
