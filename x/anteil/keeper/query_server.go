package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"google.golang.org/protobuf/encoding/protojson"
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
	bz, err := protojson.Marshal(params)
	if err != nil {
		return nil, err
	}
	return &anteilv1.QueryParamsResponse{Json: string(bz)}, nil
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
	_ = sdkquery.PageRequest{}
	return &anteilv1.QueryTradesResponse{Trades: nil, Pagination: nil}, nil
}

func (s QueryServer) Auctions(ctx context.Context, req *anteilv1.QueryAuctionsRequest) (*anteilv1.QueryAuctionsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	auctions, err := s.k.GetAllAuctions(sdkCtx)
	if err != nil {
		return nil, err
	}
	return &anteilv1.QueryAuctionsResponse{Auctions: auctions, Pagination: nil}, nil
}
