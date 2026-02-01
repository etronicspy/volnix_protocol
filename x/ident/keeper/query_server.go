package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

type QueryServer struct {
	k Keeper
	identv1.UnimplementedQueryServer
}

func NewQueryServer(k *Keeper) QueryServer { return QueryServer{k: *k} }

var _ identv1.QueryServer = QueryServer{}

func (s QueryServer) Params(ctx context.Context, _ *identv1.QueryParamsRequest) (*identv1.QueryParamsResponse, error) {
	// Simple stub implementation
	return &identv1.QueryParamsResponse{
		Params: &identv1.Params{
			MaxIdentitiesPerAddress:     5,
			RequireIdentityVerification: true,
			DefaultVerificationProvider: "default-provider",
		},
	}, nil
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

	return &identv1.QueryVerifiedAccountResponse{VerifiedAccount: account}, nil
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
		VerifiedAccounts: accounts,
		Pagination:       pagination,
	}, nil
}

// VerifiedAccountsByRole method removed - not in current protobuf definition

func (s QueryServer) VerificationProviders(ctx context.Context, req *identv1.QueryVerificationProvidersRequest) (*identv1.QueryVerificationProvidersResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	list, err := s.k.GetAllVerificationProviders(sdkCtx)
	if err != nil {
		return nil, err
	}
	// Convert keeper VerificationProvider to proto
	protoList := make([]*identv1.VerificationProvider, 0, len(list))
	for _, p := range list {
		protoList = append(protoList, &identv1.VerificationProvider{
			ProviderId:        p.ProviderID,
			ProviderName:      p.ProviderName,
			ProviderPublicKey: p.PublicKey,
			IsAccredited:      p.IsActive,
			AccreditationDate: p.RegistrationTime,
			AccreditationHash: p.AccreditationHash,
		})
	}
	// Simple pagination if requested
	if req.Pagination != nil {
		offset := req.Pagination.Offset
		limit := req.Pagination.Limit
		if limit > 0 && offset < uint64(len(protoList)) {
			end := offset + limit
			if end > uint64(len(protoList)) {
				end = uint64(len(protoList))
			}
			protoList = protoList[offset:end]
		}
	}
	return &identv1.QueryVerificationProvidersResponse{
		VerificationProviders: protoList,
		Pagination:            nil,
	}, nil
}

func (s QueryServer) mustEmbedUnimplementedQueryServer() {}
