package keeper

import (
	"context"

	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	identv1 "github.com/helvetia-protocol/helvetia-protocol/proto/gen/go/helvetia/ident/v1"
)

type QueryServer struct {
	k Keeper
	identv1.UnimplementedQueryServer
}

func NewQueryServer(k Keeper) QueryServer { return QueryServer{k: k} }

var _ identv1.QueryServer = QueryServer{}

func (s QueryServer) Params(ctx context.Context, _ *identv1.QueryParamsRequest) (*identv1.QueryParamsResponse, error) {
	// placeholder: return empty JSON string until real Params type is wired to protobuf
	return &identv1.QueryParamsResponse{Json: "{}"}, nil
}

func (s QueryServer) VerifiedAccount(ctx context.Context, req *identv1.QueryVerifiedAccountRequest) (*identv1.QueryVerifiedAccountResponse, error) {
	return &identv1.QueryVerifiedAccountResponse{Account: nil}, nil
}

func (s QueryServer) VerifiedAccounts(ctx context.Context, req *identv1.QueryVerifiedAccountsRequest) (*identv1.QueryVerifiedAccountsResponse, error) {
	_ = sdkquery.PageRequest{}
	return &identv1.QueryVerifiedAccountsResponse{Accounts: nil, Pagination: nil}, nil
}

func (s QueryServer) VerifiedAccountsByRole(ctx context.Context, req *identv1.QueryVerifiedAccountsByRoleRequest) (*identv1.QueryVerifiedAccountsByRoleResponse, error) {
	_ = sdkquery.PageRequest{}
	return &identv1.QueryVerifiedAccountsByRoleResponse{Accounts: nil, Pagination: nil}, nil
}
