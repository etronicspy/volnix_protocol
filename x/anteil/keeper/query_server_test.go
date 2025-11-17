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

type QueryServerTestSuite struct {
	suite.Suite

	cdc         codec.Codec
	ctx         sdk.Context
	keeper      *Keeper
	queryServer QueryServer
	storeKey    storetypes.StoreKey
	paramStore  paramtypes.Subspace
}

func (suite *QueryServerTestSuite) SetupTest() {
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)
	suite.storeKey = storetypes.NewKVStoreKey(types.StoreKey)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore = suite.paramStore.WithKeyTable(types.ParamKeyTable())
	suite.keeper = NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)
	suite.queryServer = NewQueryServer(suite.keeper)
	// Set default params
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func TestQueryServerTestSuite(t *testing.T) {
	suite.Run(t, new(QueryServerTestSuite))
}

func (suite *QueryServerTestSuite) TestParams() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &anteilv1.QueryParamsRequest{}
	
	resp, err := suite.queryServer.Params(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotEmpty(suite.T(), resp.Json)
}

func (suite *QueryServerTestSuite) TestOrders() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &anteilv1.QueryOrdersRequest{}
	
	resp, err := suite.queryServer.Orders(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	// Orders can be nil or empty slice - both are valid
	if resp.Orders != nil {
		require.Empty(suite.T(), resp.Orders)
	}
}

func (suite *QueryServerTestSuite) TestTrades() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &anteilv1.QueryTradesRequest{}
	
	resp, err := suite.queryServer.Trades(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	// Trades can be nil or empty slice - both are valid
	if resp.Trades != nil {
		require.Empty(suite.T(), resp.Trades)
	}
}

func (suite *QueryServerTestSuite) TestAuctions() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &anteilv1.QueryAuctionsRequest{}
	
	resp, err := suite.queryServer.Auctions(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	// Auctions can be nil or empty slice - both are valid
	if resp.Auctions != nil {
		require.Empty(suite.T(), resp.Auctions)
	}
}

