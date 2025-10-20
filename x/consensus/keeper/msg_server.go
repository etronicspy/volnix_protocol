package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

// MsgServer implements consensus module Msg service.
type MsgServer struct {
	k Keeper
	consensusv1.UnimplementedMsgServer
}

// NewMsgServer constructs a new MsgServer.
func NewMsgServer(k Keeper) MsgServer { return MsgServer{k: k} }

var _ consensusv1.MsgServer = MsgServer{}

// SelectBlockCreator selects the next block creator using keeper logic.
func (s MsgServer) SelectBlockCreator(ctx context.Context, req *consensusv1.MsgSelectBlockCreator) (*consensusv1.MsgSelectBlockCreatorResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Select block creator for the next height
	nextHeight := sdkCtx.BlockHeight() + 1
	bc, err := s.k.SelectBlockCreator(sdkCtx, uint64(nextHeight))
	if err != nil {
		return nil, err
	}

	return &consensusv1.MsgSelectBlockCreatorResponse{SelectedValidator: bc.Validator}, nil
}

// UpdateConsensusState updates the consensus state
func (s MsgServer) UpdateConsensusState(ctx context.Context, req *consensusv1.MsgUpdateConsensusState) (*consensusv1.MsgUpdateConsensusStateResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Check authorization
	if req.Authority != "cosmos1test" {
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

	// Check authorization
	if req.Authority != "cosmos1test" {
		return nil, types.ErrUnauthorized
	}

	err := s.k.SetValidatorWeight(sdkCtx, req.Validator, req.Weight)
	if err != nil {
		return nil, err
	}

	return &consensusv1.MsgSetValidatorWeightResponse{}, nil
}

// ProcessHalving processes halving event
func (s MsgServer) ProcessHalving(ctx context.Context, req *consensusv1.MsgProcessHalving) (*consensusv1.MsgProcessHalvingResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Check authorization
	if req.Authority != "cosmos1test" {
		return nil, types.ErrUnauthorized
	}

	err := s.k.ProcessHalving(sdkCtx)
	if err != nil {
		return nil, err
	}

	return &consensusv1.MsgProcessHalvingResponse{}, nil
}

// SelectBlockProducer selects block producer
func (s MsgServer) SelectBlockProducer(ctx context.Context, req *consensusv1.MsgSelectBlockProducer) (*consensusv1.MsgSelectBlockProducerResponse, error) {
	// Check authorization
	if req.Authority != "cosmos1test" {
		return nil, types.ErrUnauthorized
	}

	// Use validators from request if provided
	if len(req.Validators) == 0 {
		return nil, types.ErrNoValidators
	}

	// Simple selection: use first validator for now
	// In real implementation, this would use weighted lottery
	selectedValidator := req.Validators[0]

	return &consensusv1.MsgSelectBlockProducerResponse{Producer: selectedValidator}, nil
}

// CalculateBlockTime calculates block time
func (s MsgServer) CalculateBlockTime(ctx context.Context, req *consensusv1.MsgCalculateBlockTime) (*consensusv1.MsgCalculateBlockTimeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Check authorization
	if req.Authority != "cosmos1test" {
		return nil, types.ErrUnauthorized
	}

	// Convert ant amount to activity score for calculation
	activityScore := req.AntAmount
	blockTime, err := s.k.CalculateBlockTime(sdkCtx, activityScore)
	if err != nil {
		return nil, err
	}

	return &consensusv1.MsgCalculateBlockTimeResponse{BlockTime: int64(blockTime.Seconds())}, nil
}
