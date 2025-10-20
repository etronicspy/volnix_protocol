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

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
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
	suite.storeKey = storetypes.NewKVStoreKey("test_consensus")
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
	suite.keeper.SetParams(suite.ctx, *types.DefaultParams())
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
	require.Equal(suite.T(), types.DefaultParams(), &params)

	// Test setting new params
	newParams := types.DefaultParams()
	newParams.BaseBlockTime = "10s"
	suite.keeper.SetParams(suite.ctx, *newParams)

	// Verify params were set
	retrievedParams := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), *newParams, retrievedParams)
}

func (suite *KeeperTestSuite) TestSelectBlockProducer() {
	// Create some validators
	validators := []string{"cosmos1test1", "cosmos2test2", "cosmos3test3"}

	// Test selecting block producer
	producer, err := suite.keeper.SelectBlockProducer(suite.ctx, validators)
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), producer)
	require.Contains(suite.T(), validators, producer)

	// Test with empty validators list
	_, err = suite.keeper.SelectBlockProducer(suite.ctx, []string{})
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrNoValidators, err)
}

func (suite *KeeperTestSuite) TestCalculateBlockTime() {
	// Test calculating block time
	blockTime, err := suite.keeper.CalculateBlockTime(suite.ctx, "1000000")
	require.NoError(suite.T(), err)
	require.NotZero(suite.T(), blockTime)

	// Test with zero ant amount
	_, err = suite.keeper.CalculateBlockTime(suite.ctx, "0")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrInvalidAntAmount, err)
}

func (suite *KeeperTestSuite) TestProcessHalving() {
	// Test processing halving
	err := suite.keeper.ProcessHalving(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify halving was processed
	halvingInfo, err := suite.keeper.GetHalvingInfo(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), halvingInfo)
}

func (suite *KeeperTestSuite) TestGetHalvingInfo() {
	// Test getting halving info
	halvingInfo, err := suite.keeper.GetHalvingInfo(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), halvingInfo)

	// Verify default values
	require.Equal(suite.T(), uint64(0), halvingInfo.LastHalvingHeight)
	require.Equal(suite.T(), uint64(100000), halvingInfo.HalvingInterval)
	require.Equal(suite.T(), uint64(100000), halvingInfo.NextHalvingHeight)
}

func (suite *KeeperTestSuite) TestSetHalvingInfo() {
	// Create halving info
	halvingInfo := &consensusv1.HalvingInfo{
		LastHalvingHeight: 100000,
		HalvingInterval:   100000,
		NextHalvingHeight: 200000,
	}

	// Test setting halving info
	err := suite.keeper.SetHalvingInfo(suite.ctx, *halvingInfo)
	require.NoError(suite.T(), err)

	// Verify halving info was set
	retrievedInfo, err := suite.keeper.GetHalvingInfo(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), halvingInfo.LastHalvingHeight, retrievedInfo.LastHalvingHeight)
	require.Equal(suite.T(), halvingInfo.HalvingInterval, retrievedInfo.HalvingInterval)
	require.Equal(suite.T(), halvingInfo.NextHalvingHeight, retrievedInfo.NextHalvingHeight)
}

func (suite *KeeperTestSuite) TestGetConsensusState() {
	// Test getting consensus state
	state, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), state)

	// Verify default values
	require.Equal(suite.T(), uint64(0), state.CurrentHeight)
	require.Equal(suite.T(), "0", state.TotalAntBurned)
	require.NotNil(suite.T(), state.LastBlockTime)
	require.Empty(suite.T(), state.ActiveValidators)
}

func (suite *KeeperTestSuite) TestSetConsensusState() {
	// Create consensus state
	state := &consensusv1.ConsensusState{
		CurrentHeight:    1000,
		TotalAntBurned:   "1000000",
		LastBlockTime:    timestamppb.Now(),
		ActiveValidators: []string{"cosmos1test1", "cosmos2test2"},
	}

	// Test setting consensus state
	err := suite.keeper.SetConsensusState(suite.ctx, *state)
	require.NoError(suite.T(), err)

	// Verify consensus state was set
	retrievedState, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), state.CurrentHeight, retrievedState.CurrentHeight)
	require.Equal(suite.T(), state.TotalAntBurned, retrievedState.TotalAntBurned)
	require.Len(suite.T(), retrievedState.ActiveValidators, 2)
}

func (suite *KeeperTestSuite) TestUpdateConsensusState() {
	// Test updating consensus state
	err := suite.keeper.UpdateConsensusState(suite.ctx, 1000, "1000000", []string{"cosmos1test1"})
	require.NoError(suite.T(), err)

	// Verify consensus state was updated
	state, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(1000), state.CurrentHeight)
	require.Equal(suite.T(), "1000000", state.TotalAntBurned)
	require.Len(suite.T(), state.ActiveValidators, 1)
	require.Equal(suite.T(), "cosmos1test1", state.ActiveValidators[0])
}

func (suite *KeeperTestSuite) TestGetValidatorWeight() {
	// Test getting validator weight
	weight, err := suite.keeper.GetValidatorWeight(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.NotZero(suite.T(), weight)

	// Test with empty validator address
	_, err = suite.keeper.GetValidatorWeight(suite.ctx, "")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidatorAddress, err)
}

func (suite *KeeperTestSuite) TestSetValidatorWeight() {
	// Test setting validator weight
	err := suite.keeper.SetValidatorWeight(suite.ctx, "cosmos1test", "1000000")
	require.NoError(suite.T(), err)

	// Verify weight was set
	weight, err := suite.keeper.GetValidatorWeight(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "1000000", weight)

	// Test with empty validator address
	err = suite.keeper.SetValidatorWeight(suite.ctx, "", "1000000")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidatorAddress, err)
}

func (suite *KeeperTestSuite) TestGetAllValidatorWeights() {
	// Set some validator weights
	err := suite.keeper.SetValidatorWeight(suite.ctx, "cosmos1test", "1000000")
	require.NoError(suite.T(), err)

	err = suite.keeper.SetValidatorWeight(suite.ctx, "cosmos2test", "2000000")
	require.NoError(suite.T(), err)

	// Test getting all validator weights
	weights, err := suite.keeper.GetAllValidatorWeights(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), weights, 2)

	// Verify all weights are present
	validators := make([]string, 2)
	for i, weight := range weights {
		validators[i] = weight.Validator
	}
	require.Contains(suite.T(), validators, "cosmos1test")
	require.Contains(suite.T(), validators, "cosmos2test")
}

func (suite *KeeperTestSuite) TestBeginBlocker() {
	// Test BeginBlocker
	err := suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify consensus state was updated
	state, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), state)
}

func (suite *KeeperTestSuite) TestEndBlocker() {
	// Test EndBlocker
	err := suite.keeper.EndBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify consensus state was updated
	state, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), state)
}

func (suite *KeeperTestSuite) TestDefaultParams() {
	// Test that default params are valid
	params := types.DefaultParams()
	require.NoError(suite.T(), types.ValidateParams(params))

	// Test specific values
	require.Equal(suite.T(), "5s", params.BaseBlockTime)
	require.Equal(suite.T(), uint64(1000), params.HighActivityThreshold)
	require.Equal(suite.T(), uint64(100), params.LowActivityThreshold)
	require.Equal(suite.T(), "1000000uvx", params.MinBurnAmount)
	require.Equal(suite.T(), "1000000000uvx", params.MaxBurnAmount)
	require.Equal(suite.T(), uint64(10), params.BlockCreatorSelectionRounds)
	require.Equal(suite.T(), "0.95", params.ActivityDecayRate)
	require.Equal(suite.T(), "0.1", params.MoaPenaltyRate)
}

func (suite *KeeperTestSuite) TestParamsValidation() {
	// Test valid params
	params := types.DefaultParams()
	require.NoError(suite.T(), types.ValidateParams(params))

	// Test invalid params
	invalidParams := params
	invalidParams.BaseBlockTime = ""
	err := types.ValidateParams(invalidParams)
	require.Error(suite.T(), err)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
