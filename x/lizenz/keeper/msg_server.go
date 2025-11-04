package keeper

import (
	"context"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
)

type MsgServer struct {
	lizenzv1.UnimplementedMsgServer
	k *Keeper
}

func NewMsgServer(k *Keeper) MsgServer {
	return MsgServer{k: k}
}

var _ lizenzv1.MsgServer = (*MsgServer)(nil)

func (s MsgServer) ActivateLZN(ctx context.Context, req *lizenzv1.MsgActivateLZN) (*lizenzv1.MsgActivateLZNResponse, error) {
	// Simple stub implementation
	return &lizenzv1.MsgActivateLZNResponse{
		Success:      true,
		ActivationId: "activation-123",
	}, nil
}

func (s MsgServer) DeactivateLZN(ctx context.Context, req *lizenzv1.MsgDeactivateLZN) (*lizenzv1.MsgDeactivateLZNResponse, error) {
	// Simple stub implementation
	return &lizenzv1.MsgDeactivateLZNResponse{
		Success:        true,
		DeactivationId: "deactivation-123",
	}, nil
}
