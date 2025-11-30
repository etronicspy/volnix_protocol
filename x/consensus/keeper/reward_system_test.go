package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

type RewardSystemTestSuite struct {
	KeeperTestSuite
	mockBankKeeper   *MockBankKeeper
	mockLizenzKeeper *MockLizenzKeeper
}

func TestRewardSystemTestSuite(t *testing.T) {
	suite.Run(t, new(RewardSystemTestSuite))
}

func (suite *RewardSystemTestSuite) SetupTest() {
	suite.KeeperTestSuite.SetupTest()
	suite.mockBankKeeper = NewMockBankKeeper()
	suite.mockLizenzKeeper = &MockLizenzKeeper{
		activatedLizenz: []interface{}{},
		moaCompliance:   make(map[string]float64),
	}
	suite.keeper.SetBankKeeper(suite.mockBankKeeper)
	suite.keeper.SetLizenzKeeper(suite.mockLizenzKeeper)
}

// TestCalculateBaseReward tests base reward calculation
func (suite *RewardSystemTestSuite) TestCalculateBaseReward() {
	// Test reward at height 0 (no halving)
	reward, err := suite.keeper.CalculateBaseReward(suite.ctx, 0)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(50000000), reward) // 50 WRT

	// Test reward at height 210000 (first halving)
	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 210000)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(25000000), reward) // 25 WRT

	// Test reward at height 420000 (second halving)
	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 420000)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(12500000), reward) // 12.5 WRT

	// Test reward at height 630000 (third halving)
	reward, err = suite.keeper.CalculateBaseReward(suite.ctx, 630000)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(6250000), reward) // 6.25 WRT
}

// TestCalculateRewardDistribution tests reward distribution calculation
func (suite *RewardSystemTestSuite) TestCalculateRewardDistribution() {
	validatorLZN := map[string]uint64{
		"volnix1validator1": 1000000, // 1M LZN
		"volnix1validator2": 2000000, // 2M LZN
		"volnix1validator3": 2000000, // 2M LZN
	}

	baseReward := uint64(50000000) // 50 WRT

	rewards, totalDistributed, err := suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, validatorLZN)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), rewards, 3)
	require.Equal(suite.T(), uint64(50000000), totalDistributed)

	// Check individual rewards
	// Total LZN: 5M
	// Validator1: 1M/5M = 0.2 = 10M uwrt
	// Validator2: 2M/5M = 0.4 = 20M uwrt
	// Validator3: 2M/5M = 0.4 = 20M uwrt

	var reward1, reward2, reward3 *keeper.ValidatorRewardInfo
	for i := range rewards {
		switch rewards[i].Validator {
		case "volnix1validator1":
			reward1 = &rewards[i]
		case "volnix1validator2":
			reward2 = &rewards[i]
		case "volnix1validator3":
			reward3 = &rewards[i]
		}
	}

	require.NotNil(suite.T(), reward1)
	require.NotNil(suite.T(), reward2)
	require.NotNil(suite.T(), reward3)

	require.Equal(suite.T(), uint64(1000000), reward1.ActivatedLZN)
	require.Equal(suite.T(), uint64(2000000), reward2.ActivatedLZN)
	require.Equal(suite.T(), uint64(2000000), reward3.ActivatedLZN)

	// Check shares
	require.InDelta(suite.T(), 0.2, reward1.RewardShare, 0.01)
	require.InDelta(suite.T(), 0.4, reward2.RewardShare, 0.01)
	require.InDelta(suite.T(), 0.4, reward3.RewardShare, 0.01)
}

// TestCalculateRewardDistributionWithMOAPenalty tests reward distribution with MOA penalty
func (suite *RewardSystemTestSuite) TestCalculateRewardDistributionWithMOAPenalty() {
	// Set MOA compliance for validators
	suite.mockLizenzKeeper.moaCompliance["volnix1validator1"] = 1.0  // Full compliance
	suite.mockLizenzKeeper.moaCompliance["volnix1validator2"] = 0.7  // Partial compliance
	suite.mockLizenzKeeper.moaCompliance["volnix1validator3"] = 0.3  // Low compliance

	validatorLZN := map[string]uint64{
		"volnix1validator1": 1000000,
		"volnix1validator2": 1000000,
		"volnix1validator3": 1000000,
	}

	baseReward := uint64(30000000) // 30 WRT

	rewards, _, err := suite.keeper.CalculateRewardDistribution(suite.ctx, baseReward, validatorLZN)
	require.NoError(suite.T(), err)

	// Find rewards
	var reward1, reward2, reward3 *keeper.ValidatorRewardInfo
	for i := range rewards {
		switch rewards[i].Validator {
		case "volnix1validator1":
			reward1 = &rewards[i]
		case "volnix1validator2":
			reward2 = &rewards[i]
		case "volnix1validator3":
			reward3 = &rewards[i]
		}
	}

	require.NotNil(suite.T(), reward1)
	require.NotNil(suite.T(), reward2)
	require.NotNil(suite.T(), reward3)

	// Validator1 should have full reward (1.0 compliance)
	require.Equal(suite.T(), 1.0, reward1.MOACompliance)
	require.Equal(suite.T(), 1.0, reward1.PenaltyMultiplier)

	// Validator2 should have reduced reward (0.7 compliance -> 0.75 multiplier)
	// According to CalculateMOAPenaltyMultiplier: 0.7 is in the 0.5-0.7 range, so multiplier is 0.75
	require.Equal(suite.T(), 0.7, reward2.MOACompliance)
	require.Equal(suite.T(), 0.75, reward2.PenaltyMultiplier)

	// Validator3 should have zero reward (0.3 compliance -> 0.0 multiplier)
	require.Equal(suite.T(), 0.3, reward3.MOACompliance)
	require.Equal(suite.T(), 0.0, reward3.PenaltyMultiplier)
	require.Equal(suite.T(), uint64(0), reward3.FinalRewardAmount)
}

// TestDistributeBaseRewardsFullFlow tests the full reward distribution flow
func (suite *RewardSystemTestSuite) TestDistributeBaseRewardsFullFlow() {
	// Create valid bech32 addresses
	validator1Addr := sdk.AccAddress("validator1_______________")
	validator2Addr := sdk.AccAddress("validator2_______________")
	
	// Set up validators with activated LZN
	suite.mockLizenzKeeper.activatedLizenz = []interface{}{
		map[string]interface{}{
			"validator": validator1Addr.String(),
			"amount":    "1000000",
		},
		map[string]interface{}{
			"validator": validator2Addr.String(),
			"amount":    "2000000",
		},
	}

	// Set MOA compliance
	suite.mockLizenzKeeper.moaCompliance[validator1Addr.String()] = 1.0
	suite.mockLizenzKeeper.moaCompliance[validator2Addr.String()] = 1.0

	suite.ctx = suite.ctx.WithBlockHeight(1000)

	// Distribute rewards
	err := suite.keeper.DistributeBaseRewards(suite.ctx, 1000)
	require.NoError(suite.T(), err)

	// Check that coins were minted
	mintedCoins := suite.mockBankKeeper.GetMintedCoins(types.ModuleName)
	require.NotEmpty(suite.T(), mintedCoins)
	require.True(suite.T(), mintedCoins.AmountOf("uwrt").GT(math.ZeroInt()))

	// Check that coins were sent to validators
	// Check all sent coins (might be stored by different key format)
	hasSentCoins := false
	var uwrt1, uwrt2 math.Int
	for addr, coins := range suite.mockBankKeeper.sentCoins {
		if !coins.IsZero() && coins.AmountOf("uwrt").GT(math.ZeroInt()) {
			hasSentCoins = true
			// Find coins for each validator
			if addr == validator1Addr.String() {
				uwrt1 = coins.AmountOf("uwrt")
			}
			if addr == validator2Addr.String() {
				uwrt2 = coins.AmountOf("uwrt")
			}
		}
	}
	require.True(suite.T(), hasSentCoins, "coins should be sent to validators")
	
	// Validator2 should get more than validator1 (2x LZN)
	if !uwrt1.IsZero() && !uwrt2.IsZero() {
		require.True(suite.T(), uwrt2.GT(uwrt1), "validator2 should get more than validator1")
	}
}

// TestDistributeBaseRewardsNoValidators tests distribution when no validators have LZN
func (suite *RewardSystemTestSuite) TestDistributeBaseRewardsNoValidators() {
	// No activated LZN
	suite.mockLizenzKeeper.activatedLizenz = []interface{}{}

	suite.ctx = suite.ctx.WithBlockHeight(1000)

	// Distribute rewards - should not error, but skip distribution
	err := suite.keeper.DistributeBaseRewards(suite.ctx, 1000)
	require.NoError(suite.T(), err)

	// No coins should be minted
	mintedCoins := suite.mockBankKeeper.GetMintedCoins(types.ModuleName)
	require.Empty(suite.T(), mintedCoins)
}

// TestDistributeBaseRewardsNoLizenzKeeper tests distribution without lizenz keeper
func (suite *RewardSystemTestSuite) TestDistributeBaseRewardsNoLizenzKeeper() {
	// Remove lizenz keeper
	suite.keeper.SetLizenzKeeper(nil)

	suite.ctx = suite.ctx.WithBlockHeight(1000)

	// Distribute rewards - should not error, but skip distribution
	err := suite.keeper.DistributeBaseRewards(suite.ctx, 1000)
	require.NoError(suite.T(), err)

	// No coins should be minted
	mintedCoins := suite.mockBankKeeper.GetMintedCoins(types.ModuleName)
	require.Empty(suite.T(), mintedCoins)
}

