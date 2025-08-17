package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
	"golang.org/x/crypto/sha3"
)

type MsgServer struct {
	k Keeper
	identv1.UnimplementedMsgServer
}

func NewMsgServer(k *Keeper) MsgServer { return MsgServer{k: *k} }

var _ identv1.MsgServer = MsgServer{}

func (s MsgServer) VerifyIdentity(ctx context.Context, req *identv1.MsgVerifyIdentity) (*identv1.MsgVerifyIdentityResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req.Address == "" {
		return nil, types.ErrEmptyAddress
	}
	if req.ZkpProof == "" {
		return nil, types.ErrEmptyIdentityHash
	}
	if req.VerificationProvider == "" {
		return nil, types.ErrEmptyAddress
	}

	// Check if account already exists
	existingAccount, err := s.k.GetVerifiedAccount(sdkCtx, req.Address)
	if err == nil && existingAccount != nil {
		return nil, types.ErrAccountAlreadyExists
	}

	// Create new verified account (using ZKP proof hash as identity hash)
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(req.ZkpProof))
	identityHash := string(hash.Sum(nil))
	account := types.NewVerifiedAccount(req.Address, identv1.Role_ROLE_CITIZEN, identityHash)

	// Store the account
	if err := s.k.SetVerifiedAccount(sdkCtx, account); err != nil {
		return nil, err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"identity_verified",
			sdk.NewAttribute("address", req.Address),
			sdk.NewAttribute("verification_provider", req.VerificationProvider),
			sdk.NewAttribute("identity_hash", identityHash),
		),
	)

	return &identv1.MsgVerifyIdentityResponse{
		Success:        true,
		VerificationId: "verification_" + req.Address,
		IdentityHash:   identityHash,
	}, nil
}

func (s MsgServer) MigrateRole(ctx context.Context, req *identv1.MsgMigrateRole) (*identv1.MsgMigrateRoleResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req.ToAddress == "" {
		return nil, types.ErrEmptyAddress
	}
	if req.ZkpProof == "" {
		return nil, types.ErrEmptyIdentityHash
	}

	// Get source account from request
	sender := req.FromAddress

	// Get source account
	sourceAccount, err := s.k.GetVerifiedAccount(sdkCtx, sender)
	if err != nil {
		return nil, err
	}

	// Check if target account already exists
	_, err = s.k.GetVerifiedAccount(sdkCtx, req.ToAddress)
	if err == nil {
		return nil, types.ErrAccountAlreadyExists
	}

	// Create new account with migrated role and ZKP proof
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(req.ZkpProof))
	identityHash := string(hash.Sum(nil))
	newAccount := types.NewVerifiedAccount(req.ToAddress, sourceAccount.Role, identityHash)

	// Store the new account
	if err := s.k.SetVerifiedAccount(sdkCtx, newAccount); err != nil {
		return nil, err
	}

	// Delete the old account
	if err := s.k.DeleteVerifiedAccount(sdkCtx, sender); err != nil {
		return nil, err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"role_migrated",
			sdk.NewAttribute("from_address", sender),
			sdk.NewAttribute("to_address", req.ToAddress),
			sdk.NewAttribute("role", sourceAccount.Role.String()),
		),
	)

	return &identv1.MsgMigrateRoleResponse{Success: true}, nil
}

func (s MsgServer) ChangeRole(ctx context.Context, req *identv1.MsgChangeRole) (*identv1.MsgChangeRoleResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req.Address == "" {
		return nil, types.ErrEmptyAddress
	}
	if req.NewRole == identv1.Role_ROLE_UNSPECIFIED {
		return nil, types.ErrInvalidRole
	}

	// Change the role
	if err := s.k.ChangeAccountRole(sdkCtx, req.Address, req.NewRole); err != nil {
		return nil, err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"role_changed",
			sdk.NewAttribute("address", req.Address),
			sdk.NewAttribute("new_role", req.NewRole.String()),
		),
	)

	return &identv1.MsgChangeRoleResponse{Success: true}, nil
}
