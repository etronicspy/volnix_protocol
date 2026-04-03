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

func TestMsgServerTestSuite(t *testing.T) {
	suite.Run(t, new(MsgServerTestSuite))
}
