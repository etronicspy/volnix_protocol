package governance

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/volnix-protocol/volnix-protocol/x/governance/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/governance/types"
)

// DefaultGenesis returns default genesis state for governance module
func DefaultGenesis() *types.GenesisState {
	return &types.GenesisState{
		Params:    types.DefaultParams(),
		Proposals: []interface{}{}, // Empty for now, will be populated after proto generation
		Votes:     []interface{}{}, // Empty for now, will be populated after proto generation
	}
}

// ValidateGenesis performs genesis state validation for the governance module
func ValidateGenesis(genState *types.GenesisState) error {
	if genState == nil {
		return nil
	}
	return genState.Params.Validate()
}

// InitGenesis initializes the governance module genesis state
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState *types.GenesisState) {
	if genState == nil {
		genState = DefaultGenesis()
	}
	
	// Set parameters
	k.SetParams(ctx, genState.Params)
	
	// Initialize proposal ID counter
	k.SetNextProposalID(ctx, 1)
	
	// TODO: Restore proposals and votes from genesis state after proto generation
	// For now, we start with an empty state
}

// ExportGenesis exports the governance module genesis state
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	params := k.GetParams(ctx)
	
	// TODO: Export proposals and votes after proto generation
	return &types.GenesisState{
		Params:    params,
		Proposals: []interface{}{}, // Empty for now, will be populated after proto generation
		Votes:     []interface{}{}, // Empty for now, will be populated after proto generation
	}
}

