package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/knirv/hasher/internal/proto/v1"
	"github.com/knirv/hasher/pipeline/0_DATA_CONNECTOR/internal/cleaner"
	"github.com/knirv/hasher/pipeline/0_DATA_CONNECTOR/internal/encoder"
	"github.com/knirv/hasher/pipeline/0_DATA_CONNECTOR/internal/grpc"
	"github.com/knirv/hasher/pipeline/0_DATA_CONNECTOR/internal/normalizer"
	"github.com/knirv/hasher/pipeline/0_DATA_CONNECTOR/internal/writer"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Hasher     HasherConfig     `yaml:"hasher"`
	Source     SourceConfig     `yaml:"source"`
	Output     OutputConfig     `yaml:"output"`
	Processing ProcessingConfig `yaml:"processing"`
	Security   SecurityConfig   `yaml:"security"`
}

type HasherConfig struct {
	Socket  string `yaml:"socket"`
	Timeout int    `yaml:"timeout"`
}

type SourceConfig struct {
	OrgID     string   `yaml:"org_id"`
	UserID    string   `yaml:"user_id"`
	DataTypes []string `yaml:"data_types"`
}

type OutputConfig struct {
	ArrowDir  string `yaml:"arrow_dir"`
	JSONDir   string `yaml:"json_dir"`
	BatchSize int    `yaml:"batch_size"`
}

type ProcessingConfig struct {
	MaxConcurrent int  `yaml:"max_concurrent"`
	PIIScrub      bool `yaml:"pii_scrub"`
	Deduplicate   bool `yaml:"deduplicate"`
}

type SecurityConfig struct {
	PQCKeyID string `yaml:"pqc_key_id"`
}

func main() {
	configPath := flag.String("config", "config/connector.yaml", "Path to config file")
	mode := flag.String("mode", "stream", "Mode: stream, batch, once")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := setupOutputDirs(cfg); err != nil {
		log.Fatalf("Failed to setup output dirs: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		cancel()
	}()

	client, err := grpc.NewClient(cfg.Hasher.Socket)
	if err != nil {
		log.Fatalf("Failed to connect to hasher service: %v", err)
	}
	defer client.Close()

	normalizer := normalizer.NewSecurityNormalizer(cfg.Processing.PIIScrub)
	arrowEncoder := encoder.NewArrowEncoder()
	jsonEncoder := encoder.NewJSONEncoder()
	cleaner := cleaner.NewDataCleaner(cfg.Processing.Deduplicate)
	writer := writer.NewFileWriter(cfg.Output.ArrowDir, cfg.Output.JSONDir)

	log.Printf("Starting connector in %s mode", *mode)
	log.Printf("Output dirs: arrow=%s, json=%s", cfg.Output.ArrowDir, cfg.Output.JSONDir)

	switch *mode {
	case "stream":
		runStreamMode(ctx, client, cfg, normalizer, arrowEncoder, jsonEncoder, cleaner, writer)
	case "batch":
		runBatchMode(ctx, client, cfg, normalizer, arrowEncoder, jsonEncoder, cleaner, writer)
	case "once":
		runOnceMode(ctx, client, cfg, normalizer, arrowEncoder, jsonEncoder, cleaner, writer)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}

	log.Println("Connector shutdown complete")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

func setupOutputDirs(cfg *Config) error {
	if err := os.MkdirAll(cfg.Output.ArrowDir, 0755); err != nil {
		return fmt.Errorf("arrow dir: %w", err)
	}
	if err := os.MkdirAll(cfg.Output.JSONDir, 0755); err != nil {
		return fmt.Errorf("json dir: %w", err)
	}
	return nil
}

func runStreamMode(ctx context.Context, client *grpc.Client, cfg *Config, norm *normalizer.SecurityNormalizer, arrowEnc *encoder.ArrowEncoder, jsonEnc *encoder.JSONEncoder, clean *cleaner.DataCleaner, writer *writer.FileWriter) {
	dataType := v1.DataType_ALL
	if len(cfg.Source.DataTypes) > 0 && cfg.Source.DataTypes[0] != "ALL" {
		dataType = v1.DataType_value[cfg.Source.DataTypes[0]]
	}

	req := &v1.ExportRequest{
		OrgId:     cfg.Source.OrgID,
		UserId:    cfg.Source.UserID,
		DataType:  dataType,
		Encrypted: true,
	}

	stream, err := client.ExportSecurityData(ctx, req)
	if err != nil {
		log.Fatalf("Failed to start export: %v", err)
	}

	batch := make([]*normalizer.SecurityRecord, 0, cfg.Output.BatchSize)
	chunkCount := 0

	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				flushBatch(batch, arrowEnc, jsonEnc, writer)
			}
			return
		default:
		}

		chunk, err := stream.Recv()
		if err != nil {
			log.Printf("Stream ended: %v", err)
			if len(batch) > 0 {
				flushBatch(batch, arrowEnc, jsonEnc, writer)
			}
			return
		}

		chunkCount++
		log.Printf("Received chunk %d (id=%s, last=%v)", chunkCount, chunk.ChunkId, chunk.IsLast)

		records, err := norm.Process(chunk.Data)
		if err != nil {
			log.Printf("Normalize error: %v", err)
			continue
		}

		records = clean.Clean(records)
		batch = append(batch, records...)

		if len(batch) >= cfg.Output.BatchSize {
			flushBatch(batch, arrowEnc, jsonEnc, writer)
			batch = batch[:0]
		}
	}
}

func runBatchMode(ctx context.Context, client *grpc.Client, cfg *Config, norm *normalizer.SecurityNormalizer, arrowEnc *encoder.ArrowEncoder, jsonEnc *encoder.JSONEncoder, clean *cleaner.DataCleaner, writer *writer.FileWriter) {
	dataType := v1.DataType_ALL
	req := &v1.ExportRequest{
		OrgId:     cfg.Source.OrgID,
		UserId:    cfg.Source.UserID,
		DataType:  dataType,
		Encrypted: true,
	}

	allRecords := make([]*normalizer.SecurityRecord, 0)

	stream, err := client.ExportSecurityData(ctx, req)
	if err != nil {
		log.Fatalf("Failed to start export: %v", err)
	}

	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}

		records, _ := norm.Process(chunk.Data)
		records = clean.Clean(records)
		allRecords = append(allRecords, records...)

		if chunk.IsLast {
			break
		}
	}

	log.Printf("Collected %d records, writing...", len(allRecords))
	flushBatch(allRecords, arrowEnc, jsonEnc, writer)
}

func runOnceMode(ctx context.Context, client *grpc.Client, cfg *Config, norm *normalizer.SecurityNormalizer, arrowEnc *encoder.ArrowEncoder, jsonEnc *encoder.JSONEncoder, clean *cleaner.DataCleaner, writer *writer.FileWriter) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Hasher.Timeout)*time.Second)
	defer cancel()

	runBatchMode(ctx, client, cfg, norm, arrowEnc, jsonEnc, clean, writer)
}

func flushBatch(records []*normalizer.SecurityRecord, arrowEnc *encoder.ArrowEncoder, jsonEnc *encoder.JSONEncoder, writer *writer.FileWriter) {
	filename := fmt.Sprintf("knirv_%d", time.Now().UnixNano())

	arrowPath, err := writer.WriteArrow(records, filename)
	if err != nil {
		log.Printf("Arrow write error: %v", err)
	} else {
		log.Printf("Wrote %d records to %s", len(records), arrowPath)
	}

	jsonPath, err := writer.WriteJSON(records, filename)
	if err != nil {
		log.Printf("JSON write error: %v", err)
	} else {
		log.Printf("Wrote %d records to %s", len(records), jsonPath)
	}
}
