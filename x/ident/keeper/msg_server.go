package keeper

import (
	"context"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
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
	// Simple stub implementation
	return &identv1.MsgVerifyIdentityResponse{
		Success:        true,
		VerificationId: "verification-123",
		IdentityHash:   "hash-456",
	}, nil
}

func (s MsgServer) MigrateRole(ctx context.Context, req *identv1.MsgMigrateRole) (*identv1.MsgMigrateRoleResponse, error) {
	// Simple stub implementation
	return &identv1.MsgMigrateRoleResponse{
		Success:       true,
		MigrationHash: "migration-123",
	}, nil
}

func (s MsgServer) ChangeRole(ctx context.Context, req *identv1.MsgChangeRole) (*identv1.MsgChangeRoleResponse, error) {
	// Simple stub implementation
	return &identv1.MsgChangeRoleResponse{
		Success:    true,
		ChangeHash: "change-123",
	}, nil
}

func (s MsgServer) RegisterVerificationProvider(ctx context.Context, req *identv1.MsgRegisterVerificationProvider) (*identv1.MsgRegisterVerificationProviderResponse, error) {
	// Simple stub implementation
	return &identv1.MsgRegisterVerificationProviderResponse{
		Success:           true,
		AccreditationHash: "accreditation-123",
	}, nil
}
