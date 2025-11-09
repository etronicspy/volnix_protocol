package keeper

import (
	"context"
	"fmt"

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

	// Validate request
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

	// Create order
	order := types.NewOrder(
		req.Owner,
		req.OrderType,
		req.OrderSide,
		req.AntAmount,
		req.Price,
		req.IdentityHash,
	)

	// Set order in keeper
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

	// Validate request
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.OrderId == "" {
		return nil, fmt.Errorf("order ID cannot be empty")
	}
	if req.Owner == "" {
		return nil, types.ErrEmptyOwner
	}

	// Get order to verify ownership
	order, err := s.k.GetOrder(sdkCtx, req.OrderId)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if order.Owner != req.Owner {
		return nil, fmt.Errorf("unauthorized: order owner mismatch")
	}

	// Cancel order
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
	// Simple stub implementation
	return &anteilv1.MsgUpdateOrderResponse{
		Success: true,
		Status:  "updated",
	}, nil
}

func (s MsgServer) PlaceBid(ctx context.Context, req *anteilv1.MsgPlaceBid) (*anteilv1.MsgPlaceBidResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.AuctionId == "" {
		return nil, types.ErrEmptyAuctionID
	}
	if req.Bidder == "" {
		return nil, types.ErrEmptyBidder
	}
	if req.Amount == "" {
		return nil, types.ErrEmptyBidAmount
	}

	// Place bid
	err := s.k.PlaceBid(sdkCtx, req.AuctionId, req.Bidder, req.Amount)
	if err != nil {
		return nil, err
	}

	// Generate bid ID
	bidId := fmt.Sprintf("bid-%s-%s", req.AuctionId, req.Bidder)

	return &anteilv1.MsgPlaceBidResponse{
		Success: true,
		BidId:   bidId,
		Status:  "bid placed successfully",
	}, nil
}

func (s MsgServer) SettleAuction(ctx context.Context, req *anteilv1.MsgSettleAuction) (*anteilv1.MsgSettleAuctionResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate request
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.AuctionId == "" {
		return nil, types.ErrEmptyAuctionID
	}

	// Get auction to verify it exists and is closed
	auction, err := s.k.GetAuction(sdkCtx, req.AuctionId)
	if err != nil {
		return nil, err
	}

	// Verify auction is closed
	if auction.Status != anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED {
		return nil, types.ErrAuctionNotClosed
	}

	// Settle auction
	err = s.k.SettleAuction(sdkCtx, req.AuctionId)
	if err != nil {
		return nil, err
	}

	return &anteilv1.MsgSettleAuctionResponse{
		Success: true,
		Status:  "auction settled successfully",
	}, nil
}

func (s MsgServer) RegisterMarketMaker(ctx context.Context, req *anteilv1.MsgRegisterMarketMaker) (*anteilv1.MsgRegisterMarketMakerResponse, error) {
	// Simple stub implementation
	return &anteilv1.MsgRegisterMarketMakerResponse{
		Success:       true,
		MarketMakerId: "mm-123",
	}, nil
}

func (s MsgServer) ProvideLiquidity(ctx context.Context, req *anteilv1.MsgProvideLiquidity) (*anteilv1.MsgProvideLiquidityResponse, error) {
	// Simple stub implementation
	return &anteilv1.MsgProvideLiquidityResponse{
		Success:        true,
		SharesReceived: "1000",
		PoolId:         req.PoolId,
	}, nil
}

func (s MsgServer) WithdrawLiquidity(ctx context.Context, req *anteilv1.MsgWithdrawLiquidity) (*anteilv1.MsgWithdrawLiquidityResponse, error) {
	// Simple stub implementation
	return &anteilv1.MsgWithdrawLiquidityResponse{
		Success:           true,
		AntAmountReceived: "1000",
		PoolId:            req.PoolId,
	}, nil
}

func (s MsgServer) StakeANT(ctx context.Context, req *anteilv1.MsgStakeANT) (*anteilv1.MsgStakeANTResponse, error) {
	// Simple stub implementation
	return &anteilv1.MsgStakeANTResponse{
		Success:      true,
		StakedAmount: req.AntAmount,
		RewardRate:   "5.0",
	}, nil
}

func (s MsgServer) UnstakeANT(ctx context.Context, req *anteilv1.MsgUnstakeANT) (*anteilv1.MsgUnstakeANTResponse, error) {
	// Simple stub implementation
	return &anteilv1.MsgUnstakeANTResponse{
		Success:        true,
		UnstakedAmount: req.AntAmount,
		RewardsClaimed: "50",
	}, nil
}

func (s MsgServer) ClaimRewards(ctx context.Context, req *anteilv1.MsgClaimRewards) (*anteilv1.MsgClaimRewardsResponse, error) {
	// Simple stub implementation
	return &anteilv1.MsgClaimRewardsResponse{
		Success:            true,
		RewardAmount:       "100",
		TotalRewardsEarned: "1000",
	}, nil
}
