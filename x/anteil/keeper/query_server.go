package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
)

type QueryServer struct {
	k Keeper
	anteilv1.UnimplementedQueryServer
}

func NewQueryServer(k *Keeper) QueryServer { return QueryServer{k: *k} }

var _ anteilv1.QueryServer = QueryServer{}

func (s QueryServer) Params(ctx context.Context, _ *anteilv1.QueryParamsRequest) (*anteilv1.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := s.k.GetParams(sdkCtx).ToProto()
	return &anteilv1.QueryParamsResponse{Params: params}, nil
}

func (s QueryServer) Orders(ctx context.Context, req *anteilv1.QueryOrdersRequest) (*anteilv1.QueryOrdersResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var orders []*anteilv1.Order
	var err error
	if req.Owner != "" {
		orders, err = s.k.GetOrdersByOwner(sdkCtx, req.Owner)
	} else {
		orders, err = s.k.GetAllOrders(sdkCtx)
	}
	if err != nil {
		return nil, err
	}
	return &anteilv1.QueryOrdersResponse{Orders: orders, Pagination: nil}, nil
}

func (s QueryServer) Trades(ctx context.Context, req *anteilv1.QueryTradesRequest) (*anteilv1.QueryTradesResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	trades, err := s.k.GetAllTrades(sdkCtx)
	if err != nil {
		return nil, err
	}
	return &anteilv1.QueryTradesResponse{Trades: trades, Pagination: nil}, nil
}

func (s QueryServer) UserPosition(ctx context.Context, req *anteilv1.QueryUserPositionRequest) (*anteilv1.QueryUserPositionResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	position, err := s.k.GetUserPosition(sdkCtx, req.Owner)
	if err != nil {
		return nil, err
	}
	return &anteilv1.QueryUserPositionResponse{Position: position}, nil
}

func (s QueryServer) EpochState(ctx context.Context, _ *anteilv1.QueryEpochStateRequest) (*anteilv1.QueryEpochStateResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	state := s.k.GetEpochState(sdkCtx)
	return &anteilv1.QueryEpochStateResponse{EpochState: state}, nil
}
