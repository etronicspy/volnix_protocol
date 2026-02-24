package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/types/known/timestamppb"

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

func (s MsgServer) PlaceBid(ctx context.Context, req *anteilv1.MsgPlaceBid) (*anteilv1.MsgPlaceBidResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

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

	err := s.k.PlaceBid(sdkCtx, req.AuctionId, req.Bidder, req.Amount)
	if err != nil {
		return nil, err
	}

	bidId := fmt.Sprintf("bid-%s-%s", req.AuctionId, req.Bidder)

	return &anteilv1.MsgPlaceBidResponse{
		Success: true,
		BidId:   bidId,
		Status:  "bid placed successfully",
	}, nil
}

func (s MsgServer) SettleAuction(ctx context.Context, req *anteilv1.MsgSettleAuction) (*anteilv1.MsgSettleAuctionResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.AuctionId == "" {
		return nil, types.ErrEmptyAuctionID
	}

	auction, err := s.k.GetAuction(sdkCtx, req.AuctionId)
	if err != nil {
		return nil, err
	}

	if auction.Status != anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED {
		return nil, types.ErrAuctionNotClosed
	}

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
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Address == "" {
		return nil, types.ErrEmptyOwner
	}

	store := sdkCtx.KVStore(s.k.storeKey)
	makerID := fmt.Sprintf("mm_%s", req.Address)
	makerKey := types.GetMarketMakerKey(makerID)

	if store.Has(makerKey) {
		return nil, types.ErrMarketMakerAlreadyExists
	}

	mm := &anteilv1.MarketMaker{
		Address:      req.Address,
		AntBalance:   req.AntBalance,
		BidAskSpread: req.BidAskSpread,
		MinOrderSize: req.MinOrderSize,
		MaxOrderSize: req.MaxOrderSize,
		IsActive:     true,
		LastActivity: timestamppb.Now(),
	}

	mmBz, err := s.k.cdc.Marshal(mm)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal market maker: %w", err)
	}
	store.Set(makerKey, mmBz)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"anteil.market_maker_registered",
			sdk.NewAttribute("market_maker_id", makerID),
			sdk.NewAttribute("owner", req.Address),
		),
	)

	return &anteilv1.MsgRegisterMarketMakerResponse{
		Success:       true,
		MarketMakerId: makerID,
	}, nil
}

func (s MsgServer) ProvideLiquidity(ctx context.Context, req *anteilv1.MsgProvideLiquidity) (*anteilv1.MsgProvideLiquidityResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Provider == "" {
		return nil, types.ErrEmptyOwner
	}
	if req.AntAmount == "" {
		return nil, types.ErrEmptyAntAmount
	}

	depositAmount, err := math.LegacyNewDecFromStr(req.AntAmount)
	if err != nil || !depositAmount.IsPositive() {
		return nil, fmt.Errorf("invalid deposit amount: %s", req.AntAmount)
	}

	position, err := s.k.GetUserPosition(sdkCtx, req.Provider)
	if err != nil {
		return nil, types.ErrPositionNotFound
	}

	availableAnt := mustParseDec(position.AvailableAnt)
	if availableAnt.LT(depositAmount) {
		return nil, types.ErrInsufficientBalance
	}

	sharesReceived := depositAmount

	position.AvailableAnt = availableAnt.Sub(depositAmount).String()
	lockedAnt := mustParseDec(position.LockedAnt)
	position.LockedAnt = lockedAnt.Add(depositAmount).String()
	position.LastActivity = timestamppb.Now()

	if err := s.k.SetUserPosition(sdkCtx, position); err != nil {
		return nil, fmt.Errorf("failed to update user position: %w", err)
	}

	poolID := req.PoolId
	if poolID == "" {
		poolID = "default_pool"
	}
	store := sdkCtx.KVStore(s.k.storeKey)
	shareKey := types.GetLiquidityShareKey(poolID, req.Provider)

	existingShares := math.LegacyZeroDec()
	if bz := store.Get(shareKey); bz != nil {
		existingShares = mustParseDec(string(bz))
	}
	newShares := existingShares.Add(sharesReceived)
	store.Set(shareKey, []byte(newShares.String()))

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"anteil.liquidity_provided",
			sdk.NewAttribute("provider", req.Provider),
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("amount", req.AntAmount),
			sdk.NewAttribute("shares_received", sharesReceived.String()),
		),
	)

	return &anteilv1.MsgProvideLiquidityResponse{
		Success:        true,
		SharesReceived: sharesReceived.String(),
		PoolId:         poolID,
	}, nil
}

func (s MsgServer) WithdrawLiquidity(ctx context.Context, req *anteilv1.MsgWithdrawLiquidity) (*anteilv1.MsgWithdrawLiquidityResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Provider == "" {
		return nil, types.ErrEmptyOwner
	}
	if req.Shares == "" {
		return nil, fmt.Errorf("shares amount cannot be empty")
	}

	sharesToWithdraw, err := math.LegacyNewDecFromStr(req.Shares)
	if err != nil || !sharesToWithdraw.IsPositive() {
		return nil, fmt.Errorf("invalid shares amount: %s", req.Shares)
	}

	poolID := req.PoolId
	if poolID == "" {
		poolID = "default_pool"
	}

	store := sdkCtx.KVStore(s.k.storeKey)
	shareKey := types.GetLiquidityShareKey(poolID, req.Provider)
	bz := store.Get(shareKey)
	if bz == nil {
		return nil, types.ErrInsufficientShares
	}

	currentShares := mustParseDec(string(bz))
	if currentShares.LT(sharesToWithdraw) {
		return nil, types.ErrInsufficientShares
	}

	antReturned := sharesToWithdraw

	remainingShares := currentShares.Sub(sharesToWithdraw)
	if remainingShares.IsZero() {
		store.Delete(shareKey)
	} else {
		store.Set(shareKey, []byte(remainingShares.String()))
	}

	position, err := s.k.GetUserPosition(sdkCtx, req.Provider)
	if err != nil {
		return nil, types.ErrPositionNotFound
	}

	lockedAnt := mustParseDec(position.LockedAnt)
	availableAnt := mustParseDec(position.AvailableAnt)
	position.LockedAnt = lockedAnt.Sub(antReturned).String()
	position.AvailableAnt = availableAnt.Add(antReturned).String()
	position.LastActivity = timestamppb.Now()

	if err := s.k.SetUserPosition(sdkCtx, position); err != nil {
		return nil, fmt.Errorf("failed to update user position: %w", err)
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"anteil.liquidity_withdrawn",
			sdk.NewAttribute("provider", req.Provider),
			sdk.NewAttribute("pool_id", poolID),
			sdk.NewAttribute("shares_withdrawn", sharesToWithdraw.String()),
			sdk.NewAttribute("ant_returned", antReturned.String()),
		),
	)

	return &anteilv1.MsgWithdrawLiquidityResponse{
		Success:           true,
		AntAmountReceived: antReturned.String(),
		PoolId:            poolID,
	}, nil
}

func (s MsgServer) StakeANT(ctx context.Context, req *anteilv1.MsgStakeANT) (*anteilv1.MsgStakeANTResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Staker == "" {
		return nil, types.ErrEmptyOwner
	}
	if req.AntAmount == "" {
		return nil, types.ErrEmptyAntAmount
	}

	stakeAmount, err := math.LegacyNewDecFromStr(req.AntAmount)
	if err != nil || !stakeAmount.IsPositive() {
		return nil, fmt.Errorf("invalid stake amount: %s", req.AntAmount)
	}

	position, err := s.k.GetUserPosition(sdkCtx, req.Staker)
	if err != nil {
		return nil, types.ErrPositionNotFound
	}

	availableAnt := mustParseDec(position.AvailableAnt)
	if availableAnt.LT(stakeAmount) {
		return nil, types.ErrInsufficientBalance
	}

	position.AvailableAnt = availableAnt.Sub(stakeAmount).String()
	lockedAnt := mustParseDec(position.LockedAnt)
	position.LockedAnt = lockedAnt.Add(stakeAmount).String()
	position.LastActivity = timestamppb.Now()

	if err := s.k.SetUserPosition(sdkCtx, position); err != nil {
		return nil, fmt.Errorf("failed to update user position: %w", err)
	}

	store := sdkCtx.KVStore(s.k.storeKey)
	stakingKey := types.GetStakingPositionKey(req.Staker)

	existingStake := math.LegacyZeroDec()
	if bz := store.Get(stakingKey); bz != nil {
		existingStake = mustParseDec(string(bz))
	}
	newStake := existingStake.Add(stakeAmount)
	store.Set(stakingKey, []byte(newStake.String()))

	params := s.k.GetParams(sdkCtx)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"anteil.ant_staked",
			sdk.NewAttribute("staker", req.Staker),
			sdk.NewAttribute("amount", stakeAmount.String()),
			sdk.NewAttribute("total_staked", newStake.String()),
			sdk.NewAttribute("block_height", strconv.FormatInt(sdkCtx.BlockHeight(), 10)),
		),
	)

	return &anteilv1.MsgStakeANTResponse{
		Success:      true,
		StakedAmount: newStake.String(),
		RewardRate:   params.StakingRewardRate,
	}, nil
}

func (s MsgServer) UnstakeANT(ctx context.Context, req *anteilv1.MsgUnstakeANT) (*anteilv1.MsgUnstakeANTResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Staker == "" {
		return nil, types.ErrEmptyOwner
	}
	if req.AntAmount == "" {
		return nil, types.ErrEmptyAntAmount
	}

	unstakeAmount, err := math.LegacyNewDecFromStr(req.AntAmount)
	if err != nil || !unstakeAmount.IsPositive() {
		return nil, fmt.Errorf("invalid unstake amount: %s", req.AntAmount)
	}

	store := sdkCtx.KVStore(s.k.storeKey)
	stakingKey := types.GetStakingPositionKey(req.Staker)

	bz := store.Get(stakingKey)
	if bz == nil {
		return nil, types.ErrStakingPositionNotFound
	}

	currentStake := mustParseDec(string(bz))
	if currentStake.LT(unstakeAmount) {
		return nil, types.ErrInsufficientStake
	}

	remainingStake := currentStake.Sub(unstakeAmount)
	if remainingStake.IsZero() {
		store.Delete(stakingKey)
	} else {
		store.Set(stakingKey, []byte(remainingStake.String()))
	}

	position, err := s.k.GetUserPosition(sdkCtx, req.Staker)
	if err != nil {
		return nil, types.ErrPositionNotFound
	}

	lockedAnt := mustParseDec(position.LockedAnt)
	availableAnt := mustParseDec(position.AvailableAnt)
	position.LockedAnt = lockedAnt.Sub(unstakeAmount).String()
	position.AvailableAnt = availableAnt.Add(unstakeAmount).String()
	position.LastActivity = timestamppb.Now()

	if err := s.k.SetUserPosition(sdkCtx, position); err != nil {
		return nil, fmt.Errorf("failed to update user position: %w", err)
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"anteil.ant_unstaked",
			sdk.NewAttribute("staker", req.Staker),
			sdk.NewAttribute("amount", unstakeAmount.String()),
			sdk.NewAttribute("remaining_staked", remainingStake.String()),
		),
	)

	return &anteilv1.MsgUnstakeANTResponse{
		Success:        true,
		UnstakedAmount: unstakeAmount.String(),
		RewardsClaimed: "0",
	}, nil
}

func (s MsgServer) ClaimRewards(ctx context.Context, req *anteilv1.MsgClaimRewards) (*anteilv1.MsgClaimRewardsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Staker == "" {
		return nil, types.ErrEmptyOwner
	}

	store := sdkCtx.KVStore(s.k.storeKey)
	stakingKey := types.GetStakingPositionKey(req.Staker)

	bz := store.Get(stakingKey)
	if bz == nil {
		return nil, types.ErrStakingPositionNotFound
	}

	currentStake := mustParseDec(string(bz))
	if currentStake.IsZero() {
		return nil, types.ErrNoRewardsToClaim
	}

	params := s.k.GetParams(sdkCtx)
	rewardRate := mustParseDec(params.StakingRewardRate)

	rewardAmount := currentStake.Mul(rewardRate)

	if rewardAmount.IsZero() {
		return nil, types.ErrNoRewardsToClaim
	}

	position, err := s.k.GetUserPosition(sdkCtx, req.Staker)
	if err != nil {
		return nil, types.ErrPositionNotFound
	}

	availableAnt := mustParseDec(position.AvailableAnt)
	balance := mustParseDec(position.AntBalance)
	position.AvailableAnt = availableAnt.Add(rewardAmount).String()
	position.AntBalance = balance.Add(rewardAmount).String()
	position.LastActivity = timestamppb.Now()

	if err := s.k.SetUserPosition(sdkCtx, position); err != nil {
		return nil, fmt.Errorf("failed to update user position: %w", err)
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"anteil.rewards_claimed",
			sdk.NewAttribute("staker", req.Staker),
			sdk.NewAttribute("reward_amount", rewardAmount.String()),
			sdk.NewAttribute("staked_amount", currentStake.String()),
			sdk.NewAttribute("reward_rate", params.StakingRewardRate),
		),
	)

	return &anteilv1.MsgClaimRewardsResponse{
		Success:            true,
		RewardAmount:       rewardAmount.String(),
		TotalRewardsEarned: rewardAmount.String(),
	}, nil
}
