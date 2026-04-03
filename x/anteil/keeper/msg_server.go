package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

type MsgServer struct {
	anteilv1.UnimplementedMsgServer
	k *Keeper
}

func NewMsgServer(k *Keeper) MsgServer {
	return MsgServer{k: k}
}

var _ anteilv1.MsgServer = (*MsgServer)(nil)

func (s MsgServer) PlaceOrder(ctx context.Context, req *anteilv1.MsgPlaceOrder) (*anteilv1.MsgPlaceOrderResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Owner == "" {
		return nil, types.ErrEmptyOwner
	}
	if req.AntAmount == "" {
		return nil, types.ErrEmptyAntAmount
	}
	if req.Price == "" {
		return nil, types.ErrEmptyPrice
	}

	order := types.NewOrder(
		req.Owner,
		req.OrderType,
		req.OrderSide,
		req.AntAmount,
		req.Price,
		req.IdentityHash,
		sdkCtx.BlockTime(),
	)

	err := s.k.SetOrder(sdkCtx, order)
	if err != nil {
		return nil, err
	}

	return &anteilv1.MsgPlaceOrderResponse{
		Success: true,
		OrderId: order.OrderId,
		Status:  "order placed successfully",
	}, nil
}

func (s MsgServer) CancelOrder(ctx context.Context, req *anteilv1.MsgCancelOrder) (*anteilv1.MsgCancelOrderResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.OrderId == "" {
		return nil, fmt.Errorf("order ID cannot be empty")
	}
	if req.Owner == "" {
		return nil, types.ErrEmptyOwner
	}

	order, err := s.k.GetOrder(sdkCtx, req.OrderId)
	if err != nil {
		return nil, err
	}

	if order.Owner != req.Owner {
		return nil, fmt.Errorf("unauthorized: order owner mismatch")
	}

	err = s.k.CancelOrder(sdkCtx, req.OrderId)
	if err != nil {
		return nil, err
	}

	return &anteilv1.MsgCancelOrderResponse{
		Success: true,
		Status:  "order cancelled successfully",
	}, nil
}

func (s MsgServer) UpdateOrder(ctx context.Context, req *anteilv1.MsgUpdateOrder) (*anteilv1.MsgUpdateOrderResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.OrderId == "" {
		return nil, fmt.Errorf("order ID cannot be empty")
	}
	if req.Owner == "" {
		return nil, types.ErrEmptyOwner
	}

	order, err := s.k.GetOrder(sdkCtx, req.OrderId)
	if err != nil {
		return nil, err
	}

	if order.Owner != req.Owner {
		return nil, fmt.Errorf("unauthorized: order owner mismatch")
	}

	if order.Status != anteilv1.OrderStatus_ORDER_STATUS_OPEN {
		return nil, types.ErrOrderNotOpen
	}

	if req.NewAmount != "" {
		if _, err := math.LegacyNewDecFromStr(req.NewAmount); err != nil {
			return nil, fmt.Errorf("invalid ANT amount: %w", err)
		}
		order.AntAmount = req.NewAmount
	}
	if req.NewPrice != "" {
		if _, err := math.LegacyNewDecFromStr(req.NewPrice); err != nil {
			return nil, fmt.Errorf("invalid price: %w", err)
		}
		order.Price = req.NewPrice
	}

	if err := s.k.UpdateOrder(sdkCtx, order); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"anteil.order_updated",
			sdk.NewAttribute("order_id", order.OrderId),
			sdk.NewAttribute("owner", order.Owner),
			sdk.NewAttribute("ant_amount", order.AntAmount),
			sdk.NewAttribute("price", order.Price),
		),
	)

	return &anteilv1.MsgUpdateOrderResponse{
		Success: true,
		Status:  "order updated successfully",
	}, nil
}

