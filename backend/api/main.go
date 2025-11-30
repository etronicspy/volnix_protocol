package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
)

var (
	grpcAddr = flag.String("grpc-addr", "localhost:9090", "gRPC server address")
	httpAddr = flag.String("http-addr", "0.0.0.0:1317", "HTTP server address")
)

func main() {
	flag.Parse()

	// Try to connect to gRPC server (non-blocking)
	var consensusClient consensusv1.QueryClient
	conn, err := grpc.NewClient(*grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("⚠️  Warning: Failed to connect to gRPC server at %s: %v", *grpcAddr, err)
		log.Printf("⚠️  REST API will start, but gRPC endpoints will return errors")
		log.Printf("⚠️  Make sure gRPC server is running on %s", *grpcAddr)
		consensusClient = nil
	} else {
		defer conn.Close()
		consensusClient = consensusv1.NewQueryClient(conn)
		log.Printf("✅ Connected to gRPC server at %s", *grpcAddr)
	}

	// Create HTTP server
	server := NewServer(consensusClient)

	// Setup HTTP routes
	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    *httpAddr,
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting REST API server on %s", *httpAddr)
		log.Printf("Connected to gRPC server at %s", *grpcAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

