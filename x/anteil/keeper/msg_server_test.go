package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

type MsgServerTestSuite struct {
	suite.Suite

	cdc        codec.Codec
	ctx        sdk.Context
	keeper     *Keeper
	msgServer  anteilv1.MsgServer
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
}

func (suite *MsgServerTestSuite) SetupTest() {
	cfg := sdk.GetConfig()
	if cfg.GetBech32AccountAddrPrefix() != "cosmos" {
		cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
	}
	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)

	// Create store keys
	suite.storeKey = storetypes.NewKVStoreKey("test_anteil")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")

	// Create test context
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)

	// Create params keeper and subspace
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore.WithKeyTable(types.ParamKeyTable())

	// Create keeper and msg server
	suite.keeper = NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)
	suite.msgServer = NewMsgServer(suite.keeper)

	// Set default params
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func (suite *MsgServerTestSuite) TestPlaceOrder() {
	// Test valid order creation
	msg := &anteilv1.MsgPlaceOrder{
		Owner:        mustAddrInternal("0000000000000000000000000000000000000001"),
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		IdentityHash: "hash123",
	}

	resp, err := suite.msgServer.PlaceOrder(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.True(suite.T(), resp.Success)
	require.NotEmpty(suite.T(), resp.OrderId)

	// Verify order was created
	order, err := suite.keeper.GetOrder(suite.ctx, resp.OrderId)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), mustAddrInternal("0000000000000000000000000000000000000001"), order.Owner)
	require.Equal(suite.T(), anteilv1.OrderType_ORDER_TYPE_LIMIT, order.OrderType)

	// Test invalid order creation
	invalidMsg := &anteilv1.MsgPlaceOrder{
		Owner:        "",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		IdentityHash: "hash456",
	}

	_, err = suite.msgServer.PlaceOrder(suite.ctx, invalidMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyOwner, err)
}

func (suite *MsgServerTestSuite) TestCancelOrder() {
	// First create an order
	createMsg := &anteilv1.MsgPlaceOrder{
		Owner:        mustAddrInternal("0000000000000000000000000000000000000001"),
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		IdentityHash: "hash123",
	}

	createResp, err := suite.msgServer.PlaceOrder(suite.ctx, createMsg)
	require.NoError(suite.T(), err)

	// Test valid order cancellation
	msg := &anteilv1.MsgCancelOrder{
		Owner:   mustAddrInternal("0000000000000000000000000000000000000001"),
		OrderId: createResp.OrderId,
	}

	resp, err := suite.msgServer.CancelOrder(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify order is canceled
	order, err := suite.keeper.GetOrder(suite.ctx, createResp.OrderId)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.OrderStatus_ORDER_STATUS_CANCELLED, order.Status)

	// Test canceling non-existent order
	nonExistentMsg := &anteilv1.MsgCancelOrder{
		OrderId: "non_existent_id",
		Owner:   mustAddrInternal("0000000000000000000000000000000000000001"),
	}

	_, err = suite.msgServer.CancelOrder(suite.ctx, nonExistentMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *MsgServerTestSuite) TestPlaceBid() {
	// First create an auction using keeper directly
	auction := types.NewAuction(uint64(1000), "1000000", "1.0")
	err := suite.keeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Test placing bid
	msg := &anteilv1.MsgPlaceBid{
		AuctionId:    auction.AuctionId,
		Bidder:       mustAddrInternal("0000000000000000000000000000000000000001"),
		Amount:       "1000000",
		IdentityHash: "hash123",
	}

	resp, err := suite.msgServer.PlaceBid(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotEmpty(suite.T(), resp.BidId)

	// Note: In real implementation, we would verify the bid was placed
	// For now, we just verify the response is successful

	// Test placing bid on non-existent auction
	nonExistentMsg := &anteilv1.MsgPlaceBid{
		AuctionId:    "non_existent_auction",
		Bidder:       mustAddrInternal("0000000000000000000000000000000000000002"),
		Amount:       "1000000",
		IdentityHash: "hash456",
	}

	_, err = suite.msgServer.PlaceBid(suite.ctx, nonExistentMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAuctionNotFound, err)
}

func (suite *MsgServerTestSuite) TestSettleAuction() {
	// Create an auction using keeper directly
	auction := types.NewAuction(uint64(1000), "1000000", "1.0")
	err := suite.keeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Place bids using keeper directly
	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, mustAddrInternal("0000000000000000000000000000000000000001"), "1000000")
	require.NoError(suite.T(), err)

	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, mustAddrInternal("0000000000000000000000000000000000000002"), "1500000")
	require.NoError(suite.T(), err)

	// Verify winning bid was set
	updatedAuction, err := suite.keeper.GetAuction(suite.ctx, auction.AuctionId)
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), updatedAuction.WinningBid)

	// Close the auction first (required for settlement)
	updatedAuction.Status = anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED
	err = suite.keeper.UpdateAuction(suite.ctx, updatedAuction)
	require.NoError(suite.T(), err)

	// Test settling auction
	msg := &anteilv1.MsgSettleAuction{
		AuctionId: auction.AuctionId,
	}

	resp, err := suite.msgServer.SettleAuction(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify auction is settled
	finalAuction, err := suite.keeper.GetAuction(suite.ctx, auction.AuctionId)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.AuctionStatus_AUCTION_STATUS_SETTLED, finalAuction.Status)

	// Note: In real implementation, we would verify the winning bid and settlement details
}

// =============================================================================
// UpdateOrder Tests
// =============================================================================

func (suite *MsgServerTestSuite) TestUpdateOrder_Success() {
	createResp, err := suite.msgServer.PlaceOrder(suite.ctx, &anteilv1.MsgPlaceOrder{
		Owner:        mustAddrInternal("0000000000000000000000000000000000000001"),
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		IdentityHash: "hash123",
	})
	require.NoError(suite.T(), err)

	resp, err := suite.msgServer.UpdateOrder(suite.ctx, &anteilv1.MsgUpdateOrder{
		OrderId:   createResp.OrderId,
		Owner:     mustAddrInternal("0000000000000000000000000000000000000001"),
		NewAmount: "2000000",
		NewPrice:  "2.0",
	})
	require.NoError(suite.T(), err)
	require.True(suite.T(), resp.Success)

	order, err := suite.keeper.GetOrder(suite.ctx, createResp.OrderId)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "2000000", order.AntAmount)
	require.Equal(suite.T(), "2.0", order.Price)
}

func (suite *MsgServerTestSuite) TestUpdateOrder_NilRequest() {
	_, err := suite.msgServer.UpdateOrder(suite.ctx, nil)
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestUpdateOrder_OwnerMismatch() {
	createResp, err := suite.msgServer.PlaceOrder(suite.ctx, &anteilv1.MsgPlaceOrder{
		Owner:        mustAddrInternal("0000000000000000000000000000000000000001"),
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		IdentityHash: "hash",
	})
	require.NoError(suite.T(), err)

	_, err = suite.msgServer.UpdateOrder(suite.ctx, &anteilv1.MsgUpdateOrder{
		OrderId:   createResp.OrderId,
		Owner:     mustAddrInternal("0000000000000000000000000000000000000002"),
		NewAmount: "2000000",
	})
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "unauthorized")
}

func (suite *MsgServerTestSuite) TestUpdateOrder_EmptyFields() {
	_, err := suite.msgServer.UpdateOrder(suite.ctx, &anteilv1.MsgUpdateOrder{
		OrderId: "",
		Owner:   mustAddrInternal("0000000000000000000000000000000000000001"),
	})
	require.Error(suite.T(), err)

	_, err = suite.msgServer.UpdateOrder(suite.ctx, &anteilv1.MsgUpdateOrder{
		OrderId: "some-id",
		Owner:   "",
	})
	require.Error(suite.T(), err)
}

// =============================================================================
// RegisterMarketMaker Tests
// =============================================================================

func (suite *MsgServerTestSuite) TestRegisterMarketMaker_Success() {
	resp, err := suite.msgServer.RegisterMarketMaker(suite.ctx, &anteilv1.MsgRegisterMarketMaker{
		Address:      mustAddrInternal("0000000000000000000000000000000000000001"),
		AntBalance:   "10000000",
		BidAskSpread: "0.01",
		MinOrderSize: "1000",
		MaxOrderSize: "1000000",
	})
	require.NoError(suite.T(), err)
	require.True(suite.T(), resp.Success)
	require.NotEmpty(suite.T(), resp.MarketMakerId)
}

func (suite *MsgServerTestSuite) TestRegisterMarketMaker_Duplicate() {
	addr := mustAddrInternal("0000000000000000000000000000000000000001")
	_, err := suite.msgServer.RegisterMarketMaker(suite.ctx, &anteilv1.MsgRegisterMarketMaker{
		Address: addr,
	})
	require.NoError(suite.T(), err)

	_, err = suite.msgServer.RegisterMarketMaker(suite.ctx, &anteilv1.MsgRegisterMarketMaker{
		Address: addr,
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestRegisterMarketMaker_NilRequest() {
	_, err := suite.msgServer.RegisterMarketMaker(suite.ctx, nil)
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestRegisterMarketMaker_EmptyAddress() {
	_, err := suite.msgServer.RegisterMarketMaker(suite.ctx, &anteilv1.MsgRegisterMarketMaker{
		Address: "",
	})
	require.Error(suite.T(), err)
}

// =============================================================================
// StakeANT / UnstakeANT / ClaimRewards Tests
// =============================================================================

func (suite *MsgServerTestSuite) TestStakeANT_Success() {
	addr := mustAddrInternal("0000000000000000000000000000000000000001")
	position := types.NewUserPosition(addr, "10000000")
	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	resp, err := suite.msgServer.StakeANT(suite.ctx, &anteilv1.MsgStakeANT{
		Staker:    addr,
		AntAmount: "5000000",
	})
	require.NoError(suite.T(), err)
	require.True(suite.T(), resp.Success)
	require.Equal(suite.T(), "5000000.000000000000000000", resp.StakedAmount)

	pos, err := suite.keeper.GetUserPosition(suite.ctx, addr)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "5000000.000000000000000000", pos.AvailableAnt)
	require.Equal(suite.T(), "5000000.000000000000000000", pos.LockedAnt)
}

func (suite *MsgServerTestSuite) TestStakeANT_InsufficientBalance() {
	addr := mustAddrInternal("0000000000000000000000000000000000000001")
	position := types.NewUserPosition(addr, "1000000")
	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	_, err = suite.msgServer.StakeANT(suite.ctx, &anteilv1.MsgStakeANT{
		Staker:    addr,
		AntAmount: "5000000",
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestStakeANT_NilRequest() {
	_, err := suite.msgServer.StakeANT(suite.ctx, nil)
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestStakeANT_EmptyFields() {
	_, err := suite.msgServer.StakeANT(suite.ctx, &anteilv1.MsgStakeANT{
		Staker:    "",
		AntAmount: "1000",
	})
	require.Error(suite.T(), err)

	_, err = suite.msgServer.StakeANT(suite.ctx, &anteilv1.MsgStakeANT{
		Staker:    mustAddrInternal("0000000000000000000000000000000000000001"),
		AntAmount: "",
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestUnstakeANT_Success() {
	addr := mustAddrInternal("0000000000000000000000000000000000000001")
	position := types.NewUserPosition(addr, "10000000")
	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	// Stake first
	_, err = suite.msgServer.StakeANT(suite.ctx, &anteilv1.MsgStakeANT{
		Staker:    addr,
		AntAmount: "5000000",
	})
	require.NoError(suite.T(), err)

	// Unstake half
	resp, err := suite.msgServer.UnstakeANT(suite.ctx, &anteilv1.MsgUnstakeANT{
		Staker:    addr,
		AntAmount: "2500000",
	})
	require.NoError(suite.T(), err)
	require.True(suite.T(), resp.Success)

	pos, err := suite.keeper.GetUserPosition(suite.ctx, addr)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "7500000.000000000000000000", pos.AvailableAnt)
	require.Equal(suite.T(), "2500000.000000000000000000", pos.LockedAnt)
}

func (suite *MsgServerTestSuite) TestUnstakeANT_InsufficientStake() {
	addr := mustAddrInternal("0000000000000000000000000000000000000001")
	position := types.NewUserPosition(addr, "10000000")
	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	_, err = suite.msgServer.StakeANT(suite.ctx, &anteilv1.MsgStakeANT{
		Staker:    addr,
		AntAmount: "1000000",
	})
	require.NoError(suite.T(), err)

	_, err = suite.msgServer.UnstakeANT(suite.ctx, &anteilv1.MsgUnstakeANT{
		Staker:    addr,
		AntAmount: "5000000",
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestUnstakeANT_NoStakingPosition() {
	_, err := suite.msgServer.UnstakeANT(suite.ctx, &anteilv1.MsgUnstakeANT{
		Staker:    mustAddrInternal("0000000000000000000000000000000000000099"),
		AntAmount: "1000000",
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestUnstakeANT_NilRequest() {
	_, err := suite.msgServer.UnstakeANT(suite.ctx, nil)
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestClaimRewards_Success() {
	addr := mustAddrInternal("0000000000000000000000000000000000000001")
	position := types.NewUserPosition(addr, "10000000")
	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	_, err = suite.msgServer.StakeANT(suite.ctx, &anteilv1.MsgStakeANT{
		Staker:    addr,
		AntAmount: "5000000",
	})
	require.NoError(suite.T(), err)

	resp, err := suite.msgServer.ClaimRewards(suite.ctx, &anteilv1.MsgClaimRewards{
		Staker: addr,
	})
	require.NoError(suite.T(), err)
	require.True(suite.T(), resp.Success)
	require.NotEmpty(suite.T(), resp.RewardAmount)
}

func (suite *MsgServerTestSuite) TestClaimRewards_NoStake() {
	_, err := suite.msgServer.ClaimRewards(suite.ctx, &anteilv1.MsgClaimRewards{
		Staker: mustAddrInternal("0000000000000000000000000000000000000099"),
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestClaimRewards_NilRequest() {
	_, err := suite.msgServer.ClaimRewards(suite.ctx, nil)
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestClaimRewards_EmptyStaker() {
	_, err := suite.msgServer.ClaimRewards(suite.ctx, &anteilv1.MsgClaimRewards{
		Staker: "",
	})
	require.Error(suite.T(), err)
}

// =============================================================================
// ProvideLiquidity / WithdrawLiquidity Tests
// =============================================================================

func (suite *MsgServerTestSuite) TestProvideLiquidity_NilRequest() {
	_, err := suite.msgServer.ProvideLiquidity(suite.ctx, nil)
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestProvideLiquidity_EmptyProvider() {
	_, err := suite.msgServer.ProvideLiquidity(suite.ctx, &anteilv1.MsgProvideLiquidity{
		Provider:  "",
		AntAmount: "1000",
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestProvideLiquidity_EmptyAmount() {
	_, err := suite.msgServer.ProvideLiquidity(suite.ctx, &anteilv1.MsgProvideLiquidity{
		Provider:  mustAddrInternal("0000000000000000000000000000000000000001"),
		AntAmount: "",
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestWithdrawLiquidity_NilRequest() {
	_, err := suite.msgServer.WithdrawLiquidity(suite.ctx, nil)
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestWithdrawLiquidity_EmptyProvider() {
	_, err := suite.msgServer.WithdrawLiquidity(suite.ctx, &anteilv1.MsgWithdrawLiquidity{
		Provider: "",
		Shares:   "1000",
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestWithdrawLiquidity_EmptyShares() {
	_, err := suite.msgServer.WithdrawLiquidity(suite.ctx, &anteilv1.MsgWithdrawLiquidity{
		Provider: mustAddrInternal("0000000000000000000000000000000000000001"),
		Shares:   "",
	})
	require.Error(suite.T(), err)
}

func TestMsgServerTestSuite(t *testing.T) {
	suite.Run(t, new(MsgServerTestSuite))
}
