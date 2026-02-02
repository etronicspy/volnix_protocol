package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
)

func TestTestnetConfig(t *testing.T) {
	cfg := TestnetConfig()
	require.NotNil(t, cfg)
	require.Equal(t, "volnix-testnet-1", cfg.Network.ChainID)
	require.Equal(t, "debug", cfg.Logging.Level)
}

func TestMainnetConfig(t *testing.T) {
	cfg := MainnetConfig()
	require.NotNil(t, cfg)
	require.Equal(t, "volnix-1", cfg.Network.ChainID)
	require.Equal(t, "9090", cfg.Monitoring.Port)
	require.Equal(t, "warn", cfg.Logging.Level)
}

func TestValidateConfig_Valid(t *testing.T) {
	cfg := DefaultConfig()
	err := ValidateConfig(cfg)
	require.NoError(t, err)
}

func TestValidateConfig_EmptyChainID(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Network.ChainID = ""
	err := ValidateConfig(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "chain_id")
}

func TestValidateConfig_InvalidAlgorithm(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Consensus.Algorithm = "invalid"
	err := ValidateConfig(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "consensus")
}

func TestValidateConfig_InvalidLogLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logging.Level = "invalid"
	err := ValidateConfig(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "log level")
}

func TestLoadConfig_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "test-volnix", cfg.Network.ChainID)
	// SaveConfig creates the file, so it should exist now
	_, err = os.Stat(path)
	require.NoError(t, err)
}

func TestLoadConfig_FileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	orig := DefaultConfig()
	orig.Network.ChainID = "loaded-chain"
	require.NoError(t, SaveConfig(orig, path))
	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "loaded-chain", cfg.Network.ChainID)
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := DefaultConfig()
	err := SaveConfig(cfg, path)
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "test-volnix")
}

func TestNewConfigManager(t *testing.T) {
	cm := NewConfigManager("/tmp/config.json", nil)
	require.NotNil(t, cm)
	require.Equal(t, "/tmp/config.json", cm.configPath)
}

func TestConfigManager_GetConfig_BeforeLoad(t *testing.T) {
	cm := NewConfigManager("/tmp/config.json", nil)
	require.Nil(t, cm.GetConfig())
}

func TestConfigManager_Load_GetConfig_Save(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, SaveConfig(DefaultConfig(), path))
	cm := NewConfigManager(path, log.NewNopLogger())
	require.NoError(t, cm.Load())
	require.NotNil(t, cm.GetConfig())
	require.Equal(t, "test-volnix", cm.GetConfig().Network.ChainID)
	require.NoError(t, cm.Save())
}

func TestConfigManager_Save_NoConfig(t *testing.T) {
	cm := NewConfigManager(t.TempDir()+"/x.json", log.NewNopLogger())
	err := cm.Save()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no configuration")
}

func TestConfigManager_UpdateConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, SaveConfig(DefaultConfig(), path))
	cm := NewConfigManager(path, log.NewNopLogger())
	require.NoError(t, cm.Load())
	cfg := DefaultConfig()
	cfg.Network.ChainID = "updated-chain"
	require.NoError(t, cm.UpdateConfig(cfg))
	require.Equal(t, "updated-chain", cm.GetConfig().Network.ChainID)
}

func TestConfigManager_UpdateConfig_Invalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, SaveConfig(DefaultConfig(), path))
	cm := NewConfigManager(path, log.NewNopLogger())
	require.NoError(t, cm.Load())
	cfg := DefaultConfig()
	cfg.Network.ChainID = ""
	err := cm.UpdateConfig(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid configuration")
}

func TestConfigManager_GetNetworkConfig(t *testing.T) {
	cm := NewConfigManager("/tmp/x.json", nil)
	nc := cm.GetNetworkConfig()
	require.Equal(t, DefaultConfig().Network.ChainID, nc.ChainID)

	dir := t.TempDir()
	path := filepath.Join(dir, "c.json")
	require.NoError(t, SaveConfig(DefaultConfig(), path))
	cm2 := NewConfigManager(path, log.NewNopLogger())
	require.NoError(t, cm2.Load())
	nc2 := cm2.GetNetworkConfig()
	require.Equal(t, "test-volnix", nc2.ChainID)
}

func TestConfigManager_GetConsensusConfig(t *testing.T) {
	cm := NewConfigManager("/tmp/x.json", nil)
	cc := cm.GetConsensusConfig()
	require.Equal(t, DefaultConfig().Consensus.Algorithm, cc.Algorithm)
}

func TestConfigManager_GetEconomicConfig(t *testing.T) {
	cm := NewConfigManager("/tmp/x.json", nil)
	ec := cm.GetEconomicConfig()
	require.NotNil(t, ec)
}
