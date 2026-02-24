package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/log"
	cosmosdb "github.com/cosmos/cosmos-db"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/types"
)

// MinimalVolnixServer runs CometBFT with the full Volnix application
// (ident, lizenz, consensus modules wired with real keepers).
type MinimalVolnixServer struct {
	app       *VolnixApp
	node      *node.Node
	config    *cmtcfg.Config
	homeDir   string
	logger    log.Logger
	cmtLogger cmtlog.Logger
}

// NewMinimalCometBFTServer creates a minimal server for CometBFT testing
func NewMinimalCometBFTServer(homeDir string, logger log.Logger) (*MinimalVolnixServer, error) {
	// Create database
	dbPath := filepath.Join(homeDir, "data")
	db, err := cosmosdb.NewGoLevelDB("volnix", dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Real encoding config with protobuf tx codec and all module types
	encodingConfig := MakeEncodingConfig()

	// Full Volnix app with all modules wired
	app := NewVolnixApp(logger, db, nil, encodingConfig, nil)

	// Enable disk persistence for snapshots (survives restarts)
	if app.snapshotManager != nil {
		app.snapshotManager.SetSnapshotDir(filepath.Join(homeDir, "data", "snapshots"))
	}

	// Create CometBFT config
	config := cmtcfg.DefaultConfig()
	config.SetRoot(homeDir)
	config.Moniker = "volnix-node"

	// Configure consensus
	config.Consensus.TimeoutPropose = 3 * time.Second
	config.Consensus.TimeoutPrevote = 1 * time.Second
	config.Consensus.TimeoutPrecommit = 1 * time.Second
	config.Consensus.TimeoutCommit = 5 * time.Second
	config.Consensus.CreateEmptyBlocks = true
	config.Consensus.CreateEmptyBlocksInterval = 0 * time.Second

	// Configure P2P
	config.P2P.ListenAddress = "tcp://0.0.0.0:26656"
	config.P2P.MaxNumInboundPeers = 40
	config.P2P.MaxNumOutboundPeers = 10

	// Configure RPC
	config.RPC.ListenAddress = "tcp://0.0.0.0:26657"
	config.RPC.CORSAllowedOrigins = []string{"*"}

	// Configure mempool
	config.Mempool.Size = 5000
	config.Mempool.MaxTxsBytes = 1073741824

	// Create CometBFT logger adapter
	cmtLogger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))

	server := &MinimalVolnixServer{
		app:       app,
		config:    config,
		homeDir:   homeDir,
		logger:    logger,
		cmtLogger: cmtLogger,
	}

	return server, nil
}

// Start starts the minimal server with CometBFT node
func (s *MinimalVolnixServer) Start(ctx context.Context) error {
	s.logger.Info("🚀 Starting Minimal Volnix Protocol with CometBFT...")

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
	s.logger.Info("   🔗 Chain ID: volnix-1")
	s.logger.Info("   📁 Home: " + s.homeDir)
	s.logger.Info("   💾 Database: GoLevelDB")
	s.logger.Info("   🏗️  Framework: Minimal Cosmos SDK + CometBFT")

	s.logger.Info("🌐 Network endpoints:")
	s.logger.Info("   🔗 RPC: " + s.config.RPC.ListenAddress)
	s.logger.Info("   📡 P2P: " + s.config.P2P.ListenAddress)

	// Start CometBFT node
	s.logger.Info("⚡ Starting CometBFT consensus...")
	if err := s.node.Start(); err != nil {
		return fmt.Errorf("failed to start CometBFT node: %w", err)
	}

	s.logger.Info("🎯 Minimal Volnix Protocol node is running!")
	s.logger.Info("✨ Ready for consensus and P2P networking!")

	// Wait for context cancellation
	<-ctx.Done()

	return s.Stop()
}

// Stop stops the server and CometBFT node
func (s *MinimalVolnixServer) Stop() error {
	s.logger.Info("🛑 Stopping Minimal Volnix Protocol node...")

	if s.node != nil && s.node.IsRunning() {
		if err := s.node.Stop(); err != nil {
			s.logger.Error("Failed to stop CometBFT node", "error", err)
			return err
		}
		s.logger.Info("✅ CometBFT node stopped")
	}

	s.logger.Info("✅ Minimal Volnix Protocol node stopped successfully")
	return nil
}

// GetApp returns the Volnix app
func (s *MinimalVolnixServer) GetApp() *VolnixApp {
	return s.app
}

// GetNode returns the CometBFT node
func (s *MinimalVolnixServer) GetNode() *node.Node {
	return s.node
}

// Config returns the CometBFT config (for monitoring and CLI)
func (s *MinimalVolnixServer) Config() *cmtcfg.Config {
	return s.config
}

// Logger returns the server logger
func (s *MinimalVolnixServer) Logger() log.Logger {
	return s.logger
}

// InitializeFiles creates config/genesis and resets priv_validator_state (exported for tests)
func (s *MinimalVolnixServer) InitializeFiles() error {
	return s.initializeFiles()
}

// CreateGenesisFile creates the genesis file at the given path (exported for tests)
func (s *MinimalVolnixServer) CreateGenesisFile(genesisFile string) error {
	configDir := filepath.Join(s.homeDir, "config")
	dataDir := filepath.Join(s.homeDir, "data")
	privValKeyFile := filepath.Join(configDir, "priv_validator_key.json")
	privValStateFile := filepath.Join(dataDir, "priv_validator_state.json")
	var pv *privval.FilePV
	if _, err := os.Stat(privValKeyFile); os.IsNotExist(err) {
		pv = privval.GenFilePV(privValKeyFile, privValStateFile)
	} else {
		pv = privval.LoadFilePV(privValKeyFile, privValStateFile)
	}
	return s.createGenesisFile(genesisFile, pv)
}

// createCometBFTNode creates and configures the CometBFT node
func (s *MinimalVolnixServer) createCometBFTNode() error {
	// Load or generate node key
	nodeKeyFile := filepath.Join(s.config.RootDir, "config", "node_key.json")
	nodeKey, err := p2p.LoadOrGenNodeKey(nodeKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load or generate node key: %w", err)
	}

	// Load private validator (key was generated during initializeFiles)
	privValKeyFile := filepath.Join(s.config.RootDir, "config", "priv_validator_key.json")
	privValStateFile := filepath.Join(s.config.RootDir, "data", "priv_validator_state.json")
	privValidator := privval.LoadFilePV(privValKeyFile, privValStateFile)

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
func (s *MinimalVolnixServer) initializeFiles() error {
	configDir := filepath.Join(s.homeDir, "config")
	dataDir := filepath.Join(s.homeDir, "data")

	// Ensure directories exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// Re-apply CreateEmptyBlocks so config file and memory stay in sync
	s.config.Consensus.CreateEmptyBlocks = true
	s.config.Consensus.CreateEmptyBlocksInterval = 0 * time.Second

	// Generate or load validator key (single source of truth)
	privValKeyFile := filepath.Join(configDir, "priv_validator_key.json")
	privValStateFile := filepath.Join(dataDir, "priv_validator_state.json")
	var pv *privval.FilePV
	if _, err := os.Stat(privValKeyFile); os.IsNotExist(err) {
		pv = privval.GenFilePV(privValKeyFile, privValStateFile)
	} else {
		pv = privval.LoadFilePV(privValKeyFile, privValStateFile)
	}

	// CosmJS compatibility: always reset priv_validator_state on startup
	pv.Reset()
	pv.Save()

	// Create genesis file if it doesn't exist
	genesisFile := filepath.Join(configDir, "genesis.json")
	if _, err := os.Stat(genesisFile); os.IsNotExist(err) {
		if err := s.createGenesisFile(genesisFile, pv); err != nil {
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

// createGenesisFile creates a genesis file with proper module state.
// The app_state is left empty so VolnixApp.InitChainer builds defaults
// and injects the CometBFT genesis validators into the consensus module.
// Bank balances for the demo wallet are seeded via app_state.bank.
func (s *MinimalVolnixServer) createGenesisFile(genesisFile string, pv *privval.FilePV) error {
	pubKey, err := pv.GetPubKey()
	if err != nil {
		return fmt.Errorf("failed to get validator public key: %w", err)
	}

	// Provide bank genesis so the demo wallet is funded from block 1.
	// All other modules will use their DefaultGenesis inside InitChainer.
	appState := `{
  "bank": {
    "params": {"send_enabled": [], "default_send_enabled": true},
    "balances": [
      {
        "address": "` + DemoWalletAddress + `",
        "coins": [
          {"denom": "uant", "amount": "250000000"},
          {"denom": "ulzn", "amount": "500000000"},
          {"denom": "uwrt", "amount": "1000000000"}
        ]
      }
    ],
    "supply": [
      {"denom": "uant", "amount": "250000000"},
      {"denom": "ulzn", "amount": "500000000"},
      {"denom": "uwrt", "amount": "1000000000"}
    ],
    "denom_metadata": [],
    "send_enabled": []
  }
}`

	genDoc := &types.GenesisDoc{
		GenesisTime:     time.Now(),
		ChainID:         "volnix-1",
		InitialHeight:   1,
		ConsensusParams: types.DefaultConsensusParams(),
		AppHash:         []byte{},
		AppState:        []byte(appState),
		Validators: []types.GenesisValidator{
			{
				Address: pubKey.Address(),
				PubKey:  pubKey,
				Power:   10,
				Name:    "volnix-validator",
			},
		},
	}

	return genDoc.SaveAs(genesisFile)
}

// createConfigFile creates a CometBFT config file
func (s *MinimalVolnixServer) createConfigFile(configFile string) error {
	// Write the current config to file
	cmtcfg.WriteConfigFile(configFile, s.config)
	return nil
}
