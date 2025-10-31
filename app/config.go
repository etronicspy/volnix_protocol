package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/log"
)

// Config represents the application configuration
type Config struct {
	// Network configuration
	Network NetworkConfig `json:"network"`
	
	// Consensus configuration
	Consensus ConsensusConfig `json:"consensus"`
	
	// Economic configuration
	Economic EconomicConfig `json:"economic"`
	
	// Identity configuration
	Identity IdentityConfig `json:"identity"`
	
	// Monitoring configuration
	Monitoring MonitoringConfig `json:"monitoring"`
	
	// Logging configuration
	Logging LoggingConfig `json:"logging"`
}

// NetworkConfig contains network-related configuration
type NetworkConfig struct {
	ChainID         string        `json:"chain_id"`
	ListenAddress   string        `json:"listen_address"`
	ExternalAddress string        `json:"external_address"`
	Seeds           []string      `json:"seeds"`
	PersistentPeers []string      `json:"persistent_peers"`
	MaxPeers        int           `json:"max_peers"`
	HandshakeTimeout time.Duration `json:"handshake_timeout"`
}

// ConsensusConfig contains PoVB consensus configuration
type ConsensusConfig struct {
	Algorithm           string        `json:"algorithm"`
	BlockTime           time.Duration `json:"block_time"`
	HalvingInterval     uint64        `json:"halving_interval"`
	MinValidatorWeight  uint64        `json:"min_validator_weight"`
	MaxValidatorWeight  uint64        `json:"max_validator_weight"`
	BurnProofRequired   bool          `json:"burn_proof_required"`
	WeightDecayRate     float64       `json:"weight_decay_rate"`
}

// EconomicConfig contains economic system configuration
type EconomicConfig struct {
	BaseCurrency        string  `json:"base_currency"`
	MinOrderAmount      string  `json:"min_order_amount"`
	MaxOrderAmount      string  `json:"max_order_amount"`
	TradingFee          float64 `json:"trading_fee"`
	AuctionDuration     time.Duration `json:"auction_duration"`
	MatchingInterval    time.Duration `json:"matching_interval"`
	PriceDecimalPlaces  int     `json:"price_decimal_places"`
	VolumeDecimalPlaces int     `json:"volume_decimal_places"`
}

// IdentityConfig contains identity system configuration
type IdentityConfig struct {
	VerificationRequired    bool          `json:"verification_required"`
	VerificationTimeout     time.Duration `json:"verification_timeout"`
	MaxPendingVerifications int           `json:"max_pending_verifications"`
	RoleMigrationEnabled    bool          `json:"role_migration_enabled"`
	AutoVerificationEnabled bool          `json:"auto_verification_enabled"`
}

// MonitoringConfig contains monitoring system configuration
type MonitoringConfig struct {
	Enabled         bool   `json:"enabled"`
	Port            string `json:"port"`
	MetricsInterval time.Duration `json:"metrics_interval"`
	HealthCheckPath string `json:"health_check_path"`
	PrometheusEnabled bool `json:"prometheus_enabled"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	OutputPath string `json:"output_path"`
	MaxSize    int    `json:"max_size_mb"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age_days"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Network: NetworkConfig{
			ChainID:         "test-volnix",
			ListenAddress:   "tcp://0.0.0.0:26656",
			ExternalAddress: "",
			Seeds:           []string{},
			PersistentPeers: []string{},
			MaxPeers:        50,
			HandshakeTimeout: 20 * time.Second,
		},
		Consensus: ConsensusConfig{
			Algorithm:          "PoVB",
			BlockTime:          5 * time.Second,
			HalvingInterval:    210000, // ~4 years with 5s blocks
			MinValidatorWeight: 1000,
			MaxValidatorWeight: 1000000,
			BurnProofRequired:  true,
			WeightDecayRate:    0.001,
		},
		Economic: EconomicConfig{
			BaseCurrency:        "ANT",
			MinOrderAmount:      "0.001",
			MaxOrderAmount:      "1000000.0",
			TradingFee:          0.001, // 0.1%
			AuctionDuration:     24 * time.Hour,
			MatchingInterval:    1 * time.Second,
			PriceDecimalPlaces:  8,
			VolumeDecimalPlaces: 8,
		},
		Identity: IdentityConfig{
			VerificationRequired:    true,
			VerificationTimeout:     72 * time.Hour,
			MaxPendingVerifications: 1000,
			RoleMigrationEnabled:    true,
			AutoVerificationEnabled: false,
		},
		Monitoring: MonitoringConfig{
			Enabled:           true,
			Port:              "8080",
			MetricsInterval:   30 * time.Second,
			HealthCheckPath:   "/health",
			PrometheusEnabled: true,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputPath: "volnix.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     30,
		},
	}
}

// TestnetConfig returns configuration for testnet
func TestnetConfig() *Config {
	config := DefaultConfig()
	config.Network.ChainID = "volnix-testnet-1"
	config.Consensus.BlockTime = 3 * time.Second
	config.Consensus.HalvingInterval = 50000 // Faster halving for testing
	config.Economic.TradingFee = 0.0005 // Lower fees for testing
	config.Logging.Level = "debug"
	return config
}

// MainnetConfig returns configuration for mainnet
func MainnetConfig() *Config {
	config := DefaultConfig()
	config.Network.ChainID = "volnix-1"
	config.Consensus.BlockTime = 6 * time.Second
	config.Economic.TradingFee = 0.002 // Higher fees for mainnet
	config.Monitoring.Port = "9090"
	config.Logging.Level = "warn"
	return config
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config if file doesn't exist
		config := DefaultConfig()
		if err := SaveConfig(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config) error {
	// Validate network config
	if config.Network.ChainID == "" {
		return fmt.Errorf("chain_id cannot be empty")
	}
	if config.Network.ListenAddress == "" {
		return fmt.Errorf("listen_address cannot be empty")
	}
	if config.Network.MaxPeers <= 0 {
		return fmt.Errorf("max_peers must be positive")
	}

	// Validate consensus config
	if config.Consensus.Algorithm != "PoVB" {
		return fmt.Errorf("unsupported consensus algorithm: %s", config.Consensus.Algorithm)
	}
	if config.Consensus.BlockTime <= 0 {
		return fmt.Errorf("block_time must be positive")
	}
	if config.Consensus.HalvingInterval == 0 {
		return fmt.Errorf("halving_interval must be positive")
	}
	if config.Consensus.MinValidatorWeight >= config.Consensus.MaxValidatorWeight {
		return fmt.Errorf("min_validator_weight must be less than max_validator_weight")
	}

	// Validate economic config
	if config.Economic.BaseCurrency == "" {
		return fmt.Errorf("base_currency cannot be empty")
	}
	if config.Economic.TradingFee < 0 || config.Economic.TradingFee > 1 {
		return fmt.Errorf("trading_fee must be between 0 and 1")
	}
	if config.Economic.AuctionDuration <= 0 {
		return fmt.Errorf("auction_duration must be positive")
	}

	// Validate identity config
	if config.Identity.VerificationTimeout <= 0 {
		return fmt.Errorf("verification_timeout must be positive")
	}
	if config.Identity.MaxPendingVerifications <= 0 {
		return fmt.Errorf("max_pending_verifications must be positive")
	}

	// Validate monitoring config
	if config.Monitoring.Enabled && config.Monitoring.Port == "" {
		return fmt.Errorf("monitoring port cannot be empty when monitoring is enabled")
	}

	// Validate logging config
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	return nil
}

// ConfigManager manages application configuration
type ConfigManager struct {
	config     *Config
	configPath string
	logger     log.Logger
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string, logger log.Logger) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
		logger:     logger,
	}
}

// Load loads the configuration
func (cm *ConfigManager) Load() error {
	config, err := LoadConfig(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cm.config = config
	cm.logger.Info("Configuration loaded", "path", cm.configPath)
	return nil
}

// Save saves the current configuration
func (cm *ConfigManager) Save() error {
	if cm.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	if err := SaveConfig(cm.config, cm.configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	cm.logger.Info("Configuration saved", "path", cm.configPath)
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// UpdateConfig updates the configuration
func (cm *ConfigManager) UpdateConfig(newConfig *Config) error {
	if err := ValidateConfig(newConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cm.config = newConfig
	return cm.Save()
}

// GetNetworkConfig returns network configuration
func (cm *ConfigManager) GetNetworkConfig() NetworkConfig {
	if cm.config != nil {
		return cm.config.Network
	}
	return DefaultConfig().Network
}

// GetConsensusConfig returns consensus configuration
func (cm *ConfigManager) GetConsensusConfig() ConsensusConfig {
	if cm.config != nil {
		return cm.config.Consensus
	}
	return DefaultConfig().Consensus
}

// GetEconomicConfig returns economic configuration
func (cm *ConfigManager) GetEconomicConfig() EconomicConfig {
	if cm.config != nil {
		return cm.config.Economic
	}
	return DefaultConfig().Economic
}