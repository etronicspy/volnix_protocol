package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
	"google.golang.org/protobuf/encoding/protojson"
)

type QueryServer struct {
	k *Keeper
	lizenzv1.UnimplementedQueryServer
}

func NewQueryServer(k *Keeper) QueryServer { return QueryServer{k: k} }

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
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	activatedLizenzs, err := s.k.GetAllActivatedLizenz(sdkCtx)
	if err != nil {
		return nil, err
	}

	// Handle pagination
	var pagination *sdkquery.PageResponse
	if req.Pagination != nil {
		offset := req.Pagination.Offset
		limit := req.Pagination.Limit

		if limit > 0 && offset < uint64(len(activatedLizenzs)) {
			end := offset + limit
			if end > uint64(len(activatedLizenzs)) {
				end = uint64(len(activatedLizenzs))
			}
			activatedLizenzs = activatedLizenzs[offset:end]
		}

		pagination = &sdkquery.PageResponse{
			NextKey: nil, // Simplified pagination
		}
	}

	return &lizenzv1.QueryActivatedResponse{
		Items:      activatedLizenzs,
		Pagination: pagination,
	}, nil
}

func (s QueryServer) MOA(ctx context.Context, req *lizenzv1.QueryMOARequest) (*lizenzv1.QueryMOAResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.Validator == "" {
		return nil, types.ErrEmptyValidator
	}

	moaStatus, err := s.k.GetMOAStatus(sdkCtx, req.Validator)
	if err != nil {
		return nil, err
	}

	return &lizenzv1.QueryMOAResponse{Status: moaStatus}, nil
}

func (s QueryServer) Deactivating(ctx context.Context, req *lizenzv1.QueryDeactivatingRequest) (*lizenzv1.QueryDeactivatingResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	deactivatingLizenzs, err := s.k.GetAllDeactivatingLizenz(sdkCtx)
	if err != nil {
		return nil, err
	}

	// Handle pagination
	var pagination *sdkquery.PageResponse
	if req.Pagination != nil {
		offset := req.Pagination.Offset
		limit := req.Pagination.Limit

		if limit > 0 && offset < uint64(len(deactivatingLizenzs)) {
			end := offset + limit
			if end > uint64(len(deactivatingLizenzs)) {
				end = uint64(len(deactivatingLizenzs))
			}
			deactivatingLizenzs = deactivatingLizenzs[offset:end]
		}

		pagination = &sdkquery.PageResponse{
			NextKey: nil, // Simplified pagination
		}
	}

	return &lizenzv1.QueryDeactivatingResponse{
		Items:      deactivatingLizenzs,
		Pagination: pagination,
	}, nil
}

func (s QueryServer) ValidatorLZN(ctx context.Context, req *lizenzv1.QueryValidatorLZNRequest) (*lizenzv1.QueryValidatorLZNResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.Validator == "" {
		return nil, types.ErrEmptyValidator
	}

	activatedLizenz, _ := s.k.GetActivatedLizenz(sdkCtx, req.Validator)
	deactivatingLizenzs, _ := s.k.GetAllDeactivatingLizenz(sdkCtx)
	moaStatus, _ := s.k.GetMOAStatus(sdkCtx, req.Validator)

	// Filter deactivating LZN for this validator
	var validatorDeactivating []*lizenzv1.DeactivatingLizenz
	for _, dl := range deactivatingLizenzs {
		if dl.Validator == req.Validator {
			validatorDeactivating = append(validatorDeactivating, dl)
		}
	}

	return &lizenzv1.QueryValidatorLZNResponse{
		Activated:    activatedLizenz,
		Deactivating: validatorDeactivating,
		MoaStatus:    moaStatus,
	}, nil
}
