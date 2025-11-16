package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

type MsgServer struct {
	identv1.UnimplementedMsgServer
	k *Keeper
}

func NewMsgServer(k *Keeper) MsgServer {
	return MsgServer{k: k}
}

var _ identv1.MsgServer = (*MsgServer)(nil)

func (s MsgServer) VerifyIdentity(ctx context.Context, req *identv1.MsgVerifyIdentity) (*identv1.MsgVerifyIdentityResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Address == "" {
		return nil, types.ErrEmptyAddress
	}
	if req.ZkpProof == "" {
		return nil, fmt.Errorf("ZKP proof cannot be empty")
	}

	// Validate role choice: user must choose between CITIZEN or VALIDATOR
	// This is a critical requirement from the whitepaper
	if err := s.k.ValidateRoleChoice(sdkCtx, req.Address, req.DesiredRole); err != nil {
		return nil, err
	}

	// Generate identity hash from ZKP proof
	identityHash := fmt.Sprintf("hash-%s", req.ZkpProof[:16])

	// Create verified account with user's chosen role (not default CITIZEN)
	account := types.NewVerifiedAccount(
		req.Address,
		req.DesiredRole, // Use desired_role from request
		identityHash,
	)

	// Set verified account
	err := s.k.SetVerifiedAccount(sdkCtx, account)
	if err != nil {
		return nil, err
	}

	return &identv1.MsgVerifyIdentityResponse{
		Success:        true,
		VerificationId: fmt.Sprintf("verification-%s", req.Address),
		IdentityHash:   identityHash,
	}, nil
}

func (s MsgServer) MigrateRole(ctx context.Context, req *identv1.MsgMigrateRole) (*identv1.MsgMigrateRoleResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.FromAddress == "" {
		return nil, types.ErrEmptyAddress
	}
	if req.ToAddress == "" {
		return nil, types.ErrEmptyAddress
	}
	if req.ZkpProof == "" {
		return nil, fmt.Errorf("ZKP proof cannot be empty")
	}

	// Get from account to get role
	fromAccount, err := s.k.GetVerifiedAccount(sdkCtx, req.FromAddress)
	if err != nil {
		return nil, err
	}

	// Generate migration hash
	migrationHash := fmt.Sprintf("migration-%s-%s", req.FromAddress, req.ToAddress)

	// Create role migration
	migration := &identv1.RoleMigration{
		FromAddress:   req.FromAddress,
		ToAddress:     req.ToAddress,
		FromRole:      fromAccount.Role,
		ToRole:        fromAccount.Role,
		MigrationHash: migrationHash,
		ZkpProof:      req.ZkpProof,
		IsCompleted:   false,
	}

	// Set role migration
	err = s.k.SetRoleMigration(sdkCtx, migration)
	if err != nil {
		return nil, err
	}

	// Execute role migration
	err = s.k.ExecuteRoleMigration(sdkCtx, req.FromAddress, req.ToAddress)
	if err != nil {
		return nil, err
	}

	return &identv1.MsgMigrateRoleResponse{
		Success:       true,
		MigrationHash: migrationHash,
	}, nil
}

func (s MsgServer) ChangeRole(ctx context.Context, req *identv1.MsgChangeRole) (*identv1.MsgChangeRoleResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Address == "" {
		return nil, types.ErrEmptyAddress
	}
	if req.NewRole == identv1.Role_ROLE_UNSPECIFIED {
		return nil, types.ErrInvalidRole
	}

	// Change account role
	err := s.k.ChangeAccountRole(sdkCtx, req.Address, req.NewRole)
	if err != nil {
		return nil, err
	}

	return &identv1.MsgChangeRoleResponse{
		Success:    true,
		ChangeHash: fmt.Sprintf("change-%s-%d", req.Address, req.NewRole),
	}, nil
}

func (s MsgServer) RegisterVerificationProvider(ctx context.Context, req *identv1.MsgRegisterVerificationProvider) (*identv1.MsgRegisterVerificationProviderResponse, error) {
	// Simple stub implementation
	return &identv1.MsgRegisterVerificationProviderResponse{
		Success:           true,
		AccreditationHash: "accreditation-123",
	}, nil
}
