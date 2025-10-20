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

type KeeperTestSuite struct {
	suite.Suite

	cdc        codec.Codec
	ctx        sdk.Context
	keeper     *Keeper
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
}

func (suite *KeeperTestSuite) SetupTest() {
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

	// Create keeper
	suite.keeper = NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)

	// Set default params
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func (suite *KeeperTestSuite) TestNewKeeper() {
	require.NotNil(suite.T(), suite.keeper)
	require.Equal(suite.T(), suite.cdc, suite.keeper.cdc)
	require.Equal(suite.T(), suite.storeKey, suite.keeper.storeKey)
	require.Equal(suite.T(), suite.paramStore, suite.keeper.paramstore)
}

func (suite *KeeperTestSuite) TestGetSetParams() {
	// Test default params
	params := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), types.DefaultParams(), params)

	// Test setting new params
	newParams := types.DefaultParams()
	newParams.MaxOpenOrders = 5
	suite.keeper.SetParams(suite.ctx, newParams)

	// Verify params were set
	retrievedParams := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), newParams, retrievedParams)
}

func (suite *KeeperTestSuite) TestCreateOrder() {
	// Test creating order
	order := types.NewOrder("cosmos1test", anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_BUY, "1000", "1.5", "hash123")
	err := suite.keeper.CreateOrder(suite.ctx, order)
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), order.OrderId)

	// Verify order was created
	retrievedOrder, err := suite.keeper.GetOrder(suite.ctx, order.OrderId)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), order.Owner, retrievedOrder.Owner)
}

func (suite *KeeperTestSuite) TestGetOrder() {
	// Create order first
	order := types.NewOrder("cosmos1test", anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_BUY, "1000", "1.5", "hash123")
	err := suite.keeper.CreateOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Test getting existing order
	retrievedOrder, err := suite.keeper.GetOrder(suite.ctx, order.OrderId)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), order.Owner, retrievedOrder.Owner)

	// Test getting non-existent order
	_, err = suite.keeper.GetOrder(suite.ctx, "non_existent_id")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *KeeperTestSuite) TestCancelOrder() {
	// Create order first
	order := types.NewOrder("cosmos1test", anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_BUY, "1000", "1.5", "hash123")
	err := suite.keeper.CreateOrder(suite.ctx, order)
	require.NoError(suite.T(), err)

	// Test canceling order
	err = suite.keeper.CancelOrder(suite.ctx, order.OrderId)
	require.NoError(suite.T(), err)

	// Verify order is canceled
	retrievedOrder, err := suite.keeper.GetOrder(suite.ctx, order.OrderId)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.OrderStatus_ORDER_STATUS_CANCELLED, retrievedOrder.Status)

	// Test canceling non-existent order
	err = suite.keeper.CancelOrder(suite.ctx, "non_existent_id")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrOrderNotFound, err)
}

func (suite *KeeperTestSuite) TestCreateAuction() {
	// Test creating auction
	auction := types.NewAuction(uint64(1000), "1000000", "1.0")
	err := suite.keeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), auction.AuctionId)

	// Test getting auction
	retrievedAuction, err := suite.keeper.GetAuction(suite.ctx, auction.AuctionId)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(1000), retrievedAuction.BlockHeight)
	require.Equal(suite.T(), "1000000", retrievedAuction.AntAmount)
	require.Equal(suite.T(), anteilv1.AuctionStatus_AUCTION_STATUS_OPEN, retrievedAuction.Status)
}

func (suite *KeeperTestSuite) TestGetAuction() {
	// Create auction first
	auction := types.NewAuction(uint64(1000), "1000000", "1.0")
	err := suite.keeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Test getting existing auction
	retrievedAuction, err := suite.keeper.GetAuction(suite.ctx, auction.AuctionId)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(1000), retrievedAuction.BlockHeight)

	// Test getting non-existent auction
	_, err = suite.keeper.GetAuction(suite.ctx, "non_existent_id")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAuctionNotFound, err)
}

func (suite *KeeperTestSuite) TestGetAllOrders() {
	// Create multiple orders
	order1 := types.NewOrder("cosmos1test", anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_BUY, "1000", "1.5", "hash123")
	order2 := types.NewOrder("cosmos2test", anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_SELL, "2000", "2.0", "hash456")

	err := suite.keeper.CreateOrder(suite.ctx, order1)
	require.NoError(suite.T(), err)

	err = suite.keeper.CreateOrder(suite.ctx, order2)
	require.NoError(suite.T(), err)

	// Test getting all orders
	orders, err := suite.keeper.GetAllOrders(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders, 2)

	// Verify all orders are present
	orderIDs := make([]string, 2)
	for i, order := range orders {
		orderIDs[i] = order.OrderId
	}
	require.Contains(suite.T(), orderIDs, order1.OrderId)
	require.Contains(suite.T(), orderIDs, order2.OrderId)
}

func (suite *KeeperTestSuite) TestGetAllAuctions() {
	// Create multiple auctions
	auction1 := types.NewAuction(uint64(1000), "1000000", "1.0")
	auction2 := types.NewAuction(uint64(2000), "2000000", "2.0")

	err := suite.keeper.CreateAuction(suite.ctx, auction1)
	require.NoError(suite.T(), err)

	err = suite.keeper.CreateAuction(suite.ctx, auction2)
	require.NoError(suite.T(), err)

	// Test getting all auctions
	auctions, err := suite.keeper.GetAllAuctions(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), auctions, 2)

	// Verify all auctions are present
	auctionIDs := make([]string, 2)
	for i, auction := range auctions {
		auctionIDs[i] = auction.AuctionId
	}
	require.Contains(suite.T(), auctionIDs, auction1.AuctionId)
	require.Contains(suite.T(), auctionIDs, auction2.AuctionId)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
