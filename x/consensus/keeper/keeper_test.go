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

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
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
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)
	suite.storeKey = storetypes.NewKVStoreKey(types.StoreKey)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore = suite.paramStore.WithKeyTable(types.ParamKeyTable())
	suite.keeper = keeper.NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)
	suite.keeper.SetParams(suite.ctx, *types.DefaultParams())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestSetValidator() {
	validator := &consensusv1.Validator{
		Validator:     "cosmos1validator",
		AntBalance:    "1000000",
		ActivityScore: "500",
		Status:        consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
		LastActive:    timestamppb.Now(),
	}
	suite.keeper.SetValidator(suite.ctx, validator)
	retrieved, err := suite.keeper.GetValidator(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), validator.Validator, retrieved.Validator)
}

func (suite *KeeperTestSuite) TestSelectBlockCreator() {
	validators := []*consensusv1.Validator{
		{
			Validator:     "cosmos1validator1",
			AntBalance:    "1000000",
			ActivityScore: "500",
			Status:        consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
			LastActive:    timestamppb.Now(),
		},
	}
	for _, v := range validators {
		suite.keeper.SetValidator(suite.ctx, v)
	}
	blockCreator, err := suite.keeper.SelectBlockCreator(suite.ctx, 1000)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), blockCreator)
}

func (suite *KeeperTestSuite) TestCalculateBlockTime() {
	blockTime, err := suite.keeper.CalculateBlockTime(suite.ctx, "10000000")
	require.NoError(suite.T(), err)
	require.Greater(suite.T(), blockTime, time.Duration(0))
}

func (suite *KeeperTestSuite) TestProcessHalving() {
	info := types.HalvingInfo{
		LastHalvingHeight: 0,
		HalvingInterval:   100000,
		NextHalvingHeight: 100000,
	}
	err := suite.keeper.SetHalvingInfo(suite.ctx, info)
	require.NoError(suite.T(), err)
	suite.ctx = suite.ctx.WithBlockHeight(100000)
	err = suite.keeper.ProcessHalving(suite.ctx)
	require.NoError(suite.T(), err)
	retrieved, err := suite.keeper.GetHalvingInfo(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(100000), retrieved.LastHalvingHeight)
}
