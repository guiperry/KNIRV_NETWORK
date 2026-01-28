// cmd/pixie-client/main.go
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "asic-driver/internal/proto/pixie/v1"
)

var (
	serverAddr = flag.String("addr", "localhost:50051", "server address")
	mode       = flag.String("mode", "single", "operation mode: single, batch, stream, metrics, info")
	count      = flag.Int("count", 10, "number of hashes to compute")
	batchSize  = flag.Int("batch", 4, "batch size for batch mode")
	dataSize   = flag.Int("size", 64, "size of data to hash")
)

func main() {
	flag.Parse()

	// Connect to server
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewPixieServiceClient(conn)
	ctx := context.Background()

	switch *mode {
	case "single":
		runSingle(ctx, client)
	case "batch":
		runBatch(ctx, client)
	case "stream":
		runStream(ctx, client)
	case "metrics":
		showMetrics(ctx, client)
	case "info":
		showInfo(ctx, client)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func runSingle(ctx context.Context, client pb.PixieServiceClient) {
	log.Printf("Computing %d single hashes...", *count)

	totalLatency := time.Duration(0)

	for i := 0; i < *count; i++ {
		data := randomData(*dataSize)

		resp, err := client.ComputeHash(ctx, &pb.ComputeHashRequest{
			Data: data,
		})
		if err != nil {
			log.Fatalf("ComputeHash failed: %v", err)
		}

		totalLatency += time.Duration(resp.LatencyUs) * time.Microsecond

		if i == 0 {
			log.Printf("Hash #%d: %x (latency: %dus)", i+1, resp.Hash[:8], resp.LatencyUs)
		}
	}

	avgLatency := totalLatency / time.Duration(*count)
	log.Printf("Completed %d hashes, average latency: %v", *count, avgLatency)
}

func runBatch(ctx context.Context, client pb.PixieServiceClient) {
	log.Printf("Computing batch of %d hashes...", *count)

	// Prepare batch data
	data := make([][]byte, *count)
	for i := 0; i < *count; i++ {
		data[i] = randomData(*dataSize)
	}

	start := time.Now()

	resp, err := client.ComputeBatch(ctx, &pb.ComputeBatchRequest{
		Data:         data,
		MaxBatchSize: uint32(*batchSize),
	})
	if err != nil {
		log.Fatalf("ComputeBatch failed: %v", err)
	}

	elapsed := time.Since(start)

	log.Printf("Computed %d hashes in %v", resp.ProcessedCount, elapsed)
	log.Printf("Server reported latency: %dus", resp.TotalLatencyUs)
	log.Printf("Throughput: %.2f hashes/sec", float64(resp.ProcessedCount)/elapsed.Seconds())

	if len(resp.Hashes) > 0 {
		log.Printf("First hash: %x", resp.Hashes[0][:8])
	}
}

func runStream(ctx context.Context, client pb.PixieServiceClient) {
	log.Printf("Streaming %d hashes...", *count)

	stream, err := client.StreamCompute(ctx)
	if err != nil {
		log.Fatalf("StreamCompute failed: %v", err)
	}

	// Start receiver goroutine
	done := make(chan bool)
	var received int
	totalLatency := time.Duration(0)

	go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				close(done)
				return
			}

			received++
			totalLatency += time.Duration(resp.LatencyUs) * time.Microsecond

			if received == 1 || received%100 == 0 {
				log.Printf("Received hash #%d: %x (latency: %dus)",
					resp.RequestId, resp.Hash[:8], resp.LatencyUs)
			}
		}
	}()

	// Send requests
	start := time.Now()
	for i := 0; i < *count; i++ {
		data := randomData(*dataSize)

		err := stream.Send(&pb.StreamComputeRequest{
			Data:      data,
			RequestId: uint64(i + 1),
		})
		if err != nil {
			log.Fatalf("Send failed: %v", err)
		}
	}

	// Close send and wait for all responses
	stream.CloseSend()
	<-done

	elapsed := time.Since(start)
	avgLatency := totalLatency / time.Duration(received)

	log.Printf("Streamed %d hashes in %v", received, elapsed)
	log.Printf("Average latency: %v", avgLatency)
	log.Printf("Throughput: %.2f hashes/sec", float64(received)/elapsed.Seconds())
}

func showMetrics(ctx context.Context, client pb.PixieServiceClient) {
	resp, err := client.GetMetrics(ctx, &pb.GetMetricsRequest{})
	if err != nil {
		log.Fatalf("GetMetrics failed: %v", err)
	}

	fmt.Println("\n=== Pixie Metrics ===")
	fmt.Printf("Total Requests:       %d\n", resp.TotalRequests)
	fmt.Printf("Total Bytes Processed: %d (%.2f MB)\n",
		resp.TotalBytesProcessed,
		float64(resp.TotalBytesProcessed)/1024/1024)
	fmt.Printf("Average Latency:      %d µs\n", resp.AverageLatencyUs)
	fmt.Printf("Peak Latency:         %d µs\n", resp.PeakLatencyUs)
	fmt.Printf("Total Errors:         %d\n", resp.TotalErrors)
	fmt.Printf("Cache Hits:           %d\n", resp.CacheHits)
	fmt.Printf("Cache Misses:         %d\n", resp.CacheMisses)

	if len(resp.DeviceStats) > 0 {
		fmt.Println("\nDevice Stats:")
		for k, v := range resp.DeviceStats {
			fmt.Printf("  %s: %d\n", k, v)
		}
	}
}

func showInfo(ctx context.Context, client pb.PixieServiceClient) {
	resp, err := client.GetDeviceInfo(ctx, &pb.GetDeviceInfoRequest{})
	if err != nil {
		log.Fatalf("GetDeviceInfo failed: %v", err)
	}

	fmt.Println("\n=== Device Info ===")
	fmt.Printf("Device Path:      %s\n", resp.DevicePath)
	fmt.Printf("Chip Count:       %d\n", resp.ChipCount)
	fmt.Printf("Firmware Version: %s\n", resp.FirmwareVersion)
	fmt.Printf("Operational:      %v\n", resp.IsOperational)
	fmt.Printf("Uptime:           %d seconds (%.1f hours)\n",
		resp.UptimeSeconds,
		float64(resp.UptimeSeconds)/3600)
}

func randomData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}
