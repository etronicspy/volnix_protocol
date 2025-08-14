package keeper

import (
	"context"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
)

type MsgServer struct {
	k Keeper
	identv1.UnimplementedMsgServer
}

func NewMsgServer(k Keeper) MsgServer { return MsgServer{k: k} }

var _ identv1.MsgServer = MsgServer{}

func (s MsgServer) VerifyIdentity(ctx context.Context, req *identv1.MsgVerifyIdentity) (*identv1.MsgVerifyIdentityResponse, error) {
	// TODO: implement logic in future iterations
	return &identv1.MsgVerifyIdentityResponse{Success: true}, nil
}

func (s MsgServer) MigrateRole(ctx context.Context, req *identv1.MsgMigrateRole) (*identv1.MsgMigrateRoleResponse, error) {
	return &identv1.MsgMigrateRoleResponse{Success: true}, nil
}

func (s MsgServer) ChangeRole(ctx context.Context, req *identv1.MsgChangeRole) (*identv1.MsgChangeRoleResponse, error) {
	return &identv1.MsgChangeRoleResponse{Success: true}, nil
}
