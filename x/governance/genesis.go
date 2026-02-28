package governance

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/codec"
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

func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState *types.GenesisState, cdc codec.JSONCodec) {
	if genState == nil {
		genState = DefaultGenesis()
	}
	k.SetParams(ctx, genState.Params)

	var maxProposalID uint64
	for _, item := range genState.Proposals {
		bz, err := json.Marshal(item)
		if err != nil {
			continue
		}
		var p keeper.Proposal
		if err := cdc.UnmarshalJSON(bz, &p); err != nil {
			continue
		}
		if err := k.SetProposal(ctx, &p); err != nil {
			continue
		}
		if p.ProposalId > maxProposalID {
			maxProposalID = p.ProposalId
		}
	}

	for _, item := range genState.Votes {
		bz, err := json.Marshal(item)
		if err != nil {
			continue
		}
		var v keeper.Vote
		if err := cdc.UnmarshalJSON(bz, &v); err != nil {
			continue
		}
		_ = k.SetVote(ctx, &v)
	}

	nextID := uint64(1)
	if maxProposalID > 0 {
		nextID = maxProposalID + 1
	}
	k.SetNextProposalID(ctx, nextID)
}

func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	params := k.GetParams(ctx)

	proposals, _ := k.GetAllProposals(ctx)
	exportedProposals := make([]interface{}, len(proposals))
	for i, p := range proposals {
		exportedProposals[i] = p
	}

	votes, _ := k.GetAllVotes(ctx)
	exportedVotes := make([]interface{}, len(votes))
	for i, v := range votes {
		exportedVotes[i] = v
	}

	return &types.GenesisState{
		Params:    params,
		Proposals: exportedProposals,
		Votes:     exportedVotes,
	}
}
