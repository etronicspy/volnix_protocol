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
		Owner:        "cosmos1test",
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
	require.Equal(suite.T(), "cosmos1test", order.Owner)
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
		Owner:        "cosmos1test",
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
		Owner:   "cosmos1test",
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
		Owner:   "cosmos1test",
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
		Bidder:       "cosmos1test",
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
		Bidder:       "cosmos2test",
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
	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, "cosmos1test", "1000000")
	require.NoError(suite.T(), err)

	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, "cosmos2test", "1500000")
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

func TestMsgServerTestSuite(t *testing.T) {
	suite.Run(t, new(MsgServerTestSuite))
}
