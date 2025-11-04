package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/log"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/types"
)

// VolnixServer wraps Volnix app with CometBFT node functionality
type VolnixServer struct {
	app       *MinimalVolnixApp
	node      *node.Node
	config    *cmtcfg.Config
	homeDir   string
	logger    log.Logger
	cmtLogger cmtlog.Logger
}

// REMOVED: Old NewCometBFTServer function that was causing protobuf registration issues
// Use NewMinimalCometBFTServer from minimal_server.go instead

// Start starts the Volnix server with CometBFT node
func (s *VolnixServer) Start(ctx context.Context) error {
	s.logger.Info("🚀 Starting Volnix Protocol with CometBFT...")

	// Initialize files and configuration
	if err := s.initializeFiles(); err != nil {
		return fmt.Errorf("failed to initialize files: %w", err)
	}

	// Create CometBFT node
	if err := s.createCometBFTNode(); err != nil {
		return fmt.Errorf("failed to create CometBFT node: %w", err)
	}

	s.logger.Info("✅ CometBFT node created successfully")
	s.logger.Info("🌐 Network configuration:")
	s.logger.Info("   🔗 Chain ID: test-volnix")
	s.logger.Info("   📁 Home: " + s.homeDir)
	s.logger.Info("   💾 Database: GoLevelDB")
	s.logger.Info("   🏗️  Framework: Cosmos SDK + CometBFT")

	s.logger.Info("📦 Active modules:")
	s.logger.Info("   ✅ ident - Identity & ZKP verification")
	s.logger.Info("   ✅ lizenz - LZN license management")
	s.logger.Info("   ✅ anteil - ANT internal market")
	s.logger.Info("   ✅ consensus - PoVB consensus")

	s.logger.Info("🌐 Network endpoints:")
	s.logger.Info("   🔗 RPC: " + s.config.RPC.ListenAddress)
	s.logger.Info("   📡 P2P: " + s.config.P2P.ListenAddress)

	// Start CometBFT node
	s.logger.Info("⚡ Starting CometBFT consensus...")
	if err := s.node.Start(); err != nil {
		return fmt.Errorf("failed to start CometBFT node: %w", err)
	}

	s.logger.Info("🎯 Volnix Protocol node is running!")
	s.logger.Info("✨ Ready for transactions, queries, and consensus!")

	// Wait for context cancellation
	<-ctx.Done()

	return s.Stop()
}

// Stop stops the Volnix server and CometBFT node
func (s *VolnixServer) Stop() error {
	s.logger.Info("🛑 Stopping Volnix Protocol node...")

	if s.node != nil && s.node.IsRunning() {
		if err := s.node.Stop(); err != nil {
			s.logger.Error("Failed to stop CometBFT node", "error", err)
			return err
		}
		s.logger.Info("✅ CometBFT node stopped")
	}

	s.logger.Info("✅ Volnix Protocol node stopped successfully")
	return nil
}

// GetApp returns the Volnix app
func (s *VolnixServer) GetApp() *MinimalVolnixApp {
	return s.app
}

// GetNode returns the CometBFT node
func (s *VolnixServer) GetNode() *node.Node {
	return s.node
}

// createCometBFTNode creates and configures the CometBFT node
func (s *VolnixServer) createCometBFTNode() error {
	// Load or generate node key
	nodeKeyFile := filepath.Join(s.config.RootDir, "config", "node_key.json")
	nodeKey, err := p2p.LoadOrGenNodeKey(nodeKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load or generate node key: %w", err)
	}

	// Load or generate private validator
	privValKeyFile := filepath.Join(s.config.RootDir, "config", "priv_validator_key.json")
	privValStateFile := filepath.Join(s.config.RootDir, "data", "priv_validator_state.json")
	privValidator := privval.LoadOrGenFilePV(privValKeyFile, privValStateFile)

	// Create genesis provider
	genesisFile := filepath.Join(s.config.RootDir, "config", "genesis.json")
	genesisProvider := func() (*types.GenesisDoc, error) {
		return types.GenesisDocFromFile(genesisFile)
	}

	// Create database provider
	dbProvider := cmtcfg.DefaultDBProvider

	// Create metrics provider (disabled for now)
	metricsProvider := node.DefaultMetricsProvider(s.config.Instrumentation)

	// Create ABCI wrapper and client creator
	abciWrapper := NewABCIWrapper(s.app)
	clientCreator := proxy.NewLocalClientCreator(abciWrapper)

	// Create CometBFT node
	s.node, err = node.NewNode(
		s.config,
		privValidator,
		nodeKey,
		clientCreator,
		genesisProvider,
		dbProvider,
		metricsProvider,
		s.cmtLogger,
	)
	if err != nil {
		return fmt.Errorf("failed to create CometBFT node: %w", err)
	}

	return nil
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

// createGenesisFile creates a default genesis file compatible with CometBFT
func (s *VolnixServer) createGenesisFile(genesisFile string) error {
	// Create a proper genesis document
	genDoc := &types.GenesisDoc{
		GenesisTime:     time.Now(),
		ChainID:         "test-volnix",
		InitialHeight:   1,
		ConsensusParams: types.DefaultConsensusParams(),
		AppHash:         []byte{},
		AppState: []byte(`{
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
}`),
	}

	// Add default validator if none exists
	privValKeyFile := filepath.Join(s.config.RootDir, "config", "priv_validator_key.json")
	if _, err := os.Stat(privValKeyFile); os.IsNotExist(err) {
		// Generate private validator
		privVal := privval.GenFilePV(privValKeyFile, filepath.Join(s.config.RootDir, "data", "priv_validator_state.json"))
		pubKey, err := privVal.GetPubKey()
		if err != nil {
			return fmt.Errorf("failed to get validator public key: %w", err)
		}

		// Add validator to genesis
		validator := types.GenesisValidator{
			Address: pubKey.Address(),
			PubKey:  pubKey,
			Power:   10,
			Name:    "volnix-validator",
		}
		genDoc.Validators = []types.GenesisValidator{validator}
	}

	// Save genesis file
	return genDoc.SaveAs(genesisFile)
}

// createConfigFile creates a default CometBFT config file
func (s *VolnixServer) createConfigFile(configFile string) error {
	// Write the current config to file
	cmtcfg.WriteConfigFile(configFile, s.config)
	return nil
}
