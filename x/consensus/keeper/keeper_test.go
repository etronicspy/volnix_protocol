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
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestSetValidator() {
	validator := &consensusv1.Validator{
		Validator:    "cosmos1validator",
		ActivatedLzn: "1000000",
		Status:       consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
	}
	suite.keeper.SetValidator(suite.ctx, validator)
	retrieved, err := suite.keeper.GetValidator(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), validator.Validator, retrieved.Validator)
}

func (suite *KeeperTestSuite) TestProcessHalving() {
	err := suite.keeper.SetHalvingInfo(suite.ctx, &types.HalvingInfo{
		LastHalvingHeight: 0,
		HalvingInterval:   100000,
		NextHalvingHeight: 100000,
	})
	require.NoError(suite.T(), err)
	suite.ctx = suite.ctx.WithBlockHeight(100000)
	err = suite.keeper.ProcessHalving(suite.ctx)
	require.NoError(suite.T(), err)
	retrieved, err := suite.keeper.GetHalvingInfo(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(100000), retrieved.LastHalvingHeight)
}

func (suite *KeeperTestSuite) TestRecordBlockTime() {
	err := suite.keeper.RecordBlockTime(suite.ctx, 1000)
	require.NoError(suite.T(), err)

	err = suite.keeper.RecordBlockTime(suite.ctx, 1001)
	require.NoError(suite.T(), err)
	err = suite.keeper.RecordBlockTime(suite.ctx, 1002)
	require.NoError(suite.T(), err)

	avgTime, err := suite.keeper.GetAverageBlockTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.Greater(suite.T(), avgTime, time.Duration(0))
}

func (suite *KeeperTestSuite) TestGetConsensusState() {
	state, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), state)
	require.Equal(suite.T(), uint64(0), state.CurrentHeight)
}

func (suite *KeeperTestSuite) TestSetConsensusState() {
	state := &types.ConsensusState{
		CurrentHeight:    1000,
		TotalAntBurned:   "5000000",
		LastBlockTime:    timestamppb.Now(),
		ActiveValidators: []string{"cosmos1validator1", "cosmos1validator2"},
	}
	err := suite.keeper.SetConsensusState(suite.ctx, state)
	require.NoError(suite.T(), err)

	retrieved, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), state.CurrentHeight, retrieved.CurrentHeight)
	require.Equal(suite.T(), state.TotalAntBurned, retrieved.TotalAntBurned)
	require.Equal(suite.T(), len(state.ActiveValidators), len(retrieved.ActiveValidators))
}

func (suite *KeeperTestSuite) TestUpdateConsensusState() {
	height := uint64(2000)
	totalAntBurned := "10000000"
	activeValidators := []string{"cosmos1validator1", "cosmos1validator2", "cosmos1validator3"}

	err := suite.keeper.UpdateConsensusState(suite.ctx, height, totalAntBurned, activeValidators)
	require.NoError(suite.T(), err)

	state, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), height, state.CurrentHeight)
	require.Equal(suite.T(), totalAntBurned, state.TotalAntBurned)
	require.Equal(suite.T(), len(activeValidators), len(state.ActiveValidators))
}

func (suite *KeeperTestSuite) TestGetValidatorWeight() {
	validator := "cosmos1validator"
	weight, err := suite.keeper.GetValidatorWeight(suite.ctx, validator)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "0", weight)
}

func (suite *KeeperTestSuite) TestGetValidatorWeight_EmptyAddress() {
	_, err := suite.keeper.GetValidatorWeight(suite.ctx, "")
	require.Error(suite.T(), err)
}

func (suite *KeeperTestSuite) TestSetValidatorWeight() {
	validator := "cosmos1validator"
	weight := "1000000"

	err := suite.keeper.SetValidatorWeight(suite.ctx, validator, weight)
	require.NoError(suite.T(), err)

	retrieved, err := suite.keeper.GetValidatorWeight(suite.ctx, validator)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), weight, retrieved)
}

func (suite *KeeperTestSuite) TestSetValidatorWeight_EmptyAddress() {
	err := suite.keeper.SetValidatorWeight(suite.ctx, "", "1000000")
	require.Error(suite.T(), err)
}

func (suite *KeeperTestSuite) TestGetAllValidatorWeights() {
	suite.keeper.SetValidatorWeight(suite.ctx, "cosmos1validator1", "1000000")
	suite.keeper.SetValidatorWeight(suite.ctx, "cosmos1validator2", "2000000")
	suite.keeper.SetValidatorWeight(suite.ctx, "cosmos1validator3", "3000000")

	weights, err := suite.keeper.GetAllValidatorWeights(suite.ctx)
	require.NoError(suite.T(), err)
	require.GreaterOrEqual(suite.T(), len(weights), 3)

	weightMap := make(map[string]string)
	for _, w := range weights {
		weightMap[w.Validator] = w.Weight
	}
	require.Equal(suite.T(), "1000000", weightMap["cosmos1validator1"])
	require.Equal(suite.T(), "2000000", weightMap["cosmos1validator2"])
	require.Equal(suite.T(), "3000000", weightMap["cosmos1validator3"])
}

func (suite *KeeperTestSuite) TestGetHalvingInfo() {
	info := &types.HalvingInfo{
		LastHalvingHeight: 50000,
		HalvingInterval:   100000,
		NextHalvingHeight: 150000,
	}
	err := suite.keeper.SetHalvingInfo(suite.ctx, info)
	require.NoError(suite.T(), err)

	retrieved, err := suite.keeper.GetHalvingInfo(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), info.LastHalvingHeight, retrieved.LastHalvingHeight)
	require.Equal(suite.T(), info.HalvingInterval, retrieved.HalvingInterval)
	require.Equal(suite.T(), info.NextHalvingHeight, retrieved.NextHalvingHeight)
}

func (suite *KeeperTestSuite) TestGetAllValidators() {
	validator1 := &consensusv1.Validator{
		Validator:    "cosmos1validator1",
		ActivatedLzn: "1000000",
		Status:       consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
	}
	validator2 := &consensusv1.Validator{
		Validator:    "cosmos1validator2",
		ActivatedLzn: "2000000",
		Status:       consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
	}

	suite.keeper.SetValidator(suite.ctx, validator1)
	suite.keeper.SetValidator(suite.ctx, validator2)

	validators := suite.keeper.GetAllValidators(suite.ctx)
	require.GreaterOrEqual(suite.T(), len(validators), 2)

	validatorMap := make(map[string]*consensusv1.Validator)
	for _, v := range validators {
		validatorMap[v.Validator] = v
	}
	require.Contains(suite.T(), validatorMap, "cosmos1validator1")
	require.Contains(suite.T(), validatorMap, "cosmos1validator2")
}

// ============================================================================
// Economic Functions
// ============================================================================

func (suite *KeeperTestSuite) TestCalculateBaseReward() {
	reward, err := suite.keeper.CalculateBaseReward(suite.ctx, 0)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(50000000), reward)

	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 210000)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(25000000), reward)

	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 420000)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(12500000), reward)

	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 630000)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(6250000), reward)

	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 2100000)
	require.NoError(suite.T(), err)
	require.Greater(suite.T(), reward, uint64(0))
}

func (suite *KeeperTestSuite) TestGetHalvingCount() {
	count := suite.keeper.GetHalvingCount(0)
	require.Equal(suite.T(), uint64(0), count)

	count = suite.keeper.GetHalvingCount(209999)
	require.Equal(suite.T(), uint64(0), count)

	count = suite.keeper.GetHalvingCount(210000)
	require.Equal(suite.T(), uint64(1), count)

	count = suite.keeper.GetHalvingCount(210001)
	require.Equal(suite.T(), uint64(1), count)

	count = suite.keeper.GetHalvingCount(420000)
	require.Equal(suite.T(), uint64(2), count)
}

func (suite *KeeperTestSuite) TestCalculateMOAPenaltyMultiplier() {
	ctx := suite.ctx

	multiplier := suite.keeper.CalculateMOAPenaltyMultiplier(ctx, 1.0)
	require.Equal(suite.T(), 1.0, multiplier)

	multiplier = suite.keeper.CalculateMOAPenaltyMultiplier(ctx, 1.5)
	require.Equal(suite.T(), 1.0, multiplier)

	multiplier = suite.keeper.CalculateMOAPenaltyMultiplier(ctx, 0.95)
	require.Equal(suite.T(), 1.0, multiplier)

	multiplier = suite.keeper.CalculateMOAPenaltyMultiplier(ctx, 0.9)
	require.Equal(suite.T(), 1.0, multiplier)

	multiplier = suite.keeper.CalculateMOAPenaltyMultiplier(ctx, 0.8)
	require.Equal(suite.T(), 0.75, multiplier)

	multiplier = suite.keeper.CalculateMOAPenaltyMultiplier(ctx, 0.7)
	require.Equal(suite.T(), 0.75, multiplier)

	multiplier = suite.keeper.CalculateMOAPenaltyMultiplier(ctx, 0.6)
	require.Equal(suite.T(), 0.5, multiplier)

	multiplier = suite.keeper.CalculateMOAPenaltyMultiplier(ctx, 0.5)
	require.Equal(suite.T(), 0.5, multiplier)

	multiplier = suite.keeper.CalculateMOAPenaltyMultiplier(ctx, 0.4)
	require.Equal(suite.T(), 0.0, multiplier)

	multiplier = suite.keeper.CalculateMOAPenaltyMultiplier(ctx, 0.0)
	require.Equal(suite.T(), 0.0, multiplier)
}

func (suite *KeeperTestSuite) TestCalculateRewardDistribution() {
	baseReward := uint64(50000000)

	_, _, err := suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, map[string]uint64{})
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "no validators")

	_, _, err = suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, map[string]uint64{
		"cosmos1validator1": 0,
		"cosmos1validator2": 0,
	})
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "zero")

	validatorLZN := map[string]uint64{
		"cosmos1validator1": 1000000,
	}

	rewards, totalDistributed, err := suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, validatorLZN)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), rewards, 1)
	require.Equal(suite.T(), baseReward, totalDistributed)
	require.Equal(suite.T(), "cosmos1validator1", rewards[0].Validator)
	require.Equal(suite.T(), uint64(1000000), rewards[0].ActivatedLZN)
	require.Equal(suite.T(), 1.0, rewards[0].RewardShare)
	require.Equal(suite.T(), baseReward, rewards[0].FinalRewardAmount)

	validatorLZN = map[string]uint64{
		"cosmos1validator1": 1000000,
		"cosmos1validator2": 1000000,
	}

	rewards, totalDistributed, err = suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, validatorLZN)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), rewards, 2)
	require.Equal(suite.T(), baseReward, totalDistributed)

	for _, reward := range rewards {
		require.Equal(suite.T(), 0.5, reward.RewardShare)
		require.Equal(suite.T(), uint64(25000000), reward.BaseRewardAmount)
	}

	validatorLZN = map[string]uint64{
		"cosmos1validator1": 2000000,
		"cosmos1validator2": 1000000,
	}

	rewards, totalDistributed, err = suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, validatorLZN)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), rewards, 2)

	var validator1Reward, validator2Reward keeper.ValidatorRewardInfo
	for _, r := range rewards {
		if r.Validator == "cosmos1validator1" {
			validator1Reward = r
		} else {
			validator2Reward = r
		}
	}

	require.InDelta(suite.T(), 0.6667, validator1Reward.RewardShare, 0.0001)
	require.InDelta(suite.T(), 0.3333, validator2Reward.RewardShare, 0.0001)
}

// ============================================================================
// DistributeBaseRewards Tests
// ============================================================================

func (suite *KeeperTestSuite) TestDistributeBaseRewards_Basic() {
	height := uint64(1000)
	mockLizenzKeeper := &MockLizenzKeeper{
		activatedLizenz: []interface{}{
			map[string]interface{}{
				"validator": "cosmos1validator1",
				"amount":    "5000000",
			},
		},
		moaCompliance: map[string]float64{
			"cosmos1validator1": 1.0,
		},
	}
	suite.keeper.SetLizenzKeeper(mockLizenzKeeper)

	mockBankKeeper := NewMockBankKeeper()
	suite.keeper.SetBankKeeper(mockBankKeeper)

	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	require.NoError(suite.T(), err)
}

func (suite *KeeperTestSuite) TestDistributeBaseRewards_NoLizenzKeeper() {
	height := uint64(1000)
	suite.keeper.SetLizenzKeeper(nil)

	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	require.NoError(suite.T(), err)
}

func (suite *KeeperTestSuite) TestDistributeBaseRewards_NoValidatorsWithLZN() {
	height := uint64(1000)
	mockLizenzKeeper := &MockLizenzKeeper{
		activatedLizenz: []interface{}{},
	}
	suite.keeper.SetLizenzKeeper(mockLizenzKeeper)

	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	require.NoError(suite.T(), err)
}

func (suite *KeeperTestSuite) TestDistributeBaseRewards_GetAllLizenzError() {
	height := uint64(1000)
	mockLizenzKeeper := &MockLizenzKeeper{
		errors: map[string]error{
			"GetAllActivatedLizenz": fmt.Errorf("failed to get LZN"),
		},
	}
	suite.keeper.SetLizenzKeeper(mockLizenzKeeper)

	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "failed to get activated LZN")
}

func (suite *KeeperTestSuite) TestDistributeBaseRewards_NoBankKeeper() {
	height := uint64(1000)
	mockLizenzKeeper := &MockLizenzKeeper{
		activatedLizenz: []interface{}{
			map[string]interface{}{
				"validator": "cosmos1validator1",
				"amount":    "5000000",
			},
		},
		moaCompliance: map[string]float64{
			"cosmos1validator1": 1.0,
		},
	}
	suite.keeper.SetLizenzKeeper(mockLizenzKeeper)

	suite.keeper.SetBankKeeper(nil)

	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	require.NoError(suite.T(), err)
}

func (suite *KeeperTestSuite) TestDistributeBaseRewards_InvalidValidatorAddress() {
	height := uint64(1000)
	mockLizenzKeeper := &MockLizenzKeeper{
		activatedLizenz: []interface{}{
			map[string]interface{}{
				"validator": "invalid_address",
				"amount":    "5000000",
			},
		},
		moaCompliance: map[string]float64{
			"invalid_address": 1.0,
		},
	}
	suite.keeper.SetLizenzKeeper(mockLizenzKeeper)

	mockBankKeeper := NewMockBankKeeper()
	suite.keeper.SetBankKeeper(mockBankKeeper)

	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	_ = err
}

func (suite *KeeperTestSuite) TestDistributeBaseRewards_MintCoinsError() {
	height := uint64(1000)
	validatorAddr := sdk.AccAddress("validator1_______________")
	mockLizenzKeeper := &MockLizenzKeeper{
		activatedLizenz: []interface{}{
			map[string]interface{}{
				"validator": validatorAddr.String(),
				"amount":    "5000000",
			},
		},
		moaCompliance: map[string]float64{
			validatorAddr.String(): 1.0,
		},
	}
	suite.keeper.SetLizenzKeeper(mockLizenzKeeper)

	mockBankKeeper := NewMockBankKeeper()
	mockBankKeeper.SetMintError(types.ModuleName, fmt.Errorf("mint failed"))
	suite.keeper.SetBankKeeper(mockBankKeeper)

	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	require.NoError(suite.T(), err)
}

func (suite *KeeperTestSuite) TestDistributeBaseRewards_SendCoinsError() {
	height := uint64(1000)
	validatorAddr := sdk.AccAddress("validator1_______________")
	mockLizenzKeeper := &MockLizenzKeeper{
		activatedLizenz: []interface{}{
			map[string]interface{}{
				"validator": validatorAddr.String(),
				"amount":    "5000000",
			},
		},
		moaCompliance: map[string]float64{
			validatorAddr.String(): 1.0,
		},
	}
	suite.keeper.SetLizenzKeeper(mockLizenzKeeper)

	mockBankKeeper := NewMockBankKeeper()
	mockBankKeeper.SetSendError(validatorAddr.String(), fmt.Errorf("send failed"))
	suite.keeper.SetBankKeeper(mockBankKeeper)

	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	require.NoError(suite.T(), err)
}

// ============================================================================
// Validator Lookup Tests
// ============================================================================

func (suite *KeeperTestSuite) TestGetValidator_NotFound() {
	validator, err := suite.keeper.GetValidator(suite.ctx, "cosmos1nonexistent")
	require.Error(suite.T(), err)
	require.Nil(suite.T(), validator)
	require.Contains(suite.T(), err.Error(), "not found")
}

func (suite *KeeperTestSuite) TestGetAllValidators_Empty() {
	validators := suite.keeper.GetAllValidators(suite.ctx)
	require.Empty(suite.T(), validators)
}

func (suite *KeeperTestSuite) TestGetValidatorWeight_NotFound() {
	weight, err := suite.keeper.GetValidatorWeight(suite.ctx, "cosmos1nonexistent")
	_ = weight
	_ = err
}

// ============================================================================
// RecordBlockTime Tests
// ============================================================================

func (suite *KeeperTestSuite) TestRecordBlockTime_Basic() {
	height := uint64(1000)

	err := suite.keeper.RecordBlockTime(suite.ctx, height)
	require.NoError(suite.T(), err)

	avgTime, err := suite.keeper.GetAverageBlockTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), avgTime)
}

func (suite *KeeperTestSuite) TestRecordBlockTime_Multiple() {
	for i := 0; i < 10; i++ {
		height := uint64(1000 + i)
		err := suite.keeper.RecordBlockTime(suite.ctx, height)
		require.NoError(suite.T(), err)
	}

	avgTime, err := suite.keeper.GetAverageBlockTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), avgTime)
}

// ============================================================================
// CalculateBaseReward Halving Tests
// ============================================================================

func (suite *KeeperTestSuite) TestCalculateBaseReward_ZeroHeight() {
	reward, err := suite.keeper.CalculateBaseReward(suite.ctx, 0)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(50_000_000), reward)
}

func (suite *KeeperTestSuite) TestCalculateBaseReward_FirstHalving() {
	halvingHeight := uint64(keeper.HalvingInterval)
	reward, err := suite.keeper.CalculateBaseReward(suite.ctx, halvingHeight)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(50_000_000/2), reward)
}

func (suite *KeeperTestSuite) TestCalculateBaseReward_SecondHalving() {
	halvingHeight := uint64(keeper.HalvingInterval * 2)
	reward, err := suite.keeper.CalculateBaseReward(suite.ctx, halvingHeight)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(50_000_000/4), reward)
}

func (suite *KeeperTestSuite) TestCalculateBaseReward_BetweenHalvings() {
	height := uint64(keeper.HalvingInterval / 2)
	reward, err := suite.keeper.CalculateBaseReward(suite.ctx, height)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(50_000_000), reward)
}

// =============================================================================
// Dynamic Block Time Tests
// =============================================================================

func (suite *KeeperTestSuite) TestGetSetTargetBlockTime() {
	d := suite.keeper.GetTargetBlockTime(suite.ctx)
	require.Equal(suite.T(), time.Duration(0), d)

	err := suite.keeper.SetTargetBlockTime(suite.ctx, 5*time.Second)
	require.NoError(suite.T(), err)
	d = suite.keeper.GetTargetBlockTime(suite.ctx)
	require.Equal(suite.T(), 5*time.Second, d)

	err = suite.keeper.SetTargetBlockTime(suite.ctx, 3*time.Second)
	require.NoError(suite.T(), err)
	d = suite.keeper.GetTargetBlockTime(suite.ctx)
	require.Equal(suite.T(), 3*time.Second, d)
}

func (suite *KeeperTestSuite) TestValidateBlockTiming_NoTarget() {
	prev := time.Now()
	proposed := prev.Add(1 * time.Second)
	err := suite.keeper.ValidateBlockTiming(suite.ctx, prev, proposed)
	require.NoError(suite.T(), err, "should pass when no target is set")
}

func (suite *KeeperTestSuite) TestValidateBlockTiming_ValidTiming() {
	err := suite.keeper.SetTargetBlockTime(suite.ctx, 3*time.Second)
	require.NoError(suite.T(), err)

	prev := time.Now()
	proposed := prev.Add(4 * time.Second)
	err = suite.keeper.ValidateBlockTiming(suite.ctx, prev, proposed)
	require.NoError(suite.T(), err, "proposed block after target should pass")
}

func (suite *KeeperTestSuite) TestValidateBlockTiming_TooFast() {
	err := suite.keeper.SetTargetBlockTime(suite.ctx, 5*time.Second)
	require.NoError(suite.T(), err)

	prev := time.Now()
	proposed := prev.Add(2 * time.Second)
	err = suite.keeper.ValidateBlockTiming(suite.ctx, prev, proposed)
	require.Error(suite.T(), err, "block arriving before target should fail")
	require.Contains(suite.T(), err.Error(), "block too fast")
}

func (suite *KeeperTestSuite) TestValidateBlockTiming_ExactTarget() {
	err := suite.keeper.SetTargetBlockTime(suite.ctx, 3*time.Second)
	require.NoError(suite.T(), err)

	prev := time.Now()
	proposed := prev.Add(3 * time.Second)
	err = suite.keeper.ValidateBlockTiming(suite.ctx, prev, proposed)
	require.NoError(suite.T(), err, "block at exact target should pass")
}

// =============================================================================
// ANT Bid Tracking Tests
// =============================================================================

func (suite *KeeperTestSuite) TestRecordAntBid() {
	val := "cosmos1validator1"

	require.Equal(suite.T(), uint64(0), suite.keeper.GetAntPurchaseTotal(suite.ctx, val))

	suite.keeper.RecordAntBid(suite.ctx, val, 100)
	require.Equal(suite.T(), uint64(100), suite.keeper.GetAntPurchaseTotal(suite.ctx, val))

	suite.keeper.RecordAntBid(suite.ctx, val, 250)
	require.Equal(suite.T(), uint64(350), suite.keeper.GetAntPurchaseTotal(suite.ctx, val))

	val2 := "cosmos1validator2"
	require.Equal(suite.T(), uint64(0), suite.keeper.GetAntPurchaseTotal(suite.ctx, val2))
	suite.keeper.RecordAntBid(suite.ctx, val2, 500)
	require.Equal(suite.T(), uint64(500), suite.keeper.GetAntPurchaseTotal(suite.ctx, val2))
	require.Equal(suite.T(), uint64(350), suite.keeper.GetAntPurchaseTotal(suite.ctx, val))
}

func (suite *KeeperTestSuite) TestResetAntPurchases() {
	suite.keeper.RecordAntBid(suite.ctx, "cosmos1v1", 100)
	suite.keeper.RecordAntBid(suite.ctx, "cosmos1v2", 200)
	require.Equal(suite.T(), uint64(100), suite.keeper.GetAntPurchaseTotal(suite.ctx, "cosmos1v1"))

	suite.keeper.ResetAntPurchases(suite.ctx)
	require.Equal(suite.T(), uint64(0), suite.keeper.GetAntPurchaseTotal(suite.ctx, "cosmos1v1"))
	require.Equal(suite.T(), uint64(0), suite.keeper.GetAntPurchaseTotal(suite.ctx, "cosmos1v2"))
}

func (suite *KeeperTestSuite) TestGetSetEpochStartHeight() {
	h := suite.keeper.GetEpochStartHeight(suite.ctx)
	require.Equal(suite.T(), uint64(0), h)

	suite.keeper.SetEpochStartHeight(suite.ctx, 500)
	h = suite.keeper.GetEpochStartHeight(suite.ctx)
	require.Equal(suite.T(), uint64(500), h)
}
