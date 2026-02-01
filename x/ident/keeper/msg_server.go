package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/types/known/timestamppb"

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

	// Enhanced validation with security checks
	// According to whitepaper: "через аккредитованных провайдеров"
	if err := s.k.ValidateVerificationRequest(sdkCtx, req.Address, req.ZkpProof, req.VerificationProvider, req.DesiredRole); err != nil {
		return nil, err
	}

	// Generate identity hash from ZKP proof
	identityHash := fmt.Sprintf("hash-%s", req.ZkpProof[:16])

	// IMPROVED: Check for duplicate identity hash BEFORE creating account
	// This prevents identity reuse attacks
	if err := s.k.CheckDuplicateIdentityHash(sdkCtx, identityHash, req.Address); err != nil {
		return nil, err
	}

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

	// SECURITY: Require ZKP proof for role changes to prevent unauthorized escalation
	// According to whitepaper: Role changes require identity verification
	if req.ZkpProof == "" {
		return nil, fmt.Errorf("ZKP proof is required for role changes")
	}
	
	// Get current account to verify it exists
	account, err := s.k.GetVerifiedAccount(sdkCtx, req.Address)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}
	
	// SECURITY: Validate ZKP proof for the role change
	// This prevents unauthorized role escalation attacks
	if err := s.k.ValidateRoleChangeProof(sdkCtx, req.Address, account.IdentityHash, req.ZkpProof, req.NewRole); err != nil {
		return nil, fmt.Errorf("invalid ZKP proof for role change: %w", err)
	}

	// Change account role
	err = s.k.ChangeAccountRole(sdkCtx, req.Address, req.NewRole)
	if err != nil {
		return nil, err
	}

	return &identv1.MsgChangeRoleResponse{
		Success:    true,
		ChangeHash: fmt.Sprintf("change-%s-%d", req.Address, req.NewRole),
	}, nil
}

func (s MsgServer) RegisterVerificationProvider(ctx context.Context, req *identv1.MsgRegisterVerificationProvider) (*identv1.MsgRegisterVerificationProviderResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.ProviderId == "" {
		return nil, fmt.Errorf("provider_id cannot be empty")
	}
	if req.ProviderName == "" {
		return nil, fmt.Errorf("provider_name cannot be empty")
	}
	if req.ProviderPublicKey == "" {
		return nil, fmt.Errorf("provider_public_key cannot be empty")
	}

	// Deterministic accreditation hash from provider id and proof (no ZKP verification here)
	payload := req.ProviderId + req.AccreditationProof
	hash := sha256.Sum256([]byte(payload))
	accreditationHash := hex.EncodeToString(hash[:])

	if err := s.k.SetAccreditationRecord(sdkCtx, accreditationHash, true); err != nil {
		return nil, fmt.Errorf("failed to set accreditation record: %w", err)
	}

	provider := &VerificationProvider{
		ProviderID:         req.ProviderId,
		ProviderName:       req.ProviderName,
		PublicKey:          req.ProviderPublicKey,
		AccreditationHash:  accreditationHash,
		IsActive:           true,
		RegistrationTime:  timestamppb.Now(),
		ExpirationTime:     nil,
	}
	if err := s.k.SetVerificationProvider(sdkCtx, provider); err != nil {
		return nil, fmt.Errorf("failed to set verification provider: %w", err)
	}

	return &identv1.MsgRegisterVerificationProviderResponse{
		Success:           true,
		AccreditationHash: accreditationHash,
	}, nil
}
