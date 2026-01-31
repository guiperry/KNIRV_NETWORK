package cortex

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// InferenceProvider is an interface for the actual LLM backend (e.g., LoRAX, vLLM)
type InferenceProvider interface {
	StreamInference(ctx context.Context, traitID string, prompt string) (<-chan string, error)
}

// Gateway implements the CortexService gRPC server
type Gateway struct {
	UnimplementedCortexServiceServer
	Provider InferenceProvider
	
	// Management for background "Deep Learning Sessions"
	trainingQueue chan *TrainingData
	activeJobs    sync.Map
}

func NewGateway(provider InferenceProvider) *Gateway {
	g := &Gateway{
		Provider:      provider,
		trainingQueue: make(chan *TrainingData, 100),
	}
	// Start the background training worker
	go g.trainingWorker()
	return g
}

// GenerateStream handles real-time requests from cortex.wasm
func (g *Gateway) GenerateStream(req *InferenceRequest, stream CortexService_GenerateStreamServer) error {
	log.Printf("Requesting Trait: %s", req.TraitId)

	// Call the underlying LLM provider (this is where the Hot-Swap happens)
	tokens, err := g.Provider.StreamInference(stream.Context(), req.TraitId, req.Prompt)
	if err != nil {
		return fmt.Errorf("provider error: %v", err)
	}

	for token := range tokens {
		if err := stream.Send(&InferenceResponse{Token: token}); err != nil {
			return err
		}
	}

	return stream.Send(&InferenceResponse{IsFinal: true})
}

// SyncTrainingData receives batches for the "Deep Learning Session"
func (g *Gateway) SyncTrainingData(stream CortexService_SyncTrainingDataServer) error {
	for {
		data, err := stream.Recv()
		if err != nil {
			return stream.SendAndClose(&TrainingStatus{Status: "SYNC_COMPLETE"})
		}
		
		// Queue for background training without blocking the user
		g.trainingQueue <- data
	}
}

// trainingWorker runs in the background
func (g *Gateway) trainingWorker() {
	for data := range g.trainingQueue {
		log.Printf("Starting Deep Learning Session for Trait: %s", data.TraitId)
		// Here you would trigger an external Python script (using os/exec) 
		// or call a specialized training API (like Predibase or Anyscale)
		// e.g., exec.Command("python3", "train_lora.py", "--data", ...).Run()
	}
}