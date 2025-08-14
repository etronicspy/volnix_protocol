package keeper

import (
	"context"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
)

type MsgServer struct {
	k Keeper
	anteilv1.UnimplementedMsgServer
}

func NewMsgServer(k Keeper) MsgServer { return MsgServer{k: k} }

var _ anteilv1.MsgServer = MsgServer{}

func (s MsgServer) PlaceOrder(ctx context.Context, req *anteilv1.MsgPlaceOrder) (*anteilv1.MsgPlaceOrderResponse, error) {
	return &anteilv1.MsgPlaceOrderResponse{OrderId: "stub"}, nil
}

func (s MsgServer) CancelOrder(ctx context.Context, req *anteilv1.MsgCancelOrder) (*anteilv1.MsgCancelOrderResponse, error) {
	return &anteilv1.MsgCancelOrderResponse{Success: true}, nil
}

func (s MsgServer) PlaceBid(ctx context.Context, req *anteilv1.MsgPlaceBid) (*anteilv1.MsgPlaceBidResponse, error) {
	return &anteilv1.MsgPlaceBidResponse{Success: true}, nil
}

func (MsgServer) mustEmbedUnimplementedMsgServer() {}
