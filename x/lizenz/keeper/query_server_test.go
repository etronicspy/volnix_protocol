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
	"google.golang.org/protobuf/types/known/timestamppb"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
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

func (suite *QueryServerTestSuite) TestGetRewardHistory() {
	validator := "cosmos1validator"
	
	// Create activated lizenz with reward history
	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:         validator,
		Amount:            "1000000",
		ActivationTime:    timestamppb.Now(),
		IdentityHash:      "test_identity_hash",
		TotalRewardsEarned: "5000000",
		LastRewardBlock:   "1000",
		LastRewardTime:    timestamppb.Now(),
	}
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Add some reward history using UpdateRewardStats
	err = suite.keeper.UpdateRewardStats(suite.ctx, validator, 1000000, 1000, 0.95, 1.0, 1000000)
	require.NoError(suite.T(), err)
	err = suite.keeper.UpdateRewardStats(suite.ctx, validator, 2000000, 2000, 1.0, 1.0, 2000000)
	require.NoError(suite.T(), err)

	// Query reward history
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &lizenzv1.QueryRewardHistoryRequest{
		Validator: validator,
	}
	
	resp, err := suite.queryServer.GetRewardHistory(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.Equal(suite.T(), validator, resp.Validator)
	require.GreaterOrEqual(suite.T(), len(resp.RewardHistory), 2)
	require.Equal(suite.T(), uint64(2), resp.TotalRecords)
}

func (suite *QueryServerTestSuite) TestGetRewardHistory_NilRequest() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	
	_, err := suite.queryServer.GetRewardHistory(ctx, nil)
	require.Error(suite.T(), err)
}

func (suite *QueryServerTestSuite) TestGetRewardStats() {
	validator := "cosmos1validator"
	
	// Create activated lizenz
	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:         validator,
		Amount:            "1000000",
		ActivationTime:    timestamppb.Now(),
		IdentityHash:      "test_identity_hash",
		TotalRewardsEarned: "5000000",
		LastRewardBlock:   "1000",
		LastRewardTime:    timestamppb.Now(),
	}
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Add reward history using UpdateRewardStats
	err = suite.keeper.UpdateRewardStats(suite.ctx, validator, 1000000, 1000, 0.95, 1.0, 1000000)
	require.NoError(suite.T(), err)

	// Query reward stats
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &lizenzv1.QueryRewardStatsRequest{
		Validator: validator,
	}
	
	resp, err := suite.queryServer.GetRewardStats(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	// TotalRewardsEarned should be 5000000 (initial) + 1000000 (from UpdateRewardStats) = 6000000
	require.Equal(suite.T(), "6000000", resp.TotalRewardsEarned)
	require.Equal(suite.T(), "1000", resp.LastRewardBlock)
	require.NotNil(suite.T(), resp.LastRewardTime)
	require.GreaterOrEqual(suite.T(), len(resp.RewardHistory), 1)
	require.GreaterOrEqual(suite.T(), resp.TotalRewardsCount, uint64(1))
}

func (suite *QueryServerTestSuite) TestGetRewardStats_NilRequest() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	
	_, err := suite.queryServer.GetRewardStats(ctx, nil)
	require.Error(suite.T(), err)
}

