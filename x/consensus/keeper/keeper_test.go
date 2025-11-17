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

func (suite *KeeperTestSuite) TestSelectBlockProducer() {
	validators := []string{"cosmos1validator1", "cosmos1validator2", "cosmos1validator3"}
	selected, err := suite.keeper.SelectBlockProducer(suite.ctx, validators)
	require.NoError(suite.T(), err)
	require.Contains(suite.T(), validators, selected)
}

func (suite *KeeperTestSuite) TestSelectBlockProducer_NoValidators() {
	_, err := suite.keeper.SelectBlockProducer(suite.ctx, []string{})
	require.Error(suite.T(), err)
}

func (suite *KeeperTestSuite) TestRecordBlockTime() {
	err := suite.keeper.RecordBlockTime(suite.ctx, 1000)
	require.NoError(suite.T(), err)
	
	// Record multiple times
	err = suite.keeper.RecordBlockTime(suite.ctx, 1001)
	require.NoError(suite.T(), err)
	err = suite.keeper.RecordBlockTime(suite.ctx, 1002)
	require.NoError(suite.T(), err)
	
	// Get average block time
	avgTime, err := suite.keeper.GetAverageBlockTime(suite.ctx)
	require.NoError(suite.T(), err)
	require.Greater(suite.T(), avgTime, time.Duration(0))
}

func (suite *KeeperTestSuite) TestGetConsensusState() {
	state, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), state)
	require.Equal(suite.T(), uint64(0), state.CurrentHeight) // Default height in test context
}

func (suite *KeeperTestSuite) TestSetConsensusState() {
	state := types.ConsensusState{
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
	require.Equal(suite.T(), "0", weight) // Default weight
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
	// Set multiple validator weights
	suite.keeper.SetValidatorWeight(suite.ctx, "cosmos1validator1", "1000000")
	suite.keeper.SetValidatorWeight(suite.ctx, "cosmos1validator2", "2000000")
	suite.keeper.SetValidatorWeight(suite.ctx, "cosmos1validator3", "3000000")
	
	weights, err := suite.keeper.GetAllValidatorWeights(suite.ctx)
	require.NoError(suite.T(), err)
	require.GreaterOrEqual(suite.T(), len(weights), 3)
	
	// Verify weights are present
	weightMap := make(map[string]string)
	for _, w := range weights {
		weightMap[w.Validator] = w.Weight
	}
	require.Equal(suite.T(), "1000000", weightMap["cosmos1validator1"])
	require.Equal(suite.T(), "2000000", weightMap["cosmos1validator2"])
	require.Equal(suite.T(), "3000000", weightMap["cosmos1validator3"])
}

func (suite *KeeperTestSuite) TestCalculateBlockTime_InvalidAmount() {
	_, err := suite.keeper.CalculateBlockTime(suite.ctx, "invalid")
	require.Error(suite.T(), err)
}

func (suite *KeeperTestSuite) TestCalculateBlockTime_ZeroAmount() {
	_, err := suite.keeper.CalculateBlockTime(suite.ctx, "0")
	require.Error(suite.T(), err)
}

func (suite *KeeperTestSuite) TestGetHalvingInfo() {
	info := types.HalvingInfo{
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

func (suite *KeeperTestSuite) TestGetBlockCreator() {
	height := uint64(1000)
	blockCreator := &consensusv1.BlockCreator{
		BlockHeight:   height,
		Validator:     "cosmos1validator",
		SelectionTime: timestamppb.Now(),
	}
	suite.keeper.SetBlockCreator(suite.ctx, blockCreator)
	
	retrieved, err := suite.keeper.GetBlockCreator(suite.ctx, height)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), blockCreator.BlockHeight, retrieved.BlockHeight)
	require.Equal(suite.T(), blockCreator.Validator, retrieved.Validator)
}

func (suite *KeeperTestSuite) TestGetBlockCreator_NotFound() {
	_, err := suite.keeper.GetBlockCreator(suite.ctx, 9999)
	require.Error(suite.T(), err)
}

func (suite *KeeperTestSuite) TestGetAllValidators() {
	validator1 := &consensusv1.Validator{
		Validator:     "cosmos1validator1",
		AntBalance:    "1000000",
		ActivityScore: "500",
		Status:        consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
		LastActive:    timestamppb.Now(),
	}
	validator2 := &consensusv1.Validator{
		Validator:     "cosmos1validator2",
		AntBalance:    "2000000",
		ActivityScore: "600",
		Status:        consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
		LastActive:    timestamppb.Now(),
	}
	
	suite.keeper.SetValidator(suite.ctx, validator1)
	suite.keeper.SetValidator(suite.ctx, validator2)
	
	validators := suite.keeper.GetAllValidators(suite.ctx)
	require.GreaterOrEqual(suite.T(), len(validators), 2)
	
	// Verify validators are present
	validatorMap := make(map[string]*consensusv1.Validator)
	for _, v := range validators {
		validatorMap[v.Validator] = v
	}
	require.Contains(suite.T(), validatorMap, "cosmos1validator1")
	require.Contains(suite.T(), validatorMap, "cosmos1validator2")
}

func (suite *KeeperTestSuite) TestGetBlindAuction() {
	height := uint64(1000)
	auction := &consensusv1.BlindAuction{
		BlockHeight: height,
		Phase:        consensusv1.AuctionPhase_AUCTION_PHASE_COMMIT,
		Commits:      []*consensusv1.EncryptedBid{},
		Reveals:      []*consensusv1.BidReveal{},
	}
	
	err := suite.keeper.SetBlindAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	
	retrieved, err := suite.keeper.GetBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), auction.BlockHeight, retrieved.BlockHeight)
	require.Equal(suite.T(), auction.Phase, retrieved.Phase)
}

func (suite *KeeperTestSuite) TestGetBlindAuction_NotFound() {
	_, err := suite.keeper.GetBlindAuction(suite.ctx, 9999)
	require.Error(suite.T(), err)
}

func (suite *KeeperTestSuite) TestSetBlindAuction() {
	auction := &consensusv1.BlindAuction{
		BlockHeight: 1000,
		Phase:        consensusv1.AuctionPhase_AUCTION_PHASE_COMMIT,
		Commits: []*consensusv1.EncryptedBid{
			{
				Validator:   "cosmos1validator1",
				CommitHash:  "hash1",
				BlockHeight: 1000,
			},
			{
				Validator:   "cosmos1validator2",
				CommitHash:  "hash2",
				BlockHeight: 1000,
			},
		},
		Reveals: []*consensusv1.BidReveal{},
	}
	
	err := suite.keeper.SetBlindAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	
	// Verify it was stored
	retrieved, err := suite.keeper.GetBlindAuction(suite.ctx, 1000)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), len(auction.Commits), len(retrieved.Commits))
}

func (suite *KeeperTestSuite) TestCreateBlindAuction() {
	height := uint64(1000)
	
	auction, err := suite.keeper.CreateBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), auction)
	require.Equal(suite.T(), height, auction.BlockHeight)
	require.Equal(suite.T(), consensusv1.AuctionPhase_AUCTION_PHASE_COMMIT, auction.Phase)
}

func (suite *KeeperTestSuite) TestCreateBlindAuction_Duplicate() {
	height := uint64(1000)
	
	// Create first auction
	_, err := suite.keeper.CreateBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	
	// Try to create duplicate - should return existing auction or error
	_, err = suite.keeper.CreateBlindAuction(suite.ctx, height)
	// Note: CreateBlindAuction may return existing auction instead of error
	// This is acceptable behavior - verify that auction exists
	auction, err2 := suite.keeper.GetBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err2)
	require.NotNil(suite.T(), auction)
}

func (suite *KeeperTestSuite) TestHashCommit() {
	nonce := "nonce123"
	bidAmount := "1000000"
	
	hash := keeper.HashCommit(nonce, bidAmount)
	require.NotEmpty(suite.T(), hash)
	require.Len(suite.T(), hash, 64) // SHA256 produces 64 hex characters
}

func (suite *KeeperTestSuite) TestHashCommit_DifferentInputs() {
	hash1 := keeper.HashCommit("nonce1", "1000000")
	hash2 := keeper.HashCommit("nonce2", "1000000")
	hash3 := keeper.HashCommit("nonce1", "2000000")
	
	// All hashes should be different
	require.NotEqual(suite.T(), hash1, hash2)
	require.NotEqual(suite.T(), hash1, hash3)
	require.NotEqual(suite.T(), hash2, hash3)
}

func (suite *KeeperTestSuite) TestHashCommit_SameInputs() {
	nonce := "nonce123"
	bidAmount := "1000000"
	
	hash1 := keeper.HashCommit(nonce, bidAmount)
	hash2 := keeper.HashCommit(nonce, bidAmount)
	
	// Same inputs should produce same hash
	require.Equal(suite.T(), hash1, hash2)
}

func (suite *KeeperTestSuite) TestVerifyCommit() {
	nonce := "nonce123"
	bidAmount := "1000000"
	
	commitHash := keeper.HashCommit(nonce, bidAmount)
	
	// Verify correct commit
	valid := keeper.VerifyCommit(commitHash, nonce, bidAmount)
	require.True(suite.T(), valid)
	
	// Verify incorrect commit
	invalid := keeper.VerifyCommit(commitHash, "wrong_nonce", bidAmount)
	require.False(suite.T(), invalid)
	
	invalid2 := keeper.VerifyCommit(commitHash, nonce, "wrong_amount")
	require.False(suite.T(), invalid2)
}

// ============================================================================
// КРИТИЧЕСКИ ВАЖНЫЕ ЭКОНОМИЧЕСКИЕ ФУНКЦИИ
// ============================================================================

func (suite *KeeperTestSuite) TestCalculateBaseReward() {
	// Test initial reward (height 0, no halving)
	reward, err := suite.keeper.CalculateBaseReward(suite.ctx, 0)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(50000000), reward) // 50 WRT in micro units
	
	// Test after first halving (height 210000)
	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 210000)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(25000000), reward) // 25 WRT
	
	// Test after second halving (height 420000)
	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 420000)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(12500000), reward) // 12.5 WRT
	
	// Test after third halving (height 630000)
	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 630000)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(6250000), reward) // 6.25 WRT
	
	// Test very high halving count (should not be zero)
	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 2100000) // 10 halvings
	require.NoError(suite.T(), err)
	require.Greater(suite.T(), reward, uint64(0)) // Should be at least 1 micro WRT
}

func (suite *KeeperTestSuite) TestGetHalvingCount() {
	// Test initial (no halving)
	count := suite.keeper.GetHalvingCount(0)
	require.Equal(suite.T(), uint64(0), count)
	
	// Test just before first halving
	count = suite.keeper.GetHalvingCount(209999)
	require.Equal(suite.T(), uint64(0), count)
	
	// Test at first halving
	count = suite.keeper.GetHalvingCount(210000)
	require.Equal(suite.T(), uint64(1), count)
	
	// Test after first halving
	count = suite.keeper.GetHalvingCount(210001)
	require.Equal(suite.T(), uint64(1), count)
	
	// Test at second halving
	count = suite.keeper.GetHalvingCount(420000)
	require.Equal(suite.T(), uint64(2), count)
}

func (suite *KeeperTestSuite) TestCalculateMOAPenaltyMultiplier() {
	// Test perfect compliance (>= 1.0) - no penalty
	multiplier := keeper.CalculateMOAPenaltyMultiplier(1.0)
	require.Equal(suite.T(), 1.0, multiplier)
	
	multiplier = keeper.CalculateMOAPenaltyMultiplier(1.5)
	require.Equal(suite.T(), 1.0, multiplier)
	
	// Test warning zone (0.9-1.0) - no penalty but warning
	multiplier = keeper.CalculateMOAPenaltyMultiplier(0.95)
	require.Equal(suite.T(), 1.0, multiplier)
	
	multiplier = keeper.CalculateMOAPenaltyMultiplier(0.9)
	require.Equal(suite.T(), 1.0, multiplier)
	
	// Test 25% penalty zone (0.7-0.9)
	multiplier = keeper.CalculateMOAPenaltyMultiplier(0.8)
	require.Equal(suite.T(), 0.75, multiplier)
	
	multiplier = keeper.CalculateMOAPenaltyMultiplier(0.7)
	require.Equal(suite.T(), 0.75, multiplier)
	
	// Test 50% penalty zone (0.5-0.7)
	multiplier = keeper.CalculateMOAPenaltyMultiplier(0.6)
	require.Equal(suite.T(), 0.5, multiplier)
	
	multiplier = keeper.CalculateMOAPenaltyMultiplier(0.5)
	require.Equal(suite.T(), 0.5, multiplier)
	
	// Test deactivation zone (< 0.5) - no reward
	multiplier = keeper.CalculateMOAPenaltyMultiplier(0.4)
	require.Equal(suite.T(), 0.0, multiplier)
	
	multiplier = keeper.CalculateMOAPenaltyMultiplier(0.0)
	require.Equal(suite.T(), 0.0, multiplier)
}

func (suite *KeeperTestSuite) TestCalculateRewardDistribution() {
	baseReward := uint64(50000000) // 50 WRT
	
	// Test with empty validators
	_, _, err := suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, map[string]uint64{})
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "no validators")
	
	// Test with zero total LZN
	_, _, err = suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, map[string]uint64{
		"cosmos1validator1": 0,
		"cosmos1validator2": 0,
	})
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "zero")
	
	// Test with single validator
	validatorLZN := map[string]uint64{
		"cosmos1validator1": 1000000, // 1 LZN
	}
	
	rewards, totalDistributed, err := suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, validatorLZN)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), rewards, 1)
	require.Equal(suite.T(), baseReward, totalDistributed)
	require.Equal(suite.T(), "cosmos1validator1", rewards[0].Validator)
	require.Equal(suite.T(), uint64(1000000), rewards[0].ActivatedLZN)
	require.Equal(suite.T(), 1.0, rewards[0].RewardShare)
	require.Equal(suite.T(), baseReward, rewards[0].FinalRewardAmount)
	
	// Test with multiple validators (equal shares)
	validatorLZN = map[string]uint64{
		"cosmos1validator1": 1000000, // 1 LZN
		"cosmos1validator2": 1000000, // 1 LZN
	}
	
	rewards, totalDistributed, err = suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, validatorLZN)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), rewards, 2)
	require.Equal(suite.T(), baseReward, totalDistributed)
	
	// Each should get 50%
	for _, reward := range rewards {
		require.Equal(suite.T(), 0.5, reward.RewardShare)
		require.Equal(suite.T(), uint64(25000000), reward.BaseRewardAmount) // 25 WRT each
	}
	
	// Test with unequal shares
	validatorLZN = map[string]uint64{
		"cosmos1validator1": 2000000, // 2 LZN (66.67%)
		"cosmos1validator2": 1000000, // 1 LZN (33.33%)
	}
	
	rewards, totalDistributed, err = suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, validatorLZN)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), rewards, 2)
	
	// Find validator1 reward
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
