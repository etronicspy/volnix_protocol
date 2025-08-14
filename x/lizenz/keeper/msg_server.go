package keeper

import (
	"context"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
)

type MsgServer struct {
	k Keeper
	lizenzv1.UnimplementedMsgServer
}

func NewMsgServer(k Keeper) MsgServer { return MsgServer{k: k} }

var _ lizenzv1.MsgServer = MsgServer{}

func (s MsgServer) ActivateLZN(ctx context.Context, req *lizenzv1.MsgActivateLZN) (*lizenzv1.MsgActivateLZNResponse, error) {
	return &lizenzv1.MsgActivateLZNResponse{Success: true}, nil
}

func (s MsgServer) DeactivateLZN(ctx context.Context, req *lizenzv1.MsgDeactivateLZN) (*lizenzv1.MsgDeactivateLZNResponse, error) {
	return &lizenzv1.MsgDeactivateLZNResponse{Success: true}, nil
}

func (MsgServer) mustEmbedUnimplementedMsgServer() {}
