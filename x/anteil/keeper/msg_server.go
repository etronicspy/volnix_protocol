package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

type MsgServer struct {
	k Keeper
	anteilv1.UnimplementedMsgServer
}

func NewMsgServer(k *Keeper) MsgServer { return MsgServer{k: *k} }

var _ anteilv1.MsgServer = MsgServer{}

func (s MsgServer) PlaceOrder(ctx context.Context, req *anteilv1.MsgPlaceOrder) (*anteilv1.MsgPlaceOrderResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create order
	order := types.NewOrder(
		req.Owner,
		req.OrderType,
		req.OrderSide,
		req.AntAmount,
		req.Price,
		req.IdentityHash,
	)

	// Place order using keeper
	err := s.k.CreateOrder(sdkCtx, order)
	if err != nil {
		return nil, err
	}

	return &anteilv1.MsgPlaceOrderResponse{
		Success: true,
		OrderId: order.OrderId,
	}, nil
}

func (s MsgServer) CancelOrder(ctx context.Context, req *anteilv1.MsgCancelOrder) (*anteilv1.MsgCancelOrderResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Cancel order using keeper
	err := s.k.CancelOrder(sdkCtx, req.OrderId)
	if err != nil {
		return nil, err
	}

	return &anteilv1.MsgCancelOrderResponse{
		Success: true,
		Status:  "cancelled",
	}, nil
}

func (s MsgServer) PlaceBid(ctx context.Context, req *anteilv1.MsgPlaceBid) (*anteilv1.MsgPlaceBidResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Place bid using keeper
	err := s.k.PlaceBid(sdkCtx, req.AuctionId, req.Bidder, req.Amount)
	if err != nil {
		return nil, err
	}

	// Generate bid ID (in real implementation this would be returned from keeper)
	bidId := req.AuctionId + "_" + req.Bidder + "_" + string(rune(sdkCtx.BlockHeight()))

	return &anteilv1.MsgPlaceBidResponse{
		Success: true,
		BidId:   bidId,
	}, nil
}

func (s MsgServer) SettleAuction(ctx context.Context, req *anteilv1.MsgSettleAuction) (*anteilv1.MsgSettleAuctionResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Settle auction using keeper
	err := s.k.SettleAuction(sdkCtx, req.AuctionId)
	if err != nil {
		return nil, err
	}

	return &anteilv1.MsgSettleAuctionResponse{
		Success: true,
	}, nil
}
