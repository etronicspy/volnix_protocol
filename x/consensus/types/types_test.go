package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

func TestDefaultGenesis(t *testing.T) {
	genesis := types.DefaultGenesis()

	require.NotNil(t, genesis)
	require.NotNil(t, genesis.Params)
	require.Equal(t, "5s", genesis.Params.BaseBlockTime)
	require.Equal(t, "50000000uwrt", genesis.Params.BaseBlockReward)
	require.Equal(t, "0.5", genesis.Params.BurnCapLambda)
	require.Equal(t, "community_pool", genesis.Params.FeePolicyBZero)
	require.Equal(t, uint64(210000), genesis.Params.HalvingInterval)
	require.Equal(t, "1.0", genesis.Params.MoaPenaltyThresholdHigh)
	require.Equal(t, "0.9", genesis.Params.MoaPenaltyThresholdWarning)
	require.Equal(t, "0.7", genesis.Params.MoaPenaltyThresholdMedium)
	require.Equal(t, "0.5", genesis.Params.MoaPenaltyThresholdLow)
	require.Equal(t, uint64(40000000), genesis.Params.MaxBlockGas)
	require.Equal(t, uint64(22020096), genesis.Params.MaxBlockBytes)
	require.Empty(t, genesis.Validators)
}

func TestValidateGenesis(t *testing.T) {
	tests := []struct {
		name    string
		genesis *types.GenesisState
		wantErr bool
	}{
		{
			name:    "valid genesis",
			genesis: types.DefaultGenesis(),
			wantErr: false,
		},
		{
			name: "nil params",
			genesis: &types.GenesisState{
				Params: nil,
			},
			wantErr: true,
		},
		{
			name: "valid custom genesis",
			genesis: &types.GenesisState{
				Params: types.DefaultParams(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateGenesis(tt.genesis)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()

	require.NotNil(t, params)
	require.Equal(t, "5s", params.BaseBlockTime)
	require.Equal(t, "50000000uwrt", params.BaseBlockReward)
	require.Equal(t, "0.5", params.BurnCapLambda)
	require.Equal(t, "community_pool", params.FeePolicyBZero)
	require.Equal(t, uint64(210000), params.HalvingInterval)
	require.Equal(t, "1.0", params.MoaPenaltyThresholdHigh)
	require.Equal(t, "0.9", params.MoaPenaltyThresholdWarning)
	require.Equal(t, "0.7", params.MoaPenaltyThresholdMedium)
	require.Equal(t, "0.5", params.MoaPenaltyThresholdLow)
	require.Equal(t, uint64(40000000), params.MaxBlockGas)
	require.Equal(t, uint64(22020096), params.MaxBlockBytes)
}

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name    string
		params  *types.Params
		wantErr bool
	}{
		{
			name:    "valid params",
			params:  types.DefaultParams(),
			wantErr: false,
		},
		{
			name:    "nil params",
			params:  nil,
			wantErr: true,
		},
		{
			name: "empty base block time",
			params: &types.Params{
				BaseBlockTime:              "",
				BaseBlockReward:            "50000000uwrt",
				BurnCapLambda:              "0.333333333333333333",
				FeePolicyBZero:             "community_pool",
				HalvingInterval:            210000,
				MoaPenaltyThresholdHigh:    "1.0",
				MoaPenaltyThresholdWarning: "0.9",
				MoaPenaltyThresholdMedium:  "0.7",
				MoaPenaltyThresholdLow:     "0.5",
				MaxBlockGas:                40000000,
				MaxBlockBytes:              22020096,
			},
			wantErr: true,
		},
		{
			name: "empty base block reward",
			params: &types.Params{
				BaseBlockTime:              "5s",
				BaseBlockReward:            "",
				BurnCapLambda:              "0.333333333333333333",
				FeePolicyBZero:             "community_pool",
				HalvingInterval:            210000,
				MoaPenaltyThresholdHigh:    "1.0",
				MoaPenaltyThresholdWarning: "0.9",
				MoaPenaltyThresholdMedium:  "0.7",
				MoaPenaltyThresholdLow:     "0.5",
				MaxBlockGas:                40000000,
				MaxBlockBytes:              22020096,
			},
			wantErr: true,
		},
		{
			name: "empty burn cap lambda",
			params: &types.Params{
				BaseBlockTime:              "5s",
				BaseBlockReward:            "50000000uwrt",
				BurnCapLambda:              "",
				FeePolicyBZero:             "community_pool",
				HalvingInterval:            210000,
				MoaPenaltyThresholdHigh:    "1.0",
				MoaPenaltyThresholdWarning: "0.9",
				MoaPenaltyThresholdMedium:  "0.7",
				MoaPenaltyThresholdLow:     "0.5",
				MaxBlockGas:                40000000,
				MaxBlockBytes:              22020096,
			},
			wantErr: true,
		},
		{
			name: "zero halving interval",
			params: &types.Params{
				BaseBlockTime:              "5s",
				BaseBlockReward:            "50000000uwrt",
				BurnCapLambda:              "0.333333333333333333",
				FeePolicyBZero:             "community_pool",
				HalvingInterval:            0,
				MoaPenaltyThresholdHigh:    "1.0",
				MoaPenaltyThresholdWarning: "0.9",
				MoaPenaltyThresholdMedium:  "0.7",
				MoaPenaltyThresholdLow:     "0.5",
				MaxBlockGas:                40000000,
				MaxBlockBytes:              22020096,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateParams(tt.params)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestKeyPrefix(t *testing.T) {
	key := types.KeyPrefix("test")
	require.NotNil(t, key)
	require.Equal(t, []byte("test"), key)
}

func TestGetValidatorKey(t *testing.T) {
	validator := "cosmos1validator"
	key := types.GetValidatorKey(validator)
	require.NotNil(t, key)
	require.Contains(t, string(key), validator)
}

func TestGetBlockCreatorKey(t *testing.T) {
	height := uint64(100)
	key := types.GetBlockCreatorKey(height)
	require.NotNil(t, key)
	require.Contains(t, string(key), "100")
}

func TestGetValidatorWeightKey(t *testing.T) {
	validator := "cosmos1validator"
	key := types.GetValidatorWeightKey(validator)
	require.NotNil(t, key)
	require.Contains(t, string(key), validator)
}

func TestKeyHalvingInfo(t *testing.T) {
	key := types.KeyHalvingInfo()
	require.NotNil(t, key)
	require.Equal(t, types.HalvingInfoKey, key)
}

func TestGetBlockTimeKey(t *testing.T) {
	height := uint64(100)
	key := types.GetBlockTimeKey(height)
	require.NotNil(t, key)
	require.NotEmpty(t, key)
	
	// Test different heights produce different keys
	key2 := types.GetBlockTimeKey(200)
	require.NotEqual(t, key, key2)
}

func TestKeyConsensusState(t *testing.T) {
	key := types.KeyConsensusState()
	require.NotNil(t, key)
	require.Equal(t, types.ConsensusStateKey, key)
}

func TestGetBlindAuctionKey(t *testing.T) {
	// Test basic functionality
	height := uint64(100)
	key := types.GetBlindAuctionKey(height)
	require.NotNil(t, key)
	require.NotEmpty(t, key)
	
	// Test different heights
	key2 := types.GetBlindAuctionKey(200)
	require.NotNil(t, key2)
	require.NotEmpty(t, key2)
	
	// Test same height produces same key
	key3 := types.GetBlindAuctionKey(100)
	require.Equal(t, string(key), string(key3), "keys for same height should be equal")
}

func TestGetBidHistoryKey(t *testing.T) {
	validator := "cosmos1validator"
	key := types.GetBidHistoryKey(validator)
	require.NotNil(t, key)
	require.NotEmpty(t, key)
	
	// Test different validators produce different keys
	key2 := types.GetBidHistoryKey("cosmos1validator2")
	require.NotEqual(t, key, key2)
}
