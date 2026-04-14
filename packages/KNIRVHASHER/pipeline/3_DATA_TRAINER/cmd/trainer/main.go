package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"knirvbase"
	evo_grpo "knirvhasher/pipeline/3_DATA_TRAINER/internal/evo_grpo"
	gates "knirvhasher/pipeline/3_DATA_TRAINER/internal/gates"
	knowledge_base "knirvhasher/pipeline/3_DATA_TRAINER/internal/knowledge_base"
)

func main() {
	var inputDir, outputDir string
	flag.StringVar(&inputDir, "input", "./data/input", "Input directory containing .nrv files")
	flag.StringVar(&outputDir, "output", "./data/output", "Output directory for trained models")
	flag.Parse()

	// Initialize knirvbase connection
	kvbase, err := knirvbase.NewCollection(context.Background(), "knirvhasher_trainer")
	if err != nil {
		log.Fatalf("Failed to connect to knirvbase: %v", err)
	}
	defer kvbase.Close()

	// Initialize components
	gateTrainer := gates.NewUserSecurityGates(kvbase)
	evoTrainer := evo_grpo.NewEvoGRPO(kvbase)
	kbIndexer := knowledge_base.NewNRVKnowledgeBase(kvbase)

	// Process all .nrv files in input directory
	err = filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".nrv" {
			return processNRVFile(path, outputDir, gateTrainer, evoTrainer, kbIndexer)
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to process files: %v", err)
	}

	fmt.Println("Training completed successfully")
}

func processNRVFile(path, outputDir string, gateTrainer *gates.UserSecurityGates,
	evoTrainer *evo_grpo.EvoGRPO, kbIndexer *knowledge_base.NRVKnowledgeBase) error {

	fmt.Printf("Processing %s\n", path)

	// Read NRV bracket data
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file %s: %w", path, err)
	}

	// Parse NRV data
	nrvData, err := parseNRVData(data)
	if err != nil {
		return fmt.Errorf("parse NRV data: %w", err)
	}

	// Train user security gates
	gateModel, err := gateTrainer.Train(nrvData.UserID, nrvData.Brackets)
	if err != nil {
		return fmt.Errorf("train gates: %w", err)
	}

	// Run Evo-GRPO optimization
	optimizedModel, err := evoTrainer.Optimize(gateModel)
	if err != nil {
		return fmt.Errorf("optimize model: %w", err)
	}

	// Re-index knowledge base
	err = kbIndexer.ReIndex(nrvData.UserID, optimizedModel)
	if err != nil {
		return fmt.Errorf("re-index knowledge base: %w", err)
	}

	// Save trained model
	modelPath := filepath.Join(outputDir, fmt.Sprintf("%s_model.nrv", nrvData.UserID))
	err = saveModel(modelPath, optimizedModel)
	if err != nil {
		return fmt.Errorf("save model: %w", err)
	}

	fmt.Printf("Completed processing %s\n", path)
	return nil
}

type NRVData struct {
	UserID   string
	Brackets [][]byte
}

func parseNRVData(data []byte) (*NRVData, error) {
	// Placeholder: Parse NRV bracket format
	// In real implementation, this would decode the NRV binary format
	return &NRVData{
		UserID:   "user123", // Placeholder
		Brackets: [][]byte{data},
	}, nil
}

func saveModel(path string, model interface{}) error {
	// Placeholder: Save model to file
	return os.WriteFile(path, []byte(fmt.Sprintf("%v", model)), 0644)
}
