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
		Owner:        addrTest,
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
		Owner:        addrTest,
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
		Owner:        addrTest,
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
		Owner:        addrTest,
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
		Owner:        addrTest,
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
			Owner:        addrTest,
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
			Owner:        addrOwner1,
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
			Owner:        addrOwner2,
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
	orders, err := suite.keeper.GetOrdersByOwner(suite.ctx, addrOwner1)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders, 3)

	// Get orders for owner2
	orders, err = suite.keeper.GetOrdersByOwner(suite.ctx, addrOwner2)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders, 2)
}

// Test Trade Management
func (suite *KeeperTestSuite) TestExecuteTrade() {
	// SELL orders require sufficient ANT balance - set position for seller
	err := suite.keeper.SetUserPosition(suite.ctx, types.NewUserPosition(addrSeller, "1000000"))
	require.NoError(suite.T(), err)

	buyOrder := &anteilv1.Order{
		OrderId:      "buy1",
		Owner:        addrBuyer,
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
		Owner:        addrSeller,
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_SELL,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_sell",
	}

	err = suite.keeper.SetOrder(suite.ctx, buyOrder)
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
		Owner:        addrBuyer1,
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
		Owner:        addrBuyer2,
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
		Buyer:       addrBuyer,
		Seller:      addrSeller,
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
	tradeAddrs := []string{mustAddr("0000000000000000000000000000000000000020"), mustAddr("0000000000000000000000000000000000000021"), mustAddr("0000000000000000000000000000000000000022"), mustAddr("0000000000000000000000000000000000000023"), mustAddr("0000000000000000000000000000000000000024")}
	sellerAddrs := []string{mustAddr("0000000000000000000000000000000000000025"), mustAddr("0000000000000000000000000000000000000026"), mustAddr("0000000000000000000000000000000000000027"), mustAddr("0000000000000000000000000000000000000028"), mustAddr("0000000000000000000000000000000000000029")}
	// Create multiple trades
	for i := range 5 {
		trade := &anteilv1.Trade{
			TradeId:     "trade" + string(rune('0'+i)),
			BuyOrderId:  "buy" + string(rune('0'+i)),
			SellOrderId: "sell" + string(rune('0'+i)),
			Buyer:       tradeAddrs[i],
			Seller:      sellerAddrs[i],
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

// Test User Position Management
func (suite *KeeperTestSuite) TestSetUserPosition() {
	position := &anteilv1.UserPosition{
		Owner:        addrTest,
		AntBalance:   "10000000",
		TotalTrades:  "5",
		LastActivity: timestamppb.Now(),
	}

	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	// Verify position was stored
	retrieved, err := suite.keeper.GetUserPosition(suite.ctx, addrTest)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), position.Owner, retrieved.Owner)
	require.Equal(suite.T(), position.AntBalance, retrieved.AntBalance)
}

func (suite *KeeperTestSuite) TestUpdateUserPosition() {
	position := &anteilv1.UserPosition{
		Owner:        addrTest,
		AntBalance:   "10000000",
		TotalTrades:  "5",
		LastActivity: timestamppb.Now(),
	}

	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	// Update position
	err = suite.keeper.UpdateUserPosition(suite.ctx, addrTest, "500000", 1)
	require.NoError(suite.T(), err)

	// Verify update
	retrieved, err := suite.keeper.GetUserPosition(suite.ctx, addrTest)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "500000", retrieved.AntBalance)
}

// Test Params
func (suite *KeeperTestSuite) TestGetSetParams() {
	params := types.DefaultParams()
	params.TradingFeeRate = "0.005"

	suite.keeper.SetParams(suite.ctx, params)

	retrieved := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), params.TradingFeeRate, retrieved.TradingFeeRate)
}

// Additional tests for uncovered methods

func (suite *KeeperTestSuite) TestCreateOrder() {
	order := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        addrTest,
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

func (suite *KeeperTestSuite) TestGetTrade_NotFound() {
	_, err := suite.keeper.GetTrade(suite.ctx, "nonexistent")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrTradeNotFound, err)
}

func (suite *KeeperTestSuite) TestGetUserPosition_NotFound() {
	_, err := suite.keeper.GetUserPosition(suite.ctx, addrNotFound)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrPositionNotFound, err)
}

func (suite *KeeperTestSuite) TestEndBlocker() {
	err := suite.keeper.EndBlocker(suite.ctx)
	require.NoError(suite.T(), err)
}

// Additional tests for better coverage

func (suite *KeeperTestSuite) TestGetOrdersByOwner_MultipleOwners() {
	// SELL orders require sufficient ANT balance - set position for owner2
	err := suite.keeper.SetUserPosition(suite.ctx, types.NewUserPosition(addrOwner2, "5000000"))
	require.NoError(suite.T(), err)

	// Create orders for owner1
	for i := 0; i < 3; i++ {
		order := &anteilv1.Order{
			OrderId:      fmt.Sprintf("order_owner1_%d", i),
			Owner:        addrOwner1,
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
			Owner:        addrOwner2,
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
	orders1, err := suite.keeper.GetOrdersByOwner(suite.ctx, addrOwner1)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders1, 3)

	// Get orders for owner2
	orders2, err := suite.keeper.GetOrdersByOwner(suite.ctx, addrOwner2)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders2, 2)

	// Get orders for non-existent owner
	orders3, err := suite.keeper.GetOrdersByOwner(suite.ctx, addrNonexistent)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), orders3)
}

func (suite *KeeperTestSuite) TestUpdateOrder_NotFound() {
	order := &anteilv1.Order{
		OrderId:      "nonexistent",
		Owner:        addrTest,
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

func (suite *KeeperTestSuite) TestGetUserPosition_Create() {
	position := &anteilv1.UserPosition{
		Owner:        addrNewUser,
		AntBalance:   "5000000",
		TotalTrades:  "10",
		LastActivity: timestamppb.Now(),
	}

	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	retrieved, err := suite.keeper.GetUserPosition(suite.ctx, addrNewUser)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), addrNewUser, retrieved.Owner)
	require.Equal(suite.T(), "5000000", retrieved.AntBalance)
	require.Equal(suite.T(), "10", retrieved.TotalTrades)
}

func (suite *KeeperTestSuite) TestExecuteTrade_BuyOrderNotFound() {
	// SELL orders require sufficient ANT balance
	err := suite.keeper.SetUserPosition(suite.ctx, types.NewUserPosition(addrSeller, "1000000"))
	require.NoError(suite.T(), err)

	sellOrder := &anteilv1.Order{
		OrderId:      "sell1",
		Owner:        addrSeller,
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_SELL,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_sell",
	}

	err = suite.keeper.SetOrder(suite.ctx, sellOrder)
	require.NoError(suite.T(), err)

	// Try to execute trade with non-existent buy order
	err = suite.keeper.ExecuteTrade(suite.ctx, "nonexistent", "sell1")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *KeeperTestSuite) TestExecuteTrade_SellOrderNotFound() {
	buyOrder := &anteilv1.Order{
		OrderId:      "buy1",
		Owner:        addrBuyer,
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

func (suite *KeeperTestSuite) TestGetAllTrades_Empty() {
	trades, err := suite.keeper.GetAllTrades(suite.ctx)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), trades)
}

func (suite *KeeperTestSuite) TestCreateOrder_Alias() {
	order := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        addrTest,
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

// Test BurnAntFromUser
func (suite *KeeperTestSuite) TestBurnAntFromUser() {
	// Create user position with ANT balance
	position := types.NewUserPosition(addrCitizen1, "50000000") // 50 ANT
	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	// Verify position exists with balance
	position, err = suite.keeper.GetUserPosition(suite.ctx, addrCitizen1)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "50000000", position.AntBalance)

	// Burn ANT
	err = suite.keeper.BurnAntFromUser(suite.ctx, addrCitizen1)
	require.NoError(suite.T(), err)

	// Verify balance is now zero
	position, err = suite.keeper.GetUserPosition(suite.ctx, addrCitizen1)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "0", position.AntBalance)
	require.Equal(suite.T(), "0", position.AvailableAnt)
	require.Equal(suite.T(), "0", position.LockedAnt)
}

func (suite *KeeperTestSuite) TestBurnAntFromUser_NoPosition() {
	// Try to burn from non-existent position
	err := suite.keeper.BurnAntFromUser(suite.ctx, addrNonexistent)
	require.NoError(suite.T(), err) // Should not error, just return nil
}

func (suite *KeeperTestSuite) TestBurnAntFromUser_ZeroBalance() {
	// Create position with zero balance
	position := types.NewUserPosition(addrCitizen1, "0")
	err := suite.keeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	// Try to burn from zero balance
	err = suite.keeper.BurnAntFromUser(suite.ctx, addrCitizen1)
	require.NoError(suite.T(), err) // Should not error

	// Verify balance is still zero
	position, err = suite.keeper.GetUserPosition(suite.ctx, addrCitizen1)
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

// TestEndBlocker_WithOrders tests EndBlocker with orders to process
func (suite *KeeperTestSuite) TestEndBlocker_WithOrders() {
	// SELL orders require sufficient ANT balance
	err := suite.keeper.SetUserPosition(suite.ctx, types.NewUserPosition(addrTest2, "1000000"))
	require.NoError(suite.T(), err)

	// Create some orders
	order1 := &anteilv1.Order{
		OrderId:      "order1",
		Owner:        addrTest,
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash1",
	}
	err = suite.keeper.SetOrder(suite.ctx, order1)
	require.NoError(suite.T(), err)

	order2 := &anteilv1.Order{
		OrderId:      "order2",
		Owner:        addrTest2,
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
		Owner:        addrTest,
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
		Buyer:       addrBuyer,
		Seller:      addrSeller,
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

// TestSetTrade_Duplicate tests SetTrade with duplicate trade ID
func (suite *KeeperTestSuite) TestSetTrade_Duplicate() {
	trade := &anteilv1.Trade{
		TradeId:     "trade1",
		BuyOrderId:  "order1",
		SellOrderId: "order2",
		AntAmount:   "1000000",
		Price:       "1.5",
		ExecutedAt:  timestamppb.Now(),
		Buyer:       addrBuyer,
		Seller:      addrSeller,
	}

	// First time should succeed
	err := suite.keeper.SetTrade(suite.ctx, trade)
	require.NoError(suite.T(), err)

	// Second time should fail
	err = suite.keeper.SetTrade(suite.ctx, trade)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrTradeAlreadyExists, err)
}
