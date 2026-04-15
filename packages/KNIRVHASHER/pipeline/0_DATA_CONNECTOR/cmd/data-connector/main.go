package main

import (
	"context"
	"log"
	"net"
	"os"

	writer "data-connector/internal/writer"
	"github.com/knirvcorp/knirvbase/pkg/knirvbase"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
	hasherpb "knirvhasher/proto"
)

type Config struct {
	ListenAddr string      `yaml:"listen_addr"`
	KNIRVBASE  KNIRVConfig `yaml:"knirvbase"`
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

	// Create gRPC server
	server := grpc.NewServer()
	hasherService := &ConnectorService{
		writer: writer.NewMDWriter(collection),
	}

	hasherpb.RegisterHasherServiceServer(server, hasherService)

	// Listen on the configured address
	lis, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	log.Printf("0_DATA_CONNECTOR listening on %s", config.ListenAddr)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// ConnectorService implements the HasherService server
type ConnectorService struct {
	hasherpb.UnimplementedHasherServiceServer
	writer *writer.MDWriter
}

// ExportSecurityData receives encrypted data chunks and writes them to knirvbase
func (cs *ConnectorService) ExportSecurityData(stream hasherpb.HasherService_ExportSecurityDataServer) error {
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}

		if err := cs.writer.WriteChunk(chunk); err != nil {
			log.Printf("write chunk %s: %v", chunk.ChunkId, err)
			continue
		}
	}

	return stream.SendAndClose(&hasherpb.ExportResponse{
		Status:  "success",
		Message: "Data exported successfully",
	})
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
