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
	require.Equal(t, uint64(1000), genesis.Params.HighActivityThreshold)
	require.Equal(t, uint64(100), genesis.Params.LowActivityThreshold)
	require.Equal(t, "1000000uvx", genesis.Params.MinBurnAmount)
	require.Equal(t, "1000000000uvx", genesis.Params.MaxBurnAmount)
	require.Equal(t, uint64(10), genesis.Params.BlockCreatorSelectionRounds)
	require.Equal(t, "0.95", genesis.Params.ActivityDecayRate)
	require.Equal(t, "0.1", genesis.Params.MoaPenaltyRate)
	require.Empty(t, genesis.Validators)
	require.Empty(t, genesis.BlockCreators)
	require.Empty(t, genesis.BurnProofs)
	require.Empty(t, genesis.ActivityScores)
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
	require.Equal(t, uint64(1000), params.HighActivityThreshold)
	require.Equal(t, uint64(100), params.LowActivityThreshold)
	require.Equal(t, "1000000uvx", params.MinBurnAmount)
	require.Equal(t, "1000000000uvx", params.MaxBurnAmount)
	require.Equal(t, uint64(10), params.BlockCreatorSelectionRounds)
	require.Equal(t, "0.95", params.ActivityDecayRate)
	require.Equal(t, "0.1", params.MoaPenaltyRate)
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
				BaseBlockTime:               "",
				HighActivityThreshold:       1000,
				LowActivityThreshold:        100,
				MinBurnAmount:               "1000000uvx",
				MaxBurnAmount:               "1000000000uvx",
				BlockCreatorSelectionRounds: uint64(10),
				ActivityDecayRate:           "0.95",
				MoaPenaltyRate:              "0.1",
			},
			wantErr: true,
		},
		{
			name: "empty min burn amount",
			params: &types.Params{
				BaseBlockTime:               "5s",
				HighActivityThreshold:       1000,
				LowActivityThreshold:        100,
				MinBurnAmount:               "",
				MaxBurnAmount:               "1000000000uvx",
				BlockCreatorSelectionRounds: uint64(10),
				ActivityDecayRate:           "0.95",
				MoaPenaltyRate:              "0.1",
			},
			wantErr: true,
		},
		{
			name: "empty max burn amount",
			params: &types.Params{
				BaseBlockTime:               "5s",
				HighActivityThreshold:       1000,
				LowActivityThreshold:        100,
				MinBurnAmount:               "1000000uvx",
				MaxBurnAmount:               "",
				BlockCreatorSelectionRounds: uint64(10),
				ActivityDecayRate:           "0.95",
				MoaPenaltyRate:              "0.1",
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
