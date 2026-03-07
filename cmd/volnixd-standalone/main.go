package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"cosmossdk.io/log"

	"github.com/volnix-protocol/volnix-protocol/app"
)

// StandaloneServer wraps the app minimal server for use by monitoring and tests
type StandaloneServer struct {
	*app.MinimalVolnixServer
}

// NewStandaloneServer creates a standalone node server
func NewStandaloneServer(homeDir string, logger log.Logger) (*StandaloneServer, error) {
	s, err := app.NewMinimalCometBFTServer(homeDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create minimal server: %w", err)
	}
	return &StandaloneServer{MinimalVolnixServer: s}, nil
}

func main() {
	homeDir := flag.String("home", defaultHome(), "Home directory for node data and config")
	flag.Parse()

	logger := log.NewLogger(os.Stdout)
	logger.Info("Volnix Protocol standalone node", "home", *homeDir)

	server, err := NewStandaloneServer(*homeDir, logger)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		logger.Info("Shutdown signal received")
		cancel()
	}()

	if err := server.MinimalVolnixServer.Start(ctx); err != nil && ctx.Err() == nil {
		logger.Error("Server stopped with error", "error", err)
		os.Exit(1)
	}
	logger.Info("Node stopped")
}

func defaultHome() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(dir, ".volnix")
}
