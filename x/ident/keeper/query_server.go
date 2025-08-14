package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
	"google.golang.org/protobuf/encoding/protojson"
)

type QueryServer struct {
	k Keeper
	identv1.UnimplementedQueryServer
}

func NewQueryServer(k *Keeper) QueryServer { return QueryServer{k: *k} }

var _ identv1.QueryServer = QueryServer{}

func (s QueryServer) Params(ctx context.Context, _ *identv1.QueryParamsRequest) (*identv1.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := s.k.GetParams(sdkCtx).ToProto()
	bz, err := protojson.Marshal(params)
	if err != nil {
		return nil, err
	}
	return &identv1.QueryParamsResponse{Json: string(bz)}, nil
}

func (s QueryServer) VerifiedAccount(ctx context.Context, req *identv1.QueryVerifiedAccountRequest) (*identv1.QueryVerifiedAccountResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.Address == "" {
		return nil, types.ErrEmptyAddress
	}

	account, err := s.k.GetVerifiedAccount(sdkCtx, req.Address)
	if err != nil {
		return nil, err
	}

	return &identv1.QueryVerifiedAccountResponse{Account: account}, nil
}

func (s QueryServer) VerifiedAccounts(ctx context.Context, req *identv1.QueryVerifiedAccountsRequest) (*identv1.QueryVerifiedAccountsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	accounts, err := s.k.GetAllVerifiedAccounts(sdkCtx)
	if err != nil {
		return nil, err
	}

	// Handle pagination
	var pagination *sdkquery.PageResponse
	if req.Pagination != nil {
		// Simple pagination implementation
		offset := req.Pagination.Offset
		limit := req.Pagination.Limit

		if limit > 0 && offset < uint64(len(accounts)) {
			end := offset + limit
			if end > uint64(len(accounts)) {
				end = uint64(len(accounts))
			}
			accounts = accounts[offset:end]
		}

		pagination = &sdkquery.PageResponse{
			NextKey: nil, // Simplified pagination
		}
	}

	return &identv1.QueryVerifiedAccountsResponse{
		Accounts:   accounts,
		Pagination: pagination,
	}, nil
}

func (s QueryServer) VerifiedAccountsByRole(ctx context.Context, req *identv1.QueryVerifiedAccountsByRoleRequest) (*identv1.QueryVerifiedAccountsByRoleResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.Role == identv1.Role_ROLE_UNSPECIFIED {
		return nil, types.ErrInvalidRole
	}

	accounts, err := s.k.GetVerifiedAccountsByRole(sdkCtx, req.Role)
	if err != nil {
		return nil, err
	}

	// Handle pagination
	var pagination *sdkquery.PageResponse
	if req.Pagination != nil {
		// Simple pagination implementation
		offset := req.Pagination.Offset
		limit := req.Pagination.Limit

		if limit > 0 && offset < uint64(len(accounts)) {
			end := offset + limit
			if end > uint64(len(accounts)) {
				end = uint64(len(accounts))
			}
			accounts = accounts[offset:end]
		}

		pagination = &sdkquery.PageResponse{
			NextKey: nil, // Simplified pagination
		}
	}

	return &identv1.QueryVerifiedAccountsByRoleResponse{
		Accounts:   accounts,
		Pagination: pagination,
	}, nil
}
