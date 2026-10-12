package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/lab/hasher/data-seeder/internal/config"
	"github.com/lab/hasher/data-seeder/pkg/simulator"
	"github.com/lab/hasher/data-seeder/pkg/training"
)

func main() {
	var (
		deviceAddr    = flag.String("device-addr", "", "hasher-server gRPC address, e.g. 192.168.1.50:8888 (required unless --hash-method=software)")
		hashMethod    = flag.String("hash-method", "software", "Hash method to use: auto, asic, direct, software, cuda")
		population    = flag.Int("population", 256, "Population size for evolution (matches data-seeder default)")
		generations   = flag.Int("generations", 50, "Number of generations to run (deliberately lower than data-seeder's 2000)")
		difficulty    = flag.Int("difficulty-bits", config.DefaultDifficultyBits, fmt.Sprintf("Number of leading bits that must match (%d-%d)", config.MinDifficultyBits, config.MaxDifficultyBits))
		targetToken   = flag.Int("target-token", 42, "Target token ID for the synthetic training record")
		verbose       = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	if *hashMethod != "software" && *deviceAddr == "" {
		log.Fatalf("error: --device-addr is required when --hash-method is %q (use --hash-method=software to run without hardware)", *hashMethod)
	}

	if *deviceAddr != "" {
		os.Setenv("DEVICE_IP", *deviceAddr)
		log.Printf("DEVICE_IP set to %s", *deviceAddr)
	}

	if *hashMethod != "software" && *hashMethod != "direct" && *hashMethod != "asic" && *hashMethod != "cuda" && *hashMethod != "auto" {
		log.Fatalf("error: invalid --hash-method %q (must be auto, asic, direct, software, or cuda)", *hashMethod)
	}

	log.Printf("seed-bench: hash-method=%s population=%d generations=%d difficulty=%d target-token=%d",
		*hashMethod, *population, *generations, *difficulty, *targetToken)

	methods := []string{"direct", "asic", "cuda"}
	isHardwareMethod := false
	for _, m := range methods {
		if *hashMethod == m {
			isHardwareMethod = true
			break
		}
	}

	simConfig := &simulator.SimulatorConfig{
		DeviceType:     fmt.Sprintf("hasher_%s", *hashMethod),
		MaxConcurrency: 100,
		TargetHashRate: 500000000,
		CacheSize:      10000,
		GPUDevice:      0,
		Timeout:        30,
	}

	sim := simulator.NewHasherWrapper(simConfig)
	if err := sim.Initialize(simConfig); err != nil {
		if isHardwareMethod {
			log.Fatalf("error: hash method %q failed to initialize: %v (set --device-addr correctly and ensure hasher-server is running)", *hashMethod, err)
		}
		log.Fatalf("error: simulator failed to initialize: %v", err)
	}
	defer sim.Shutdown()

	log.Printf("Using hash method: %s", sim.GetHashMethod().Name())

	harness := training.NewEvolutionaryHarness(*population)
	if *difficulty >= config.MinDifficultyBits && *difficulty <= config.MaxDifficultyBits {
		mask := uint32(0xFFFFFFFF) << (32 - *difficulty)
		harness.SetDifficultyMask(mask)
		log.Printf("Difficulty set to %d bits (mask: 0x%08X)", *difficulty, mask)
	} else {
		log.Printf("Invalid difficulty bits: %d (must be %d-%d), using default",
			*difficulty, config.MinDifficultyBits, config.MaxDifficultyBits)
	}

	contextHash := training.ComputeContextHash([]int32{int32(*targetToken)}, 5)
	record := &training.TrainingRecord{
		SchemaVersion: 2,
		SourceFile:    "synthetic",
		TargetToken:   int32(*targetToken),
		TokenSequence: []int32{int32(*targetToken)},
		AssertionSpan: []int32{int32(*targetToken)},
		ContextHash:   contextHash,
		FeatureVector: [12]uint32{},
	}

	tokenMap := map[int32]bool{int32(*targetToken): true}

	log.Printf("Starting benchmark: %d generations x population %d", *generations, *population)
	log.Printf("Effective ComputeDoubleHash calls/generation = %d x 21 = %d", *population, *population*21)

	var genLatencies []time.Duration
	start := time.Now()

	for gen := 0; gen < *generations; gen++ {
		genStart := time.Now()
		results, err := harness.EvaluatePopulation(
			training.NewSeedPopulation(int32(*targetToken), contextHash, *population),
			record,
			tokenMap,
			sim,
		)
		genLatency := time.Since(genStart)

		if err != nil {
			log.Fatalf("error: generation %d failed: %v", gen, err)
		}

		genLatencies = append(genLatencies, genLatency)

		if *verbose || gen == 0 || gen == *generations-1 {
			var bestAdvantage float64
			for _, r := range results {
				if r.Advantage > bestAdvantage {
					bestAdvantage = r.Advantage
				}
			}
			log.Printf("[gen %d/%d] latency=%v best_advantage=%.4f",
				gen+1, *generations, genLatency, bestAdvantage)
		}
	}

	totalWall := time.Since(start)
	totalCalls := int64(*generations) * int64(*population) * 21

	sort.Slice(genLatencies, func(i, j int) bool {
		return genLatencies[i] < genLatencies[j]
	})

	var min, max, sum time.Duration
	for i, lat := range genLatencies {
		if i == 0 {
			min = lat
		}
		if lat > max {
			max = lat
		}
		sum += lat
	}
	mean := sum / time.Duration(len(genLatencies))
	p50 := genLatencies[len(genLatencies)/2]
	p95Idx := int(float64(len(genLatencies)) * 0.95)
	if p95Idx >= len(genLatencies) {
		p95Idx = len(genLatencies) - 1
	}
	p95 := genLatencies[p95Idx]

	fmt.Println()
	fmt.Println("=== seed-bench results ===")
	fmt.Printf("Hash method:      %s\n", *hashMethod)
	fmt.Printf("Population:       %d\n", *population)
	fmt.Printf("Generations:      %d\n", *generations)
	fmt.Printf("Difficulty bits:  %d\n", *difficulty)
	fmt.Println("Per-generation latency:")
	fmt.Printf("  min:    %v\n", min)
	fmt.Printf("  p50:    %v\n", p50)
	fmt.Printf("  p95:    %v\n", p95)
	fmt.Printf("  max:    %v\n", max)
	fmt.Printf("  mean:   %v\n", mean)
	fmt.Println()
	fmt.Printf("Total wall time:  %v\n", totalWall)
	fmt.Printf("Total ComputeDoubleHash calls: %d\n", totalCalls)
	if totalWall.Seconds() > 0 {
		fmt.Printf("Effective ComputeDoubleHash calls/sec: %.1f\n", float64(totalCalls) / totalWall.Seconds())
	}
	fmt.Printf("Calls/generation: %d x 21 = %d\n", *population, *population*21)
}
