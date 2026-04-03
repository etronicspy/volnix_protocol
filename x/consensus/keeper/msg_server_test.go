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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

// consensusAuthority returns the consensus module address for test msg authorization
func consensusAuthority() string {
	return authtypes.NewModuleAddress(types.ModuleName).String()
}

type MsgServerTestSuite struct {
	suite.Suite

	cdc        codec.Codec
	ctx        sdk.Context
	keeper     *Keeper
	msgServer  consensusv1.MsgServer
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
}

func (suite *MsgServerTestSuite) SetupTest() {
	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)

	// Create store keys
	suite.storeKey = storetypes.NewKVStoreKey("test_consensus")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")

	// Create test context
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)

	// Create params keeper and subspace
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore.WithKeyTable(types.ParamKeyTable())

	// Create keeper and msg server
	suite.keeper = NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)
	suite.msgServer = NewMsgServer(*suite.keeper)

	// Set default params
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func (suite *MsgServerTestSuite) TestUpdateConsensusState() {
	// Test valid consensus state update
	msg := &consensusv1.MsgUpdateConsensusState{
		Authority:        consensusAuthority(),
		CurrentHeight:    1000,
		TotalAntBurned:   "1000000",
		ActiveValidators: []string{"cosmos1test1", "cosmos2test2"},
	}

	resp, err := suite.msgServer.UpdateConsensusState(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify consensus state was updated
	state, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(1000), state.CurrentHeight)
	require.Equal(suite.T(), "1000000", state.TotalAntBurned)
	require.Len(suite.T(), state.ActiveValidators, 2)

	// Test invalid authority
	invalidMsg := &consensusv1.MsgUpdateConsensusState{
		Authority:        "cosmos2test",
		CurrentHeight:    1001,
		TotalAntBurned:   "1000001",
		ActiveValidators: []string{"cosmos1test1"},
	}

	_, err = suite.msgServer.UpdateConsensusState(suite.ctx, invalidMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrUnauthorized, err)
}

func (suite *MsgServerTestSuite) TestSetValidatorWeight() {
	// Test valid validator weight setting
	msg := &consensusv1.MsgSetValidatorWeight{
		Authority: consensusAuthority(),
		Validator: "cosmos1validator",
		Weight:    "1000000",
	}

	resp, err := suite.msgServer.SetValidatorWeight(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify validator weight was set
	weight, err := suite.keeper.GetValidatorWeight(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "1000000", weight)

	// Test invalid authority
	invalidMsg := &consensusv1.MsgSetValidatorWeight{
		Authority: "cosmos2test",
		Validator: "cosmos2validator",
		Weight:    "1000000",
	}

	_, err = suite.msgServer.SetValidatorWeight(suite.ctx, invalidMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrUnauthorized, err)

	// Test empty validator address
	emptyValidatorMsg := &consensusv1.MsgSetValidatorWeight{
		Authority: consensusAuthority(),
		Validator: "",
		Weight:    "1000000",
	}

	_, err = suite.msgServer.SetValidatorWeight(suite.ctx, emptyValidatorMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidatorAddress, err)
}

func (suite *MsgServerTestSuite) TestCalculateBlockTime() {
	// MsgCalculateBlockTime was removed; block interval comes from params and rolling average.
	p := suite.keeper.GetParams(suite.ctx)
	require.NotEmpty(suite.T(), p.BaseBlockTime)

	avg, err := suite.keeper.GetAverageBlockTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.Positive(suite.T(), avg)
}

func (suite *MsgServerTestSuite) TestSelectBlockCreator() {
	suite.T().Skip("MsgSelectBlockCreator removed in spec alignment")
}

func (suite *MsgServerTestSuite) TestCommitBid() {
	suite.T().Skip("MsgCommitBid removed in spec alignment")
}

func (suite *MsgServerTestSuite) TestRevealBid() {
	suite.T().Skip("MsgRevealBid removed in spec alignment")
}

func TestMsgServerTestSuite(t *testing.T) {
	suite.Run(t, new(MsgServerTestSuite))
}
