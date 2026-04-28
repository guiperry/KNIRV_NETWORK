package main

import (
	"context"
	"log"
	"os"
	"time"

	writer "data-connector/internal/writer"
	"github.com/knirvcorp/knirvbase/pkg/knirvbase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
	hasherpb "knirvhasher/proto/hasher/training/v1"
)

type Config struct {
	KNIRVSERVERAddr string      `yaml:"knirvserver_addr"`
	KNIRVBASE       KNIRVConfig `yaml:"knirvbase"`
	PollInterval    string      `yaml:"poll_interval"`
}

type KNIRVConfig struct {
	DataDir string `yaml:"data_dir"`
}

func main() {
	config := LoadConfig()

	// Open the single shared KNIRVBASE instance for the whole pipeline.
	db, err := knirvbase.New(context.Background(), knirvbase.Options{DataDir: config.KNIRVBASE.DataDir})
	if err != nil {
		log.Fatalf("open knirvbase: %v", err)
	}
	defer db.Shutdown()

	collection := db.Collection("connector_raw")
	w := writer.NewMDWriter(collection)

	// Connect to KNIRVSERVER gRPC server
	conn, err := grpc.NewClient(config.KNIRVSERVERAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect to knirvserver: %v", err)
	}
	defer conn.Close()

	client := hasherpb.NewHasherTrainingServiceClient(conn)

	pollInterval, _ := time.ParseDuration(config.PollInterval)
	if pollInterval == 0 {
		pollInterval = 1 * time.Hour
	}

	log.Printf("0_DATA_CONNECTOR: Connected to KNIRVSERVER at %s, polling every %s", config.KNIRVSERVERAddr, pollInterval)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		exportData(client, w)
		<-ticker.C
	}
}

func exportData(client hasherpb.HasherTrainingServiceClient, w *writer.MDWriter) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stream, err := client.ExportSecurityData(ctx, &hasherpb.ExportRequest{
		OrgId:    "default",
		UserId:    "",
		DataType: hasherpb.DataType_ALL,
		Encrypted: true,
	})
	if err != nil {
		log.Printf("start export: %v", err)
		return
	}

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("receive chunk: %v", err)
			}
			break
		}

		if err := w.WriteChunk(chunk); err != nil {
			log.Printf("write chunk %s: %v", chunk.ChunkId, err)
		}
	}

	log.Printf("export completed")
}

func LoadConfig() *Config {
	data, err := os.ReadFile("config/connector.yaml")
	if err != nil {
		// Return default config
		return &Config{
			KNIRVSERVERAddr: "localhost:50051",
			KNIRVBASE:       KNIRVConfig{DataDir: "./data"},
			PollInterval:    "1h",
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	return &cfg
}
