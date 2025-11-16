package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

type MsgServer struct {
	lizenzv1.UnimplementedMsgServer
	k *Keeper
}

func NewMsgServer(k *Keeper) MsgServer {
	return MsgServer{k: k}
}

var _ lizenzv1.MsgServer = (*MsgServer)(nil)

// ActivateLZN activates a LZN license for a validator
// According to whitepaper: only validators with verified identity can activate LZN
func (s MsgServer) ActivateLZN(ctx context.Context, req *lizenzv1.MsgActivateLZN) (*lizenzv1.MsgActivateLZNResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Validator == "" {
		return nil, types.ErrEmptyValidator
	}
	if req.Amount == "" {
		return nil, types.ErrEmptyAmount
	}

	// Identity verification and role validation happens in SetActivatedLizenz
	// through validateIdentityAndRole() method

	// Create activated LZN
	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:      req.Validator,
		Amount:         req.Amount,
		ActivationTime: timestamppb.Now(),
		LastActivity:   timestamppb.Now(),
		IdentityHash:   req.IdentityHash,
		IsEligibleForRewards: true,
	}

	// Set activated LZN (this will validate identity, role, and limits)
	err := s.k.SetActivatedLizenz(sdkCtx, activatedLizenz)
	if err != nil {
		return nil, err
	}

	// Generate activation ID
	activationId := fmt.Sprintf("activation-%s-%d", req.Validator, sdkCtx.BlockHeight())

	return &lizenzv1.MsgActivateLZNResponse{
		Success:      true,
		ActivationId: activationId,
	}, nil
}

// DeactivateLZN deactivates a LZN license for a validator
func (s MsgServer) DeactivateLZN(ctx context.Context, req *lizenzv1.MsgDeactivateLZN) (*lizenzv1.MsgDeactivateLZNResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Validator == "" {
		return nil, types.ErrEmptyValidator
	}
	if req.Reason == "" {
		return nil, types.ErrEmptyReason
	}

	// Check if LZN exists (validation)
	_, err := s.k.GetActivatedLizenz(sdkCtx, req.Validator)
	if err != nil {
		return nil, err
	}

	// Delete activated LZN
	err = s.k.DeleteActivatedLizenz(sdkCtx, req.Validator)
	if err != nil {
		return nil, err
	}

	// Generate deactivation ID
	deactivationId := fmt.Sprintf("deactivation-%s-%d", req.Validator, sdkCtx.BlockHeight())

	return &lizenzv1.MsgDeactivateLZNResponse{
		Success:        true,
		DeactivationId: deactivationId,
	}, nil
}
