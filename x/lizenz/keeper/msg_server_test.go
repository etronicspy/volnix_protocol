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

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

type MsgServerTestSuite struct {
	suite.Suite
	ctx        sdk.Context
	keeper     *keeper.Keeper
	msgServer  keeper.MsgServer
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
}

func TestMsgServerTestSuite(t *testing.T) {
	suite.Run(t, new(MsgServerTestSuite))
}

func (suite *MsgServerTestSuite) SetupTest() {
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)
	suite.storeKey = storetypes.NewKVStoreKey(types.StoreKey)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)
	suite.ctx = suite.ctx.WithBlockHeight(100).WithBlockTime(time.Now())
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore = suite.paramStore.WithKeyTable(types.ParamKeyTable())
	suite.keeper = keeper.NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
	suite.msgServer = keeper.NewMsgServer(suite.keeper)
}

// ActivateLZN Tests

func (suite *MsgServerTestSuite) TestActivateLZN_Success() {
	resp, err := suite.msgServer.ActivateLZN(suite.ctx, &lizenzv1.MsgActivateLZN{
		Validator:    "cosmos1validator1",
		Amount:       "1000000",
		IdentityHash: "hash123",
	})
	require.NoError(suite.T(), err)
	require.True(suite.T(), resp.Success)
	require.Contains(suite.T(), resp.ActivationId, "activation-cosmos1validator1-100")

	lzn, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "1000000", lzn.Amount)
	require.Equal(suite.T(), "hash123", lzn.IdentityHash)
	require.True(suite.T(), lzn.IsEligibleForRewards)
}

func (suite *MsgServerTestSuite) TestActivateLZN_UsesBlockTime() {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	suite.ctx = suite.ctx.WithBlockTime(now)

	_, err := suite.msgServer.ActivateLZN(suite.ctx, &lizenzv1.MsgActivateLZN{
		Validator:    "cosmos1val1",
		Amount:       "1000000",
		IdentityHash: "hash456",
	})
	require.NoError(suite.T(), err)

	lzn, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1val1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), now.Unix(), lzn.ActivationTime.AsTime().Unix())
	require.Equal(suite.T(), now.Unix(), lzn.LastActivity.AsTime().Unix())
}

func (suite *MsgServerTestSuite) TestActivateLZN_NilRequest() {
	_, err := suite.msgServer.ActivateLZN(suite.ctx, nil)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "nil")
}

func (suite *MsgServerTestSuite) TestActivateLZN_EmptyValidator() {
	_, err := suite.msgServer.ActivateLZN(suite.ctx, &lizenzv1.MsgActivateLZN{
		Validator: "",
		Amount:    "1000000",
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestActivateLZN_EmptyAmount() {
	_, err := suite.msgServer.ActivateLZN(suite.ctx, &lizenzv1.MsgActivateLZN{
		Validator: "cosmos1val1",
		Amount:    "",
	})
	require.Error(suite.T(), err)
}

// DeactivateLZN Tests

func (suite *MsgServerTestSuite) TestDeactivateLZN_Success() {
	suite.keeper.SetActivatedLizenz(suite.ctx, &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1val1",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IdentityHash:         "hash",
		IsEligibleForRewards: true,
	})

	resp, err := suite.msgServer.DeactivateLZN(suite.ctx, &lizenzv1.MsgDeactivateLZN{
		Validator: "cosmos1val1",
		Reason:    "voluntary",
	})
	require.NoError(suite.T(), err)
	require.True(suite.T(), resp.Success)
	require.Contains(suite.T(), resp.DeactivationId, "deactivation-cosmos1val1-100")

	_, err = suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1val1")
	require.Error(suite.T(), err, "LZN should be deleted after deactivation")
}

func (suite *MsgServerTestSuite) TestDeactivateLZN_NilRequest() {
	_, err := suite.msgServer.DeactivateLZN(suite.ctx, nil)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "nil")
}

func (suite *MsgServerTestSuite) TestDeactivateLZN_EmptyValidator() {
	_, err := suite.msgServer.DeactivateLZN(suite.ctx, &lizenzv1.MsgDeactivateLZN{
		Validator: "",
		Reason:    "test",
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestDeactivateLZN_EmptyReason() {
	_, err := suite.msgServer.DeactivateLZN(suite.ctx, &lizenzv1.MsgDeactivateLZN{
		Validator: "cosmos1val1",
		Reason:    "",
	})
	require.Error(suite.T(), err)
}

func (suite *MsgServerTestSuite) TestDeactivateLZN_NonexistentLZN() {
	_, err := suite.msgServer.DeactivateLZN(suite.ctx, &lizenzv1.MsgDeactivateLZN{
		Validator: "cosmos1nonexistent",
		Reason:    "test",
	})
	require.Error(suite.T(), err)
}
