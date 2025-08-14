package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type QueryServer struct {
	k Keeper
	lizenzv1.UnimplementedQueryServer
}

func NewQueryServer(k Keeper) QueryServer { return QueryServer{k: k} }

var _ lizenzv1.QueryServer = QueryServer{}

func (s QueryServer) Params(ctx context.Context, _ *lizenzv1.QueryParamsRequest) (*lizenzv1.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := s.k.GetParams(sdkCtx).ToProto()
	bz, err := protojson.Marshal(params)
	if err != nil {
		return nil, err
	}
	return &lizenzv1.QueryParamsResponse{Json: string(bz)}, nil
}

func (s QueryServer) Activated(ctx context.Context, req *lizenzv1.QueryActivatedRequest) (*lizenzv1.QueryActivatedResponse, error) {
	_ = sdkquery.PageRequest{}
	return &lizenzv1.QueryActivatedResponse{Items: nil, Pagination: nil}, nil
}

func (s QueryServer) MOA(ctx context.Context, req *lizenzv1.QueryMOARequest) (*lizenzv1.QueryMOAResponse, error) {
	return &lizenzv1.QueryMOAResponse{Status: "unknown"}, nil
}
