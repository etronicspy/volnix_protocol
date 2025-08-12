package keeper

import (
	"context"

	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	lizenzv1 "github.com/helvetia-protocol/helvetia-protocol/proto/gen/go/helvetia/lizenz/v1"
)

type QueryServer struct {
	k Keeper
	lizenzv1.UnimplementedQueryServer
}

func NewQueryServer(k Keeper) QueryServer { return QueryServer{k: k} }

var _ lizenzv1.QueryServer = QueryServer{}

func (s QueryServer) Params(ctx context.Context, _ *lizenzv1.QueryParamsRequest) (*lizenzv1.QueryParamsResponse, error) {
	return &lizenzv1.QueryParamsResponse{Json: "{}"}, nil
}

func (s QueryServer) Activated(ctx context.Context, req *lizenzv1.QueryActivatedRequest) (*lizenzv1.QueryActivatedResponse, error) {
	_ = sdkquery.PageRequest{}
	return &lizenzv1.QueryActivatedResponse{Items: nil, Pagination: nil}, nil
}

func (s QueryServer) MOA(ctx context.Context, req *lizenzv1.QueryMOARequest) (*lizenzv1.QueryMOAResponse, error) {
	return &lizenzv1.QueryMOAResponse{Status: "unknown"}, nil
}
