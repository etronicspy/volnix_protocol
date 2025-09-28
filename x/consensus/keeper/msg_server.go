package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
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
	bc, err := s.k.SelectBlockCreator(sdkCtx, nextHeight)
	if err != nil {
		return nil, err
	}

	return &consensusv1.MsgSelectBlockCreatorResponse{SelectedValidator: bc.Validator}, nil
}

