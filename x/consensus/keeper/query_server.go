package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
)

// QueryServer implements consensus module Query service.
type QueryServer struct {
	k Keeper
	consensusv1.UnimplementedQueryServer
}

// NewQueryServer constructs a new QueryServer.
func NewQueryServer(k Keeper) QueryServer { return QueryServer{k: k} }

var _ consensusv1.QueryServer = QueryServer{}

// Params returns module parameters.
func (s QueryServer) Params(ctx context.Context, _ *consensusv1.QueryParamsRequest) (*consensusv1.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := s.k.GetParams(sdkCtx)
	return &consensusv1.QueryParamsResponse{Params: params}, nil
}

// Validators returns all validators tracked by the consensus module.
func (s QueryServer) Validators(ctx context.Context, _ *consensusv1.QueryValidatorsRequest) (*consensusv1.QueryValidatorsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	validators := s.k.GetAllValidators(sdkCtx)
	return &consensusv1.QueryValidatorsResponse{Validators: validators}, nil
}

