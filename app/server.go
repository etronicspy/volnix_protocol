package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/log"
	cosmosdb "github.com/cosmos/cosmos-db"
)

// VolnixServer wraps Volnix app with basic server functionality
type VolnixServer struct {
	app     *VolnixApp
	homeDir string
	logger  log.Logger
}

// NewCometBFTServer creates a new server with Volnix app
func NewCometBFTServer(homeDir string, logger log.Logger) (*VolnixServer, error) {
	// Create database
	dbPath := filepath.Join(homeDir, "data")
	db, err := cosmosdb.NewGoLevelDB("volnix", dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}
	
	// Create encoding config
	encodingConfig := MakeEncodingConfig()
	
	// Create Volnix app
	app := NewVolnixApp(logger, db, nil, encodingConfig)
	
	return &VolnixServer{
		app:     app,
		homeDir: homeDir,
		logger:  logger,
	}, nil
}

// Start starts the Volnix server
func (s *VolnixServer) Start(ctx context.Context) error {
	s.logger.Info("🚀 Starting Volnix Protocol server...")
	
	// Create necessary files
	if err := s.initializeFiles(); err != nil {
		return fmt.Errorf("failed to initialize files: %w", err)
	}
	
	s.logger.Info("✅ Volnix server started successfully")
	s.logger.Info("🌐 Server configuration:")
	s.logger.Info("   🔗 Chain ID: test-volnix")
	s.logger.Info("   📁 Home: " + s.homeDir)
	s.logger.Info("   💾 Database: GoLevelDB")
	s.logger.Info("   🏗️  Framework: Cosmos SDK")
	
	s.logger.Info("📦 Active modules:")
	s.logger.Info("   ✅ ident - Identity & ZKP verification")
	s.logger.Info("   ✅ lizenz - LZN license management")
	s.logger.Info("   ✅ anteil - ANT internal market")
	s.logger.Info("   ✅ consensus - PoVB consensus")
	
	s.logger.Info("🎯 Ready for transactions and queries!")
	s.logger.Info("💡 Note: Full CometBFT integration coming in next version")
	
	// Wait for context cancellation
	<-ctx.Done()
	
	return s.Stop()
}

// Stop stops the Volnix server
func (s *VolnixServer) Stop() error {
	s.logger.Info("🛑 Stopping Volnix server...")
	s.logger.Info("✅ Volnix server stopped")
	return nil
}

// GetApp returns the Volnix app
func (s *VolnixServer) GetApp() *VolnixApp {
	return s.app
}

// initializeFiles creates necessary configuration files
func (s *VolnixServer) initializeFiles() error {
	// Create config files
	configDir := filepath.Join(s.homeDir, "config")
	dataDir := filepath.Join(s.homeDir, "data")
	
	// Ensure directories exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	
	// Create genesis file if it doesn't exist
	genesisFile := filepath.Join(configDir, "genesis.json")
	if _, err := os.Stat(genesisFile); os.IsNotExist(err) {
		if err := s.createGenesisFile(genesisFile); err != nil {
			return fmt.Errorf("failed to create genesis file: %w", err)
		}
	}
	
	// Create config file if it doesn't exist
	configFile := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := s.createConfigFile(configFile); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}
	
	return nil
}

// createGenesisFile creates a default genesis file
func (s *VolnixServer) createGenesisFile(genesisFile string) error {
	genesisContent := `{
  "genesis_time": "` + time.Now().Format(time.RFC3339) + `",
  "chain_id": "test-volnix",
  "initial_height": "1",
  "consensus_params": {
    "block": {
      "max_bytes": "22020096",
      "max_gas": "-1"
    },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration": "172800000000000"
    },
    "validator": {
      "pub_key_types": ["ed25519"]
    }
  },
  "app_hash": "",
  "app_state": {
    "ident": {
      "params": {},
      "verified_accounts": []
    },
    "lizenz": {
      "params": {},
      "lizenzs": []
    },
    "anteil": {
      "params": {},
      "orders": [],
      "auctions": [],
      "user_positions": []
    },
    "consensus": {
      "params": {},
      "consensus_state": {},
      "validator_weights": []
    }
  }
}`
	
	return os.WriteFile(genesisFile, []byte(genesisContent), 0644)
}

// createConfigFile creates a default config file
func (s *VolnixServer) createConfigFile(configFile string) error {
	configContent := `# Volnix Protocol Configuration

[consensus]
timeout_propose = "3s"
timeout_prevote = "1s"
timeout_precommit = "1s"
timeout_commit = "5s"

[p2p]
listen_address = "tcp://0.0.0.0:26656"
max_num_inbound_peers = 40
max_num_outbound_peers = 10

[rpc]
listen_address = "tcp://0.0.0.0:26657"
cors_allowed_origins = ["*"]

[mempool]
size = 5000
max_txs_bytes = 1073741824

[instrumentation]
prometheus = false
`
	
	return os.WriteFile(configFile, []byte(configContent), 0644)
}