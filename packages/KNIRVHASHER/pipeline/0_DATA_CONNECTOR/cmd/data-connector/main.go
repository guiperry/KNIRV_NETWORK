package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"data-connector/internal/connector"
	writer "data-connector/internal/writer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
	hasherpb "knirvhasher/proto/hasher/training/v1"
)

type Config struct {
	KNIRVSERVERAddr     string         `yaml:"knirvserver_addr"` // HTTP ontology API
	KNIRVSERVERGRPCAddr string         `yaml:"knirvserver_grpc_addr"`
	OutputDir           string         `yaml:"output_dir"`
	PollInterval        string         `yaml:"poll_interval"`
	Ontology            OntologyConfig `yaml:"ontology"`
	HuggingFace         HFConfig       `yaml:"huggingface"`
	Arxiv               ArxivConfig    `yaml:"arxiv"`
}

type OntologyConfig struct {
	MinEntityCount int64 `yaml:"min_entity_count"`
}
type HFConfig struct {
	Datasets     []string `yaml:"datasets"`
	Splits       []string `yaml:"splits"`
	MaxRows      int      `yaml:"max_rows_per_dataset"`
	Token        string   `yaml:"token"`
	RequestDelay string   `yaml:"request_delay"`
	CacheDir     string   `yaml:"cache_dir"`
}
type ArxivConfig struct {
	Categories           []string `yaml:"categories"`
	MaxPapers            int      `yaml:"max_papers"`
	DownloadDelaySeconds int      `yaml:"download_delay_seconds"`
}

func main() {
	config := LoadConfig()
	force := flag.String("source", "", "force ontology, huggingface, or arxiv")
	flag.Parse()
	if err := runPriority(config, connector.Tier(*force)); err != nil {
		log.Fatalf("source-priority ingest: %v", err)
	}
	if os.Getenv("KNIRV_SECURITY_EXPORT") != "1" {
		return
	}
	w := writer.NewMDWriter(filepath.Join(config.OutputDir, "security"))

	// Connect to KNIRVSERVER gRPC server
	conn, err := grpc.NewClient(config.KNIRVSERVERGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect to knirvserver: %v", err)
	}
	defer conn.Close()

	client := hasherpb.NewHasherTrainingServiceClient(conn)

	pollInterval, _ := time.ParseDuration(config.PollInterval)
	if pollInterval == 0 {
		pollInterval = 1 * time.Hour
	}

	log.Printf("0_DATA_CONNECTOR: Connected to KNIRVSERVER gRPC at %s, polling every %s", config.KNIRVSERVERGRPCAddr, pollInterval)

	exportData(client, w)
}

func runPriority(config *Config, override connector.Tier) error {
	base := "http://" + config.KNIRVSERVERAddr
	if strings.HasPrefix(config.KNIRVSERVERAddr, "http://") || strings.HasPrefix(config.KNIRVSERVERAddr, "https://") {
		base = config.KNIRVSERVERAddr
	}
	state := filepath.Join(config.OutputDir, "source_state.json")
	runner := &connector.PriorityRunner{StatePath: state, Override: override,
		Ontology: func() ([]connector.RawRecord, bool, error) {
			available, err := connector.OntologyAvailable(nil, base, config.Ontology.MinEntityCount)
			if err != nil || !available {
				return nil, true, err
			}
			rows, err := (&connector.OntologyConnector{BaseURL: base}).FetchEntities()
			return rows, true, err
		},
		HuggingFace: func() ([]connector.RawRecord, bool, error) {
			datasets := config.HuggingFace.Datasets
			if len(datasets) == 0 {
				datasets = []string{"tatsu-lab/alpaca"}
			}
			splits := config.HuggingFace.Splits
			if len(splits) == 0 {
				splits = []string{"train"}
			}
			var all []connector.RawRecord
			requestDelay, err := time.ParseDuration(config.HuggingFace.RequestDelay)
			if err != nil || requestDelay < 0 {
				requestDelay = 250 * time.Millisecond
			}
			for _, ds := range datasets {
				client := &connector.HuggingFaceConnector{Config: connector.HuggingFaceConfig{
					MaxRowsPerDS: config.HuggingFace.MaxRows,
					CacheDir:     config.HuggingFace.CacheDir,
					Token:        config.HuggingFace.Token,
				}}
				if client.Config.Token == "" {
					client.Config.Token = os.Getenv("HF_TOKEN")
				}
				limit := config.HuggingFace.MaxRows
				if limit <= 0 {
					limit = 50000
				}
				for offset := 0; offset < limit; offset += 100 {
					length := 100
					if remaining := limit - offset; remaining < length {
						length = remaining
					}
					rows, exhausted, err := client.Page(ds, splits[0], offset, length)
					if err != nil {
						return nil, false, err
					}
					all = append(all, rows...)
					if exhausted || len(rows) < length {
						break
					}
					if requestDelay > 0 {
						time.Sleep(requestDelay)
					}
				}
			}
			return all, len(all) == 0, nil
		},
		Arxiv: func() ([]connector.RawRecord, bool, error) {
			if config.Arxiv.DownloadDelaySeconds > 0 {
				time.Sleep(time.Duration(config.Arxiv.DownloadDelaySeconds) * time.Second)
			}
			return connector.FetchNextArxiv(
				nil,
				config.Arxiv.Categories,
				config.Arxiv.MaxPapers,
				filepath.Join(config.OutputDir, "arxiv_cursor.json"),
			)
		},
	}
	records, tier, err := runner.Run()
	if err != nil {
		return err
	}
	log.Printf("0_DATA_CONNECTOR: selected %s, records=%d", tier, len(records))
	// Always replace the hand-off file, including on source exhaustion. Leaving
	// the previous paper in place would make the mapper process it forever.
	return connector.WriteRecords(filepath.Join(config.OutputDir, "records.jsonl"), records)
}

func exportData(client hasherpb.HasherTrainingServiceClient, w *writer.MDWriter) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stream, err := client.ExportSecurityData(ctx, &hasherpb.ExportRequest{
		OrgId:     "default",
		UserId:    "",
		DataType:  hasherpb.DataType_ALL,
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
	cfg := &Config{
		KNIRVSERVERAddr:     "http://localhost:8084",
		KNIRVSERVERGRPCAddr: "localhost:50051",
		OutputDir:           "~/.local/share/knirvhasher/connector",
		PollInterval:        "1h",
		Ontology:            OntologyConfig{MinEntityCount: 1},
		HuggingFace:         HFConfig{Datasets: []string{"tatsu-lab/alpaca"}, Splits: []string{"train"}, MaxRows: 50000, RequestDelay: "1s", CacheDir: "~/.cache/knirvhasher/hf"},
		Arxiv:               ArxivConfig{Categories: []string{"cs.LG", "cs.CL"}, MaxPapers: 50, DownloadDelaySeconds: 2},
	}

	data, err := os.ReadFile("config/connector.yaml")
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			log.Printf("Warning: failed to parse config: %v", err)
		}
	}

	// Environment variable overrides
	if addr := os.Getenv("HASHER_GRPC_SOCKET_PATH"); addr != "" {
		// If provided as a raw path, prefix with unix:// for gRPC
		if !strings.HasPrefix(addr, "unix://") && !strings.HasPrefix(addr, "passthrough://") {
			cfg.KNIRVSERVERGRPCAddr = "unix://" + addr
		} else {
			cfg.KNIRVSERVERGRPCAddr = addr
		}
	} else if addr := os.Getenv("KNIRVSERVER_GRPC_ADDR"); addr != "" {
		cfg.KNIRVSERVERGRPCAddr = addr
	}
	if addr := os.Getenv("KNIRVSERVER_ADDR"); addr != "" {
		cfg.KNIRVSERVERAddr = addr
	}

	if outputDir := os.Getenv("KNIRV_CONNECTOR_DIR"); outputDir != "" {
		cfg.OutputDir = outputDir
	} else if appDataDir := os.Getenv("KNIRV_APP_DATA_DIR"); appDataDir != "" {
		cfg.OutputDir = filepath.Join(appDataDir, "knirvhasher", "data", "connector")
	}
	if strings.HasPrefix(cfg.OutputDir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			cfg.OutputDir = filepath.Join(home, cfg.OutputDir[2:])
		}
	}

	return cfg
}
