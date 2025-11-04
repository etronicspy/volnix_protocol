package keeper

import (
	"context"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
)

type QueryServer struct {
	lizenzv1.UnimplementedQueryServer
	k *Keeper
}

func NewQueryServer(k *Keeper) QueryServer {
	return QueryServer{k: k}
}

var _ lizenzv1.QueryServer = (*QueryServer)(nil)

func (s QueryServer) Params(ctx context.Context, req *lizenzv1.QueryParamsRequest) (*lizenzv1.QueryParamsResponse, error) {
	// Simple stub implementation
	return &lizenzv1.QueryParamsResponse{
		Params: &lizenzv1.Params{
			MaxActivatedPerValidator: 10,
			ActivityCoefficient:      "1.0",
		},
	}, nil
}

func (s QueryServer) ActivatedLizenz(ctx context.Context, req *lizenzv1.QueryActivatedLizenzRequest) (*lizenzv1.QueryActivatedLizenzResponse, error) {
	// Simple stub implementation
	return &lizenzv1.QueryActivatedLizenzResponse{
		ActivatedLizenz: &lizenzv1.ActivatedLizenz{
			Validator: req.Validator,
			Amount:    "1000000ulzn",
		},
	}, nil
}

func (s QueryServer) AllActivatedLizenz(ctx context.Context, req *lizenzv1.QueryAllActivatedLizenzRequest) (*lizenzv1.QueryAllActivatedLizenzResponse, error) {
	// Simple stub implementation
	return &lizenzv1.QueryAllActivatedLizenzResponse{
		ActivatedLizenz: []*lizenzv1.ActivatedLizenz{},
		Pagination:      nil,
	}, nil
}

func (s QueryServer) DeactivatingLizenz(ctx context.Context, req *lizenzv1.QueryDeactivatingLizenzRequest) (*lizenzv1.QueryDeactivatingLizenzResponse, error) {
	// Simple stub implementation
	return &lizenzv1.QueryDeactivatingLizenzResponse{
		DeactivatingLizenz: &lizenzv1.DeactivatingLizenz{
			Validator: req.Validator,
			Amount:    "1000000ulzn",
		},
	}, nil
}

func (s QueryServer) MOAStatus(ctx context.Context, req *lizenzv1.QueryMOAStatusRequest) (*lizenzv1.QueryMOAStatusResponse, error) {
	// Simple stub implementation
	return &lizenzv1.QueryMOAStatusResponse{
		MoaStatus: &lizenzv1.MOAStatus{
			Validator:   req.Validator,
			IsActive:    true,
			IsCompliant: true,
		},
	}, nil
}

func (s QueryServer) ValidatorIntegration(ctx context.Context, req *lizenzv1.QueryValidatorIntegrationRequest) (*lizenzv1.QueryValidatorIntegrationResponse, error) {
	// Simple stub implementation
	return &lizenzv1.QueryValidatorIntegrationResponse{
		ValidatorIntegration: &lizenzv1.ValidatorIntegration{
			Validator: req.Validator,
		},
	}, nil
}

// mustEmbedUnimplementedQueryServer is embedded via UnimplementedQueryServer
