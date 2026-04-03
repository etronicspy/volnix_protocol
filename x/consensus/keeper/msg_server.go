package keeper

import (
	"context"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

var consensusModuleAuthority = authtypes.NewModuleAddress(types.ModuleName).String()

// MsgServer implements consensus module Msg service.
type MsgServer struct {
	k Keeper
	consensusv1.UnimplementedMsgServer
}

// NewMsgServer constructs a new MsgServer.
func NewMsgServer(k Keeper) MsgServer { return MsgServer{k: k} }

var _ consensusv1.MsgServer = MsgServer{}

// UpdateConsensusState updates the consensus state
func (s MsgServer) UpdateConsensusState(ctx context.Context, req *consensusv1.MsgUpdateConsensusState) (*consensusv1.MsgUpdateConsensusStateResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.Authority != consensusModuleAuthority {
		return nil, types.ErrUnauthorized
	}

	err := s.k.UpdateConsensusState(sdkCtx, req.CurrentHeight, req.TotalAntBurned, req.ActiveValidators)
	if err != nil {
		return nil, err
	}

	return &consensusv1.MsgUpdateConsensusStateResponse{}, nil
}

// SetValidatorWeight sets validator weight
func (s MsgServer) SetValidatorWeight(ctx context.Context, req *consensusv1.MsgSetValidatorWeight) (*consensusv1.MsgSetValidatorWeightResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.Authority != consensusModuleAuthority {
		return nil, types.ErrUnauthorized
	}

	err := s.k.SetValidatorWeight(sdkCtx, req.Validator, req.Weight)
	if err != nil {
		return nil, err
	}

	return &consensusv1.MsgSetValidatorWeightResponse{}, nil
}

// DeclarePerHeightBurn handles a validator's burn declaration for the current block height (§5.4).
func (s MsgServer) DeclarePerHeightBurn(ctx context.Context, req *consensusv1.MsgDeclarePerHeightBurn) (*consensusv1.MsgDeclarePerHeightBurnResponse, error) {
	_ = sdk.UnwrapSDKContext(ctx)

	// TODO: implement per-height burn logic (global cap, fee split)
	return &consensusv1.MsgDeclarePerHeightBurnResponse{
		IncI:    "0",
		Success: true,
	}, nil
}

// RegisterConsensusMapping registers a consensus_pubkey <-> account mapping (§4.2).
func (s MsgServer) RegisterConsensusMapping(ctx context.Context, req *consensusv1.MsgRegisterConsensusMapping) (*consensusv1.MsgRegisterConsensusMappingResponse, error) {
	_ = sdk.UnwrapSDKContext(ctx)

	// TODO: derive consensus address from pubkey and persist mapping
	return &consensusv1.MsgRegisterConsensusMappingResponse{
		ConsensusAddress: "",
		Success:          true,
	}, nil
}
