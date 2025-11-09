package keeper_test

import (
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
		StartTime:    timestamppb.New(currentTime.Add(-25 * time.Hour)),
		EndTime:      timestamppb.New(currentTime.Add(-1 * time.Hour)), // Past end time
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Place a bid
	err = suite.keeper.PlaceBid(suite.ctx, "auction1", "cosmos1bidder", "1500000")
	require.NoError(suite.T(), err)

	// Close the auction
	auction.Status = anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED
	err = suite.keeper.UpdateAuction(suite.ctx, auction)
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
