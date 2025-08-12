package keeper

import (
	"context"

	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	anteilv1 "github.com/helvetia-protocol/helvetia-protocol/proto/gen/go/helvetia/anteil/v1"
)

type QueryServer struct {
	k Keeper
	anteilv1.UnimplementedQueryServer
}

func NewQueryServer(k Keeper) QueryServer { return QueryServer{k: k} }

var _ anteilv1.QueryServer = QueryServer{}

func (s QueryServer) Params(ctx context.Context, _ *anteilv1.QueryParamsRequest) (*anteilv1.QueryParamsResponse, error) {
	return &anteilv1.QueryParamsResponse{Json: "{}"}, nil
}

func (s QueryServer) Orders(ctx context.Context, req *anteilv1.QueryOrdersRequest) (*anteilv1.QueryOrdersResponse, error) {
	_ = sdkquery.PageRequest{}
	return &anteilv1.QueryOrdersResponse{Orders: nil, Pagination: nil}, nil
}

func (s QueryServer) Trades(ctx context.Context, req *anteilv1.QueryTradesRequest) (*anteilv1.QueryTradesResponse, error) {
	_ = sdkquery.PageRequest{}
	return &anteilv1.QueryTradesResponse{Trades: nil, Pagination: nil}, nil
}

func (s QueryServer) Auctions(ctx context.Context, req *anteilv1.QueryAuctionsRequest) (*anteilv1.QueryAuctionsResponse, error) {
	_ = sdkquery.PageRequest{}
	return &anteilv1.QueryAuctionsResponse{Auctions: nil, Pagination: nil}, nil
}
