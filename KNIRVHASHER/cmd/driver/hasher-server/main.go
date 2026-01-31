// cmd/driver/hasher-server/main.go
// Hasher Server - runs on ASIC device and exposes gRPC service for hash computations
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"hasher/internal/driver/device"
	pb "hasher/internal/proto/hasher/v1"
)

var (
	port          = flag.Int("port", 50051, "gRPC server port")
	enableTracing = flag.Bool("trace", true, "enable eBPF tracing")
	enableTLS     = flag.Bool("tls", false, "enable TLS")
	certFile      = flag.String("cert", "", "TLS certificate file")
	keyFile       = flag.String("key", "", "TLS key file")
)

func main() {
	flag.Parse()

	log.Printf("Hasher Server starting on port %d...", *port)
	log.Printf("eBPF tracing: %v", *enableTracing)

	// Create gRPC server
	var opts []grpc.ServerOption

	if *enableTLS {
		if *certFile == "" || *keyFile == "" {
			log.Fatal("TLS enabled but cert/key files not provided")
		}
		// Add TLS credentials here
		// creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		// opts = append(opts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(opts...)

	// Create Hasher server using device package
	hasherServer, err := device.NewHasherServer(*enableTracing)
	if err != nil {
		log.Fatalf("Failed to create Hasher server: %v", err)
	}
	defer hasherServer.Close()

	// Register service
	pb.RegisterHasherServiceServer(grpcServer, hasherServer)

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Listen on TCP port
	addr := fmt.Sprintf(":%d", *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Hasher gRPC server starting on %s", addr)
	log.Printf("eBPF tracing: %v", *enableTracing)

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down server...")
		grpcServer.GracefulStop()
	}()

	// Start serving
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
