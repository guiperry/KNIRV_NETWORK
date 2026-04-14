package main

import (
	"context"
	"log"
	"os"

	"github.com/knirvcorp/knirvbase"
	"github.com/knirvcorp/knirvhasher/pipeline/0_DATA_CONNECTOR/internal/grpc"
	"github.com/knirvcorp/knirvhasher/pipeline/0_DATA_CONNECTOR/internal/writer"
	hasherpb "github.com/knirvcorp/knirvserver/backend/internal/proto"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Hasher    HasherConfig `yaml:"hasher"`
	Source    SourceConfig `yaml:"source"`
	KNIRVBASE KNIRVConfig  `yaml:"knirvbase"`
}

type HasherConfig struct {
	Addr string `yaml:"addr"`
}

type SourceConfig struct {
	OrgID  string `yaml:"org_id"`
	UserID string `yaml:"user_id"`
}

type KNIRVConfig struct {
	DataDir string `yaml:"data_dir"`
}

func main() {
	config := LoadConfig()

	// Open the single shared KNIRVBASE instance for the whole pipeline.
	db, err := knirvbase.Open(config.KNIRVBASE.DataDir)
	if err != nil {
		log.Fatalf("open knirvbase: %v", err)
	}
	defer db.Close()

	client := grpc.NewClient(config.Hasher.Addr)
	defer client.Close()

	stream, err := client.ExportSecurityData(&hasherpb.ExportRequest{
		OrgId:     config.Source.OrgID,
		UserId:    config.Source.UserID,
		DataType:  hasherpb.DataType_ALL,
		Encrypted: true,
	})
	if err != nil {
		log.Fatalf("export stream: %v", err)
	}

	// MDWriter saves each decrypted chunk as a raw .md file in connector_raw.
	w := writer.NewMDWriter(db.Collection("connector_raw"))

	for chunk := range stream {
		if err := w.WriteChunk(chunk); err != nil {
			log.Printf("write chunk %s: %v", chunk.ChunkId, err)
		}
	}

	// Hand the open db to downstream stages (invoked sequentially or as
	// goroutines sharing the same db handle).
	// Note: miner and encoder are separate binaries in the pipeline
	log.Println("0_DATA_CONNECTOR completed. Downstream stages should be run separately.")
}

func LoadConfig() *Config {
	data, err := os.ReadFile("config/connector.yaml")
	if err != nil {
		log.Fatalf("read config: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	return &cfg
}
