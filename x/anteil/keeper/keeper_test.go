package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx        sdk.Context
	keeper     *keeper.Keeper
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
}

func (suite *KeeperTestSuite) SetupTest() {
	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)

	// Create store keys
	suite.storeKey = storetypes.NewKVStoreKey(types.StoreKey)
	tKey := storetypes.NewTransientStoreKey("transient_test")

	// Create test context
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)

	// Create params keeper and subspace
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore = suite.paramStore.WithKeyTable(types.ParamKeyTable())

	// Create keeper
	suite.keeper = keeper.NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)

	// Set default params
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// Test Order Management
func (suite *KeeperTestSuite) TestSetOrder() {
	order := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        "cosmos1test",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash123",
	}

	err := suite.keeper.SetOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Verify order was stored
	retrieved, err := suite.keeper.GetOrder(suite.ctx, "order1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), order.OrderId, retrieved.OrderId)
	require.Equal(suite.T(), order.Owner, retrieved.Owner)
}

func (suite *KeeperTestSuite) TestSetOrder_Duplicate() {
	order := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        "cosmos1test",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash123",
	}

	err := suite.keeper.SetOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Try to set duplicate
	err = suite.keeper.SetOrder(suite.ctx, order)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderAlreadyExists, err)
}

func (suite *KeeperTestSuite) TestGetOrder_NotFound() {
	_, err := suite.keeper.GetOrder(suite.ctx, "notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *KeeperTestSuite) TestUpdateOrder() {
	order := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        "cosmos1test",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash123",
	}

	err := suite.keeper.SetOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Update order
	order.Status = anteilv1.OrderStatus_ORDER_STATUS_FILLED
	err = suite.keeper.UpdateOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Verify update
	retrieved, err := suite.keeper.GetOrder(suite.ctx, "order1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.OrderStatus_ORDER_STATUS_FILLED, retrieved.Status)
}

func (suite *KeeperTestSuite) TestCancelOrder() {
	order := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        "cosmos1test",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash123",
	}

	err := suite.keeper.SetOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Cancel order
	err = suite.keeper.CancelOrder(suite.ctx, "order1")
	require.NoError(suite.T(), err)

	// Verify cancellation
	retrieved, err := suite.keeper.GetOrder(suite.ctx, "order1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.OrderStatus_ORDER_STATUS_CANCELLED, retrieved.Status)
}

func (suite *KeeperTestSuite) TestDeleteOrder() {
	order := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        "cosmos1test",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash123",
	}

	err := suite.keeper.SetOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Delete order
	err = suite.keeper.DeleteOrder(suite.ctx, "order1")
	require.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.keeper.GetOrder(suite.ctx, "order1")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *KeeperTestSuite) TestGetAllOrders() {
	// Create multiple orders
	for i := range 5 {
		order := &anteilv1.Order{
			OrderId:      "order" + string(rune(i)),
			Owner:        "cosmos1test",
			OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
			OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
			AntAmount:    "1000000",
			Price:        "1.5",
			Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
			CreatedAt:    timestamppb.Now(),
			IdentityHash: "hash" + string(rune(i)),
		}
		err := suite.keeper.SetOrder(suite.ctx, order)
		require.NoError(suite.T(), err)
	}

	orders, err := suite.keeper.GetAllOrders(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders, 5)
}

func (suite *KeeperTestSuite) TestGetOrdersByOwner() {
	// Create orders for different owners
	for i := range 3 {
		order := &anteilv1.Order{
			OrderId:      "order_owner1_" + string(rune(i)),
			Owner:        "cosmos1owner1",
			OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
			OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
			AntAmount:    "1000000",
			Price:        "1.5",
			Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
			CreatedAt:    timestamppb.Now(),
			IdentityHash: "hash" + string(rune(i)),
		}
		err := suite.keeper.SetOrder(suite.ctx, order)
		require.NoError(suite.T(), err)
	}

	for i := range 2 {
		order := &anteilv1.Order{
			OrderId:      "order_owner2_" + string(rune(i)),
			Owner:        "cosmos1owner2",
			OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
			OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
			AntAmount:    "1000000",
			Price:        "1.5",
			Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
			CreatedAt:    timestamppb.Now(),
			IdentityHash: "hash_owner2_" + string(rune(i)),
		}
		err := suite.keeper.SetOrder(suite.ctx, order)
		require.NoError(suite.T(), err)
	}

	// Get orders for owner1
	orders, err := suite.keeper.GetOrdersByOwner(suite.ctx, "cosmos1owner1")
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders, 3)

	// Get orders for owner2
	orders, err = suite.keeper.GetOrdersByOwner(suite.ctx, "cosmos1owner2")
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders, 2)
}

// Test Trade Management
func (suite *KeeperTestSuite) TestExecuteTrade() {
	buyOrder := &anteilv1.Order{
		OrderId:      "buy1",
		Owner:        "cosmos1buyer",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_buy",
	}

	sellOrder := &anteilv1.Order{
		OrderId:      "sell1",
		Owner:        "cosmos1seller",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_SELL,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_sell",
	}

	err := suite.keeper.SetOrder(suite.ctx, buyOrder)
	require.NoError(suite.T(), err)

	err = suite.keeper.SetOrder(suite.ctx, sellOrder)
	require.NoError(suite.T(), err)

	// Execute trade
	err = suite.keeper.ExecuteTrade(suite.ctx, "buy1", "sell1")
	require.NoError(suite.T(), err)

	// Verify orders are filled
	buyRetrieved, err := suite.keeper.GetOrder(suite.ctx, "buy1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.OrderStatus_ORDER_STATUS_FILLED, buyRetrieved.Status)

	sellRetrieved, err := suite.keeper.GetOrder(suite.ctx, "sell1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.OrderStatus_ORDER_STATUS_FILLED, sellRetrieved.Status)
}

func (suite *KeeperTestSuite) TestExecuteTrade_InvalidOrderType() {
	// Both orders are buy orders
	buyOrder1 := &anteilv1.Order{
		OrderId:      "buy1",
		Owner:        "cosmos1buyer1",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_buy1",
	}

	buyOrder2 := &anteilv1.Order{
		OrderId:      "buy2",
		Owner:        "cosmos1buyer2",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_buy2",
	}

	err := suite.keeper.SetOrder(suite.ctx, buyOrder1)
	require.NoError(suite.T(), err)

	err = suite.keeper.SetOrder(suite.ctx, buyOrder2)
	require.NoError(suite.T(), err)

	// Try to execute trade with two buy orders
	err = suite.keeper.ExecuteTrade(suite.ctx, "buy1", "buy2")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrInvalidOrderType, err)
}

func (suite *KeeperTestSuite) TestSetTrade() {
	trade := &anteilv1.Trade{
		TradeId:     "trade1",
		BuyOrderId:  "buy1",
		SellOrderId: "sell1",
		Buyer:       "cosmos1buyer",
		Seller:      "cosmos1seller",
		Price:       "1.5",
		AntAmount:   "1000000",
		ExecutedAt:  timestamppb.Now(),
	}

	err := suite.keeper.SetTrade(suite.ctx, trade)
	require.NoError(suite.T(), err)

	// Verify trade was stored
	retrieved, err := suite.keeper.GetTrade(suite.ctx, "trade1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), trade.TradeId, retrieved.TradeId)
}

func (suite *KeeperTestSuite) TestGetAllTrades() {
	// Create multiple trades
	for i := range 5 {
		trade := &anteilv1.Trade{
			TradeId:     "trade" + string(rune(i)),
			BuyOrderId:  "buy" + string(rune(i)),
			SellOrderId: "sell" + string(rune(i)),
			Buyer:       "cosmos1buyer" + string(rune(i)),
			Seller:      "cosmos1seller" + string(rune(i)),
			Price:       "1.5",
			AntAmount:   "1000000",
			ExecutedAt:  timestamppb.Now(),
		}
		err := suite.keeper.SetTrade(suite.ctx, trade)
		require.NoError(suite.T(), err)
	}

	trades, err := suite.keeper.GetAllTrades(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), trades, 5)
}

// Test Auction Management
func (suite *KeeperTestSuite) TestSetAuction() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Verify auction was stored
	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), auction.AuctionId, retrieved.AuctionId)
}

func (suite *KeeperTestSuite) TestSetAuction_Duplicate() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Try to set duplicate
	err = suite.keeper.SetAuction(suite.ctx, auction)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAuctionAlreadyExists, err)
}

func (suite *KeeperTestSuite) TestUpdateAuction() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Update auction
	auction.Status = anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED
	err = suite.keeper.UpdateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Verify update
	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED, retrieved.Status)
}

func (suite *KeeperTestSuite) TestGetAllAuctions() {
	// Create multiple auctions
	for i := range 5 {
		auction := &anteilv1.Auction{
			AuctionId:    "auction" + string(rune(i)),
			BlockHeight:  uint64(1000 + i),
			ReservePrice: "1000000",
			AntAmount:    "1000000",
			StartTime:    timestamppb.Now(),
			EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
			Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
			WinningBid:   "",
		}
		err := suite.keeper.SetAuction(suite.ctx, auction)
		require.NoError(suite.T(), err)
	}

	auctions, err := suite.keeper.GetAllAuctions(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), auctions, 5)
}

func (suite *KeeperTestSuite) TestPlaceBid() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Place bid
	err = suite.keeper.PlaceBid(suite.ctx, "auction1", "cosmos1bidder", "1500000")
	require.NoError(suite.T(), err)

	// Verify bid was placed
	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), retrieved.WinningBid)
}

func (suite *KeeperTestSuite) TestPlaceBid_AuctionClosed() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Try to place bid on closed auction
	err = suite.keeper.PlaceBid(suite.ctx, "auction1", "cosmos1bidder", "1500000")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAuctionClosed, err)
}

func (suite *KeeperTestSuite) TestSettleAuction() {
	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.New(currentTime.Add(-1 * time.Hour)),
		EndTime:      timestamppb.New(currentTime.Add(1 * time.Hour)), // Future end time for bidding
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Place a bid
	err = suite.keeper.PlaceBid(suite.ctx, "auction1", "cosmos1bidder", "1500000")
	require.NoError(suite.T(), err)

	// Get updated auction with winning bid
	updatedAuction, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)

	// Close the auction
	updatedAuction.Status = anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED
	err = suite.keeper.UpdateAuction(suite.ctx, updatedAuction)
	require.NoError(suite.T(), err)

	// Settle auction
	err = suite.keeper.SettleAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)

	// Verify auction is settled
	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.AuctionStatus_AUCTION_STATUS_SETTLED, retrieved.Status)
}

func (suite *KeeperTestSuite) TestSettleAuction_NotClosed() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Try to settle open auction
	err = suite.keeper.SettleAuction(suite.ctx, "auction1")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAuctionNotClosed, err)
}

// Test User Position Management
func (suite *KeeperTestSuite) TestSetUserPosition() {
	position := &anteilv1.UserPosition{
		Owner:        "cosmos1test",
		AntBalance:   "10000000",
		TotalTrades:  "5",
		TotalVolume:  "5000000",
		LastActivity: timestamppb.Now(),
	}

	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	// Verify position was stored
	retrieved, err := suite.keeper.GetUserPosition(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), position.Owner, retrieved.Owner)
	require.Equal(suite.T(), position.AntBalance, retrieved.AntBalance)
}

func (suite *KeeperTestSuite) TestUpdateUserPosition() {
	position := &anteilv1.UserPosition{
		Owner:        "cosmos1test",
		AntBalance:   "10000000",
		TotalTrades:  "5",
		TotalVolume:  "5000000",
		LastActivity: timestamppb.Now(),
	}

	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	// Update position
	err = suite.keeper.UpdateUserPosition(suite.ctx, "cosmos1test", "500000", 1)
	require.NoError(suite.T(), err)

	// Verify update
	retrieved, err := suite.keeper.GetUserPosition(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "500000", retrieved.AntBalance)
}

// Test ProcessAuctions
func (suite *KeeperTestSuite) TestProcessAuctions() {
	// Set block time to current time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Create open auction with past end time
	pastTime := currentTime.Add(-1 * time.Hour)
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.New(pastTime.Add(-24 * time.Hour)),
		EndTime:      timestamppb.New(pastTime),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Process auctions
	err = suite.keeper.ProcessAuctions(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify auction was closed
	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED, retrieved.Status)
}

// Test BeginBlocker
func (suite *KeeperTestSuite) TestBeginBlocker() {
	// Set block time to current time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Create open auction with past end time
	pastTime := currentTime.Add(-1 * time.Hour)
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.New(pastTime.Add(-24 * time.Hour)),
		EndTime:      timestamppb.New(pastTime),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Run BeginBlocker
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify auction was processed
	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED, retrieved.Status)
}

// Test Params
func (suite *KeeperTestSuite) TestGetSetParams() {
	params := types.DefaultParams()
	params.MinAntAmount = "2000000"

	suite.keeper.SetParams(suite.ctx, params)

	retrieved := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), params.MinAntAmount, retrieved.MinAntAmount)
}

// Additional tests for uncovered methods

func (suite *KeeperTestSuite) TestCreateOrder() {
	order := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        "cosmos1test",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash123",
	}

	err := suite.keeper.CreateOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Verify order was created
	retrieved, err := suite.keeper.GetOrder(suite.ctx, "order1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), order.OrderId, retrieved.OrderId)
}

func (suite *KeeperTestSuite) TestCreateAuction() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Verify auction was created
	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), auction.AuctionId, retrieved.AuctionId)
}

func (suite *KeeperTestSuite) TestGetAuction_NotFound() {
	_, err := suite.keeper.GetAuction(suite.ctx, "nonexistent")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAuctionNotFound, err)
}

func (suite *KeeperTestSuite) TestGetTrade_NotFound() {
	_, err := suite.keeper.GetTrade(suite.ctx, "nonexistent")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrTradeNotFound, err)
}

func (suite *KeeperTestSuite) TestGetUserPosition_NotFound() {
	_, err := suite.keeper.GetUserPosition(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrPositionNotFound, err)
}

func (suite *KeeperTestSuite) TestGetBid() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Place a bid
	err = suite.keeper.PlaceBid(suite.ctx, "auction1", "cosmos1bidder", "1500000")
	require.NoError(suite.T(), err)

	// Get the bid
	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), retrieved.WinningBid)

	bid, err := suite.keeper.GetBid(suite.ctx, "auction1", retrieved.WinningBid)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), bid)
	require.Equal(suite.T(), "cosmos1bidder", bid.Bidder)
}

func (suite *KeeperTestSuite) TestGetBid_NotFound() {
	_, err := suite.keeper.GetBid(suite.ctx, "auction1", "nonexistent")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrBidNotFound, err)
}

func (suite *KeeperTestSuite) TestPlaceBid_AuctionExpired() {
	// Set block time to current time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Create auction with past end time
	pastTime := currentTime.Add(-1 * time.Hour)
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.New(pastTime.Add(-24 * time.Hour)),
		EndTime:      timestamppb.New(pastTime),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Try to place bid on expired auction
	err = suite.keeper.PlaceBid(suite.ctx, "auction1", "cosmos1bidder", "1500000")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAuctionExpired, err)
}

func (suite *KeeperTestSuite) TestPlaceBid_HigherBid() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Place first bid
	err = suite.keeper.PlaceBid(suite.ctx, "auction1", "cosmos1bidder1", "1500000")
	require.NoError(suite.T(), err)

	// Place higher bid
	err = suite.keeper.PlaceBid(suite.ctx, "auction1", "cosmos1bidder2", "2000000")
	require.NoError(suite.T(), err)

	// Verify higher bid is winning
	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)

	winningBid, err := suite.keeper.GetBid(suite.ctx, "auction1", retrieved.WinningBid)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "cosmos1bidder2", winningBid.Bidder)
}

func (suite *KeeperTestSuite) TestSettleAuction_NoWinningBid() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(-1 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Try to settle auction without winning bid
	err = suite.keeper.SettleAuction(suite.ctx, "auction1")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrNoWinningBid, err)
}

func (suite *KeeperTestSuite) TestGetBidsByAuction() {
	bids, err := suite.keeper.GetBidsByAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), bids)
}

func (suite *KeeperTestSuite) TestEndBlocker() {
	err := suite.keeper.EndBlocker(suite.ctx)
	require.NoError(suite.T(), err)
}

// Additional tests for better coverage

func (suite *KeeperTestSuite) TestGetOrdersByOwner_MultipleOwners() {
	// Create orders for owner1
	for i := 0; i < 3; i++ {
		order := &anteilv1.Order{
			OrderId:      fmt.Sprintf("order_owner1_%d", i),
			Owner:        "cosmos1owner1",
			OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
			OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
			AntAmount:    "1000000",
			Price:        "1.5",
			Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
			CreatedAt:    timestamppb.Now(),
			IdentityHash: fmt.Sprintf("hash1_%d", i),
		}
		err := suite.keeper.SetOrder(suite.ctx, order)
		require.NoError(suite.T(), err)
	}

	// Create orders for owner2
	for i := 0; i < 2; i++ {
		order := &anteilv1.Order{
			OrderId:      fmt.Sprintf("order_owner2_%d", i),
			Owner:        "cosmos1owner2",
			OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
			OrderSide:    anteilv1.OrderSide_ORDER_SIDE_SELL,
			AntAmount:    "1000000",
			Price:        "1.5",
			Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
			CreatedAt:    timestamppb.Now(),
			IdentityHash: fmt.Sprintf("hash2_%d", i),
		}
		err := suite.keeper.SetOrder(suite.ctx, order)
		require.NoError(suite.T(), err)
	}

	// Get orders for owner1
	orders1, err := suite.keeper.GetOrdersByOwner(suite.ctx, "cosmos1owner1")
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders1, 3)

	// Get orders for owner2
	orders2, err := suite.keeper.GetOrdersByOwner(suite.ctx, "cosmos1owner2")
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders2, 2)

	// Get orders for non-existent owner
	orders3, err := suite.keeper.GetOrdersByOwner(suite.ctx, "cosmos1nonexistent")
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), orders3)
}

func (suite *KeeperTestSuite) TestUpdateOrder_NotFound() {
	order := &anteilv1.Order{
		OrderId:      "nonexistent",
		Owner:        "cosmos1test",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash123",
	}

	err := suite.keeper.UpdateOrder(suite.ctx, order)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *KeeperTestSuite) TestCancelOrder_NotFound() {
	err := suite.keeper.CancelOrder(suite.ctx, "nonexistent")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *KeeperTestSuite) TestDeleteOrder_NotFound() {
	err := suite.keeper.DeleteOrder(suite.ctx, "nonexistent")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *KeeperTestSuite) TestUpdateAuction_NotFound() {
	auction := &anteilv1.Auction{
		AuctionId:    "nonexistent",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.UpdateAuction(suite.ctx, auction)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAuctionNotFound, err)
}

func (suite *KeeperTestSuite) TestPlaceBid_AuctionNotFound() {
	err := suite.keeper.PlaceBid(suite.ctx, "nonexistent", "cosmos1bidder", "1500000")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAuctionNotFound, err)
}

// TestPlaceBid_OnlyValidators tests that only validators can place bids
func (suite *KeeperTestSuite) TestPlaceBid_OnlyValidators() {
	// Create mock ident keeper with citizen and validator
	mockIdentKeeper := &MockIdentKeeper{
		accounts: []*identv1.VerifiedAccount{
			{
				Address:      "cosmos1citizen",
				Role:         identv1.Role_ROLE_CITIZEN,
				IsActive:     true,
				IdentityHash: "hash1",
			},
			{
				Address:      "cosmos1validator",
				Role:         identv1.Role_ROLE_VALIDATOR,
				IsActive:     true,
				IdentityHash: "hash2",
			},
		},
	}
	suite.keeper.SetIdentKeeper(mockIdentKeeper)
	
	// Create auction
	auction := types.NewAuction(uint64(1000), "1000000", "1.0")
	err := suite.keeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	
	// Test 1: Citizen should NOT be able to place bid
	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, "cosmos1citizen", "2.0")
	require.Error(suite.T(), err, "Citizens should not be able to place bids")
	require.Contains(suite.T(), err.Error(), "only active validators can participate in auctions")
	
	// Test 2: Validator SHOULD be able to place bid
	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, "cosmos1validator", "2.0")
	require.NoError(suite.T(), err, "Validators should be able to place bids")
	
	// Test 3: Guest should NOT be able to place bid
	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, "cosmos1guest", "2.0")
	require.Error(suite.T(), err, "Guests should not be able to place bids")
}

// TestPlaceBid_ReservePrice tests reserve price validation
func (suite *KeeperTestSuite) TestPlaceBid_ReservePrice() {
	// Create mock ident keeper with validator
	mockIdentKeeper := &MockIdentKeeper{
		accounts: []*identv1.VerifiedAccount{
			{
				Address:      "cosmos1validator",
				Role:         identv1.Role_ROLE_VALIDATOR,
				IsActive:     true,
				IdentityHash: "hash1",
			},
		},
	}
	suite.keeper.SetIdentKeeper(mockIdentKeeper)
	
	// Create auction with reserve price "10.0"
	auction := types.NewAuction(uint64(1000), "1000000", "10.0")
	err := suite.keeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	
	// Test 1: Bid below reserve price should be rejected
	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, "cosmos1validator", "5.0")
	require.Error(suite.T(), err, "Bids below reserve price should be rejected")
	require.Contains(suite.T(), err.Error(), "below reserve price")
	
	// Test 2: Bid equal to reserve price should succeed
	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, "cosmos1validator", "10.0")
	require.NoError(suite.T(), err, "Bids equal to reserve price should succeed")
	
	// Test 3: Bid above reserve price should succeed
	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, "cosmos1validator", "15.0")
	require.NoError(suite.T(), err, "Bids above reserve price should succeed")
}

// TestPlaceBid_InvalidAmount tests PlaceBid with invalid amount format
func (suite *KeeperTestSuite) TestPlaceBid_InvalidAmount() {
	// Create mock ident keeper with validator
	mockIdentKeeper := &MockIdentKeeper{
		accounts: []*identv1.VerifiedAccount{
			{
				Address:      "cosmos1validator",
				Role:         identv1.Role_ROLE_VALIDATOR,
				IsActive:     true,
				IdentityHash: "hash1",
			},
		},
	}
	suite.keeper.SetIdentKeeper(mockIdentKeeper)
	
	// Create auction
	auction := types.NewAuction(uint64(1000), "1000000", "10.0")
	err := suite.keeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	
	// Test with invalid amount format
	err = suite.keeper.PlaceBid(suite.ctx, auction.AuctionId, "cosmos1validator", "invalid")
	require.Error(suite.T(), err, "Invalid amount format should be rejected")
	require.Contains(suite.T(), err.Error(), "invalid bid amount")
}

func (suite *KeeperTestSuite) TestSettleAuction_NotFound() {
	err := suite.keeper.SettleAuction(suite.ctx, "nonexistent")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAuctionNotFound, err)
}

func (suite *KeeperTestSuite) TestGetUserPosition_Create() {
	position := &anteilv1.UserPosition{
		Owner:        "cosmos1newuser",
		AntBalance:   "5000000",
		TotalTrades:  "10",
		TotalVolume:  "10000000",
		LastActivity: timestamppb.Now(),
	}

	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	retrieved, err := suite.keeper.GetUserPosition(suite.ctx, "cosmos1newuser")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "cosmos1newuser", retrieved.Owner)
	require.Equal(suite.T(), "5000000", retrieved.AntBalance)
	require.Equal(suite.T(), "10", retrieved.TotalTrades)
}

func (suite *KeeperTestSuite) TestExecuteTrade_BuyOrderNotFound() {
	sellOrder := &anteilv1.Order{
		OrderId:      "sell1",
		Owner:        "cosmos1seller",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_SELL,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_sell",
	}

	err := suite.keeper.SetOrder(suite.ctx, sellOrder)
	require.NoError(suite.T(), err)

	// Try to execute trade with non-existent buy order
	err = suite.keeper.ExecuteTrade(suite.ctx, "nonexistent", "sell1")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *KeeperTestSuite) TestExecuteTrade_SellOrderNotFound() {
	buyOrder := &anteilv1.Order{
		OrderId:      "buy1",
		Owner:        "cosmos1buyer",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_buy",
	}

	err := suite.keeper.SetOrder(suite.ctx, buyOrder)
	require.NoError(suite.T(), err)

	// Try to execute trade with non-existent sell order
	err = suite.keeper.ExecuteTrade(suite.ctx, "buy1", "nonexistent")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *KeeperTestSuite) TestGetAllOrders_Empty() {
	orders, err := suite.keeper.GetAllOrders(suite.ctx)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), orders)
}

func (suite *KeeperTestSuite) TestGetAllAuctions_Empty() {
	auctions, err := suite.keeper.GetAllAuctions(suite.ctx)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), auctions)
}

func (suite *KeeperTestSuite) TestGetAllTrades_Empty() {
	trades, err := suite.keeper.GetAllTrades(suite.ctx)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), trades)
}

func (suite *KeeperTestSuite) TestCreateOrder_Alias() {
	order := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        "cosmos1test",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash123",
	}

	// Test CreateOrder (alias for SetOrder)
	err := suite.keeper.CreateOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	retrieved, err := suite.keeper.GetOrder(suite.ctx, "order1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), order.OrderId, retrieved.OrderId)
}

func (suite *KeeperTestSuite) TestCreateAuction_Alias() {
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(24 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	// Test CreateAuction (alias for SetAuction)
	err := suite.keeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), auction.AuctionId, retrieved.AuctionId)
}

// Mock IdentKeeperInterface for testing
type MockIdentKeeper struct {
	accounts []*identv1.VerifiedAccount
	err      error
}

func (m *MockIdentKeeper) GetAllVerifiedAccounts(ctx sdk.Context) ([]*identv1.VerifiedAccount, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.accounts, nil
}

// Test DistributeAntToCitizens
func (suite *KeeperTestSuite) TestDistributeAntToCitizens() {
	// Create mock ident keeper with citizens
	mockIdentKeeper := &MockIdentKeeper{
		accounts: []*identv1.VerifiedAccount{
			{
				Address:      "cosmos1citizen1",
				Role:         identv1.Role_ROLE_CITIZEN,
				IsActive:     true,
				IdentityHash: "hash1",
			},
			{
				Address:      "cosmos1citizen2",
				Role:         identv1.Role_ROLE_CITIZEN,
				IsActive:     true,
				IdentityHash: "hash2",
			},
		},
	}

	// Set mock ident keeper
	suite.keeper.SetIdentKeeper(mockIdentKeeper)

	// Set custom params for testing
	params := suite.keeper.GetParams(suite.ctx)
	params.CitizenAntRewardRate = "5000000" // 5 ANT in micro units
	params.CitizenAntAccumulationLimit = "100000000" // 100 ANT limit
	suite.keeper.SetParams(suite.ctx, params)

	// Distribute ANT
	err := suite.keeper.DistributeAntToCitizens(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify ANT was distributed to both citizens
	position1, err := suite.keeper.GetUserPosition(suite.ctx, "cosmos1citizen1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "5000000", position1.AntBalance)

	position2, err := suite.keeper.GetUserPosition(suite.ctx, "cosmos1citizen2")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "5000000", position2.AntBalance)

	// Check events
	events := suite.ctx.EventManager().Events()
	require.GreaterOrEqual(suite.T(), len(events), 2, "Should have at least 2 events")

	// Find ant_distributed events
	antDistributedCount := 0
	for _, event := range events {
		if event.Type == "anteil.ant_distributed" {
			antDistributedCount++
		}
	}
	require.Equal(suite.T(), 2, antDistributedCount, "Should have 2 ant_distributed events")
}

func (suite *KeeperTestSuite) TestDistributeAntToCitizens_RespectsLimit() {
	// Create mock ident keeper with one citizen
	mockIdentKeeper := &MockIdentKeeper{
		accounts: []*identv1.VerifiedAccount{
			{
				Address:      "cosmos1citizen1",
				Role:         identv1.Role_ROLE_CITIZEN,
				IsActive:     true,
				IdentityHash: "hash1",
			},
		},
	}

	// Set mock ident keeper
	suite.keeper.SetIdentKeeper(mockIdentKeeper)

	// Set custom params: reward rate 50 ANT, limit 100 ANT
	params := suite.keeper.GetParams(suite.ctx)
	params.CitizenAntRewardRate = "50000000" // 50 ANT
	params.CitizenAntAccumulationLimit = "100000000" // 100 ANT limit
	suite.keeper.SetParams(suite.ctx, params)

	// Create position with balance close to limit (90 ANT)
	position := types.NewUserPosition("cosmos1citizen1", "90000000")
	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	// Distribute ANT - should cap at limit (100 ANT)
	err = suite.keeper.DistributeAntToCitizens(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify balance is capped at limit
	position, err = suite.keeper.GetUserPosition(suite.ctx, "cosmos1citizen1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "100000000", position.AntBalance, "Balance should be capped at limit")

	// Try to distribute again - should skip (already at limit)
	err = suite.keeper.DistributeAntToCitizens(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify balance is still at limit
	position, err = suite.keeper.GetUserPosition(suite.ctx, "cosmos1citizen1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "100000000", position.AntBalance, "Balance should remain at limit")
}

func (suite *KeeperTestSuite) TestDistributeAntToCitizens_OnlyCitizens() {
	// Create mock ident keeper with mixed roles
	mockIdentKeeper := &MockIdentKeeper{
		accounts: []*identv1.VerifiedAccount{
			{
				Address:      "cosmos1citizen",
				Role:         identv1.Role_ROLE_CITIZEN,
				IsActive:     true,
				IdentityHash: "hash1",
			},
			{
				Address:      "cosmos1validator",
				Role:         identv1.Role_ROLE_VALIDATOR,
				IsActive:     true,
				IdentityHash: "hash2",
			},
			{
				Address:      "cosmos1guest",
				Role:         identv1.Role_ROLE_GUEST,
				IsActive:     true,
				IdentityHash: "hash3",
			},
			{
				Address:      "cosmos1inactive",
				Role:         identv1.Role_ROLE_CITIZEN,
				IsActive:     false, // Inactive citizen
				IdentityHash: "hash4",
			},
		},
	}

	// Set mock ident keeper
	suite.keeper.SetIdentKeeper(mockIdentKeeper)

	// Set custom params
	params := suite.keeper.GetParams(suite.ctx)
	params.CitizenAntRewardRate = "10000000" // 10 ANT
	params.CitizenAntAccumulationLimit = "1000000000" // 1000 ANT limit
	suite.keeper.SetParams(suite.ctx, params)

	// Distribute ANT
	err := suite.keeper.DistributeAntToCitizens(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify active citizen received ANT
	position, err := suite.keeper.GetUserPosition(suite.ctx, "cosmos1citizen")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "10000000", position.AntBalance, "Active citizen should receive ANT")

	// UPDATED: Verify validator ALSO received ANT (validators have all citizen rights)
	validatorPosition, err := suite.keeper.GetUserPosition(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err, "Validator should have position - validators have all citizen rights")
	require.Equal(suite.T(), "10000000", validatorPosition.AntBalance, "Active validator should receive ANT")

	// Verify guest did not receive ANT
	_, err = suite.keeper.GetUserPosition(suite.ctx, "cosmos1guest")
	require.Error(suite.T(), err, "Guest should not have position")

	// Verify inactive citizen did not receive ANT
	_, err = suite.keeper.GetUserPosition(suite.ctx, "cosmos1inactive")
	require.Error(suite.T(), err, "Inactive citizen should not have position")

	// Check events - should have 2 ant_distributed events (citizen + validator)
	events := suite.ctx.EventManager().Events()
	antDistributedCount := 0
	recipientAddresses := []string{}
	for _, event := range events {
		if event.Type == "anteil.ant_distributed" {
			antDistributedCount++
			// Collect recipient addresses
			for _, attr := range event.Attributes {
				if string(attr.Key) == "citizen" {
					recipientAddresses = append(recipientAddresses, string(attr.Value))
				}
			}
		}
	}
	require.Equal(suite.T(), 2, antDistributedCount, "Should have exactly 2 ANT distribution events (citizen + validator)")
	require.Contains(suite.T(), recipientAddresses, "cosmos1citizen", "Citizen should receive ANT")
	require.Contains(suite.T(), recipientAddresses, "cosmos1validator", "Validator should receive ANT")
}

// Test BurnAntFromUser
func (suite *KeeperTestSuite) TestBurnAntFromUser() {
	// Create user position with ANT balance
	position := types.NewUserPosition("cosmos1citizen1", "50000000") // 50 ANT
	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	// Verify position exists with balance
	position, err = suite.keeper.GetUserPosition(suite.ctx, "cosmos1citizen1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "50000000", position.AntBalance)

	// Burn ANT
	err = suite.keeper.BurnAntFromUser(suite.ctx, "cosmos1citizen1")
	require.NoError(suite.T(), err)

	// Verify balance is now zero
	position, err = suite.keeper.GetUserPosition(suite.ctx, "cosmos1citizen1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "0", position.AntBalance)
	require.Equal(suite.T(), "0", position.AvailableAnt)
	require.Equal(suite.T(), "0", position.LockedAnt)
}

func (suite *KeeperTestSuite) TestBurnAntFromUser_NoPosition() {
	// Try to burn from non-existent position
	err := suite.keeper.BurnAntFromUser(suite.ctx, "cosmos1nonexistent")
	require.NoError(suite.T(), err) // Should not error, just return nil
}

func (suite *KeeperTestSuite) TestBurnAntFromUser_ZeroBalance() {
	// Create position with zero balance
	position := types.NewUserPosition("cosmos1citizen1", "0")
	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	// Try to burn from zero balance
	err = suite.keeper.BurnAntFromUser(suite.ctx, "cosmos1citizen1")
	require.NoError(suite.T(), err) // Should not error

	// Verify balance is still zero
	position, err = suite.keeper.GetUserPosition(suite.ctx, "cosmos1citizen1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "0", position.AntBalance)
}

// TestGetLastDistributionTime tests getting the last distribution time
func (suite *KeeperTestSuite) TestGetLastDistributionTime() {
	// Initially, should return zero time
	lastTime, err := suite.keeper.GetLastDistributionTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), lastTime.IsZero(), "Initial distribution time should be zero")

	// Set a distribution time
	testTime := time.Now()
	err = suite.keeper.SetLastDistributionTime(suite.ctx, testTime)
	require.NoError(suite.T(), err)

	// Retrieve it
	retrievedTime, err := suite.keeper.GetLastDistributionTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.False(suite.T(), retrievedTime.IsZero(), "Retrieved time should not be zero")
	require.WithinDuration(suite.T(), testTime, retrievedTime, time.Second, "Retrieved time should match set time")
}

// TestSetLastDistributionTime tests setting the last distribution time
func (suite *KeeperTestSuite) TestSetLastDistributionTime() {
	// Set a distribution time
	testTime := time.Now()
	err := suite.keeper.SetLastDistributionTime(suite.ctx, testTime)
	require.NoError(suite.T(), err)

	// Verify it was set
	retrievedTime, err := suite.keeper.GetLastDistributionTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.WithinDuration(suite.T(), testTime, retrievedTime, time.Second, "Retrieved time should match set time")

	// Update to a different time
	newTime := testTime.Add(24 * time.Hour)
	err = suite.keeper.SetLastDistributionTime(suite.ctx, newTime)
	require.NoError(suite.T(), err)

	// Verify it was updated
	updatedTime, err := suite.keeper.GetLastDistributionTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.WithinDuration(suite.T(), newTime, updatedTime, time.Second, "Updated time should match new time")
}

// TestBeginBlocker_WithAntDistribution tests BeginBlocker with ANT distribution to citizens
func (suite *KeeperTestSuite) TestBeginBlocker_WithAntDistribution() {
	// Set up mock ident keeper with citizens
	mockIdentKeeper := &MockIdentKeeper{
		accounts: []*identv1.VerifiedAccount{
			{
				Address:     "cosmos1citizen1",
				Role:        identv1.Role_ROLE_CITIZEN,
				IsActive:    true,
				IdentityHash: "hash1",
			},
			{
				Address:     "cosmos1citizen2",
				Role:        identv1.Role_ROLE_CITIZEN,
				IsActive:    true,
				IdentityHash: "hash2",
			},
		},
	}
	suite.keeper.SetIdentKeeper(mockIdentKeeper)

	// Set distribution period to a short duration for testing
	params := suite.keeper.GetParams(suite.ctx)
	params.CitizenAntDistributionPeriod = time.Hour // 1 hour
	suite.keeper.SetParams(suite.ctx, params)

	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Set last distribution time to past (more than distribution period ago)
	pastTime := currentTime.Add(-2 * time.Hour)
	err := suite.keeper.SetLastDistributionTime(suite.ctx, pastTime)
	require.NoError(suite.T(), err)

	// Run BeginBlocker - should trigger ANT distribution
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify last distribution time was updated
	lastTime, err := suite.keeper.GetLastDistributionTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.WithinDuration(suite.T(), currentTime, lastTime, time.Second, "Last distribution time should be updated to current time")

	// Verify citizens received ANT
	position1, err := suite.keeper.GetUserPosition(suite.ctx, "cosmos1citizen1")
	require.NoError(suite.T(), err)
	require.NotEqual(suite.T(), "0", position1.AntBalance, "Citizen 1 should have received ANT")

	position2, err := suite.keeper.GetUserPosition(suite.ctx, "cosmos1citizen2")
	require.NoError(suite.T(), err)
	require.NotEqual(suite.T(), "0", position2.AntBalance, "Citizen 2 should have received ANT")
}

// TestBeginBlocker_WithoutAntDistribution tests BeginBlocker when distribution period hasn't passed
func (suite *KeeperTestSuite) TestBeginBlocker_WithoutAntDistribution() {
	// Set up mock ident keeper
	mockIdentKeeper := &MockIdentKeeper{
		accounts: []*identv1.VerifiedAccount{
			{
				Address:     "cosmos1citizen1",
				Role:        identv1.Role_ROLE_CITIZEN,
				IsActive:    true,
				IdentityHash: "hash1",
			},
		},
	}
	suite.keeper.SetIdentKeeper(mockIdentKeeper)

	// Set distribution period to a long duration
	params := suite.keeper.GetParams(suite.ctx)
	params.CitizenAntDistributionPeriod = 24 * time.Hour // 24 hours
	suite.keeper.SetParams(suite.ctx, params)

	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Set last distribution time to recent (less than distribution period ago)
	recentTime := currentTime.Add(-1 * time.Hour) // 1 hour ago
	err := suite.keeper.SetLastDistributionTime(suite.ctx, recentTime)
	require.NoError(suite.T(), err)

	// Run BeginBlocker - should NOT trigger ANT distribution
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify last distribution time was NOT updated
	lastTime, err := suite.keeper.GetLastDistributionTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.WithinDuration(suite.T(), recentTime, lastTime, time.Second, "Last distribution time should not be updated")
}

// TestEndBlocker_WithOrders tests EndBlocker with orders to process
func (suite *KeeperTestSuite) TestEndBlocker_WithOrders() {
	// Create some orders
	order1 := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        "cosmos1test",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash1",
	}
	err := suite.keeper.SetOrder(suite.ctx, order1)
	require.NoError(suite.T(), err)

	order2 := &anteilv1.Order{
		OrderId:      "order2",
		Owner:        "cosmos1test2",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_SELL,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash2",
	}
	err = suite.keeper.SetOrder(suite.ctx, order2)
	require.NoError(suite.T(), err)

	// Run EndBlocker - should process orders
	err = suite.keeper.EndBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// EndBlocker should complete without errors
	// (actual order matching logic is tested separately)
}

// TestEndBlocker_WithAuctions tests EndBlocker with auctions to process
func (suite *KeeperTestSuite) TestEndBlocker_WithAuctions() {
	// Create an auction
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(1 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}
	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Run EndBlocker - should process auctions
	err = suite.keeper.EndBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// EndBlocker should complete without errors
	// (actual auction processing logic is tested separately)
}

// TestGetOrder_UnmarshalError tests GetOrder with invalid data in store
func (suite *KeeperTestSuite) TestGetOrder_UnmarshalError() {
	store := suite.ctx.KVStore(suite.storeKey)
	orderKey := types.GetOrderKey("order1")
	// Store invalid data
	store.Set(orderKey, []byte("invalid data"))

	// Should return error when unmarshaling fails
	_, err := suite.keeper.GetOrder(suite.ctx, "order1")
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "failed to unmarshal")
}

// TestUpdateOrder_UnmarshalError tests UpdateOrder with invalid data
func (suite *KeeperTestSuite) TestUpdateOrder_UnmarshalError() {
	// First create a valid order
	order := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        "cosmos1test",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash1",
	}
	err := suite.keeper.SetOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Update with valid order
	order.Price = "2.0"
	err = suite.keeper.UpdateOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Verify update
	updated, err := suite.keeper.GetOrder(suite.ctx, "order1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "2.0", updated.Price)
}

// TestSetTrade_InvalidData tests SetTrade with various edge cases
func (suite *KeeperTestSuite) TestSetTrade_InvalidData() {
	// Test with duplicate trade ID
	trade1 := &anteilv1.Trade{
		TradeId:     "trade1",
		BuyOrderId:  "order1",
		SellOrderId: "order2",
		AntAmount:   "1000000",
		Price:       "1.5",
		ExecutedAt:  timestamppb.Now(),
		Buyer:       "cosmos1buyer",
		Seller:      "cosmos1seller",
	}
	err := suite.keeper.SetTrade(suite.ctx, trade1)
	require.NoError(suite.T(), err)

	// Try to set the same trade again - should fail
	err = suite.keeper.SetTrade(suite.ctx, trade1)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrTradeAlreadyExists, err)
}

// TestGetTrade_UnmarshalError tests GetTrade with invalid data in store
func (suite *KeeperTestSuite) TestGetTrade_UnmarshalError() {
	store := suite.ctx.KVStore(suite.storeKey)
	tradeKey := types.GetTradeKey("trade1")
	// Store invalid data
	store.Set(tradeKey, []byte("invalid data"))

	// Should return error when unmarshaling fails
	_, err := suite.keeper.GetTrade(suite.ctx, "trade1")
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "failed to unmarshal")
}

// TestBeginBlocker_FirstDistribution tests BeginBlocker when it's the first distribution
func (suite *KeeperTestSuite) TestBeginBlocker_FirstDistribution() {
	// Set up mock ident keeper with citizens
	mockIdentKeeper := &MockIdentKeeper{
		accounts: []*identv1.VerifiedAccount{
			{
				Address:     "cosmos1citizen1",
				Role:        identv1.Role_ROLE_CITIZEN,
				IsActive:    true,
				IdentityHash: "hash1",
			},
		},
	}
	suite.keeper.SetIdentKeeper(mockIdentKeeper)

	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Don't set last distribution time (should be zero, triggering first distribution)

	// Run BeginBlocker - should trigger ANT distribution (first time)
	err := suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify last distribution time was set
	lastTime, err := suite.keeper.GetLastDistributionTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.False(suite.T(), lastTime.IsZero(), "Last distribution time should be set after first distribution")
}

// TestBeginBlocker_DistributionError tests BeginBlocker when distribution fails
func (suite *KeeperTestSuite) TestBeginBlocker_DistributionError() {
	// Set up mock ident keeper with no accounts (distribution will have no effect)
	mockIdentKeeper := &MockIdentKeeper{
		accounts: []*identv1.VerifiedAccount{},
	}
	suite.keeper.SetIdentKeeper(mockIdentKeeper)

	// Set distribution period to a short duration
	params := suite.keeper.GetParams(suite.ctx)
	params.CitizenAntDistributionPeriod = time.Hour
	suite.keeper.SetParams(suite.ctx, params)

	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Set last distribution time to past
	pastTime := currentTime.Add(-2 * time.Hour)
	err := suite.keeper.SetLastDistributionTime(suite.ctx, pastTime)
	require.NoError(suite.T(), err)

	// Run BeginBlocker - should handle error gracefully
	err = suite.keeper.BeginBlocker(suite.ctx)
	// BeginBlocker should not fail even if distribution fails
	require.NoError(suite.T(), err)
}

// TestGetAuction_UnmarshalError tests GetAuction with invalid data in store
func (suite *KeeperTestSuite) TestGetAuction_UnmarshalError() {
	store := suite.ctx.KVStore(suite.storeKey)
	auctionKey := types.GetAuctionKey("auction1")
	// Store invalid data
	store.Set(auctionKey, []byte("invalid data"))

	// Should return error when unmarshaling fails
	_, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "failed to unmarshal")
}

// TestGetBid_UnmarshalError tests GetBid with invalid data in store
func (suite *KeeperTestSuite) TestGetBid_UnmarshalError() {
	// First create an auction
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.Now(),
		EndTime:      timestamppb.New(time.Now().Add(1 * time.Hour)),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}
	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	store := suite.ctx.KVStore(suite.storeKey)
	bidKey := types.GetBidKey("auction1", "bid1")
	// Store invalid data
	store.Set(bidKey, []byte("invalid data"))

	// Should return error when unmarshaling fails
	_, err = suite.keeper.GetBid(suite.ctx, "auction1", "bid1")
	require.Error(suite.T(), err)
	// Error message may vary, but should indicate unmarshaling failure
	require.NotNil(suite.T(), err)
}

// TestSetTrade_Duplicate tests SetTrade with duplicate trade ID
func (suite *KeeperTestSuite) TestSetTrade_Duplicate() {
	trade := &anteilv1.Trade{
		TradeId:     "trade1",
		BuyOrderId:  "order1",
		SellOrderId: "order2",
		AntAmount:   "1000000",
		Price:       "1.5",
		ExecutedAt:  timestamppb.Now(),
		Buyer:       "cosmos1buyer",
		Seller:      "cosmos1seller",
	}

	// First time should succeed
	err := suite.keeper.SetTrade(suite.ctx, trade)
	require.NoError(suite.T(), err)

	// Second time should fail
	err = suite.keeper.SetTrade(suite.ctx, trade)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrTradeAlreadyExists, err)
}
