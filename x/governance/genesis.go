package governance

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/volnix-protocol/volnix-protocol/x/governance/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/governance/types"
)

func DefaultGenesis() *types.GenesisState {
	return &types.GenesisState{
		Params:    types.DefaultParams(),
		Proposals: []interface{}{},
		Votes:     []interface{}{},
	}
}

func ValidateGenesis(genState *types.GenesisState) error {
	if genState == nil {
		return nil
	}
	return genState.Params.Validate()
}

func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState *types.GenesisState) {
	if genState == nil {
		genState = DefaultGenesis()
	}
	k.SetParams(ctx, genState.Params)
	k.SetNextProposalID(ctx, 1)
}

func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	params := k.GetParams(ctx)

	proposals, _ := k.GetAllProposals(ctx)
	exportedProposals := make([]interface{}, len(proposals))
	for i, p := range proposals {
		exportedProposals[i] = p
	}

	return &types.GenesisState{
		Params:    params,
		Proposals: exportedProposals,
		Votes:     []interface{}{},
	}
}
