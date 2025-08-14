package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

type MsgServer struct {
	k *Keeper
	lizenzv1.UnimplementedMsgServer
}

func NewMsgServer(k *Keeper) MsgServer { return MsgServer{k: k} }

var _ lizenzv1.MsgServer = MsgServer{}

func (s MsgServer) ActivateLZN(ctx context.Context, req *lizenzv1.MsgActivateLZN) (*lizenzv1.MsgActivateLZNResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req.Validator == "" {
		return nil, types.ErrEmptyValidator
	}
	if req.Amount == "" {
		return nil, types.ErrEmptyAmount
	}
	if req.IdentityHash == "" {
		return nil, types.ErrEmptyIdentityHash
	}

	// Check if LZN already exists
	existingLizenz, err := s.k.GetActivatedLizenz(sdkCtx, req.Validator)
	if err == nil && existingLizenz != nil {
		return nil, types.ErrLizenzAlreadyExists
	}

	// Create new activated LZN
	lizenz := types.NewActivatedLizenz(req.Validator, req.Amount, req.IdentityHash)

	// Store the LZN
	if err := s.k.SetActivatedLizenz(sdkCtx, lizenz); err != nil {
		return nil, err
	}

	// Create initial MOA status
	moaStatus := types.NewMOAStatus(req.Validator, "100.0", "50.0") // Default values
	if err := s.k.SetMOAStatus(sdkCtx, moaStatus); err != nil {
		return nil, err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"lizenz_activated",
			sdk.NewAttribute("validator", req.Validator),
			sdk.NewAttribute("amount", req.Amount),
			sdk.NewAttribute("identity_hash", req.IdentityHash),
		),
	)

	return &lizenzv1.MsgActivateLZNResponse{
		Success:      true,
		ActivationId: fmt.Sprintf("lzn_%s_%d", req.Validator, sdkCtx.BlockHeight()),
	}, nil
}

func (s MsgServer) DeactivateLZN(ctx context.Context, req *lizenzv1.MsgDeactivateLZN) (*lizenzv1.MsgDeactivateLZNResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req.Validator == "" {
		return nil, types.ErrEmptyValidator
	}
	if req.Amount == "" {
		return nil, types.ErrEmptyAmount
	}
	if req.Reason == "" {
		return nil, types.ErrEmptyReason
	}

	// Check if activated LZN exists
	if _, err := s.k.GetActivatedLizenz(sdkCtx, req.Validator); err != nil {
		return nil, types.ErrLizenzNotFound
	}

	// Create deactivating LZN
	deactivatingLizenz := types.NewDeactivatingLizenz(req.Validator, req.Amount, req.Reason)

	// Store the deactivating LZN
	if err := s.k.SetDeactivatingLizenz(sdkCtx, deactivatingLizenz); err != nil {
		return nil, err
	}

	// Remove the activated LZN
	if err := s.k.DeleteActivatedLizenz(sdkCtx, req.Validator); err != nil {
		return nil, err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"lizenz_deactivated",
			sdk.NewAttribute("validator", req.Validator),
			sdk.NewAttribute("amount", req.Amount),
			sdk.NewAttribute("reason", req.Reason),
		),
	)

	return &lizenzv1.MsgDeactivateLZNResponse{
		Success:        true,
		DeactivationId: fmt.Sprintf("lzn_deact_%s_%d", req.Validator, sdkCtx.BlockHeight()),
	}, nil
}
