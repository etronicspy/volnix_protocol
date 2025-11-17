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

type RewardTrackerTestSuite struct {
	suite.Suite

	cdc        codec.Codec
	ctx        sdk.Context
	keeper     *Keeper
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
}

func (suite *RewardTrackerTestSuite) SetupTest() {
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
	// Set default params
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func TestRewardTrackerTestSuite(t *testing.T) {
	suite.Run(t, new(RewardTrackerTestSuite))
}

func (suite *RewardTrackerTestSuite) TestUpdateRewardStats() {
	validator := "cosmos1validator"
	
	// Create activated lizenz
	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:         validator,
		Amount:            "1000000",
		ActivationTime:    timestamppb.Now(),
		IdentityHash:      "test_identity_hash",
		TotalRewardsEarned: "0",
		LastRewardBlock:   "0",
		LastRewardTime:    timestamppb.Now(),
	}
	// First create the activated lizenz
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Update reward stats
	err = suite.keeper.UpdateRewardStats(suite.ctx, validator, 1000000, 1000, 0.95, 1.0, 1000000)
	require.NoError(suite.T(), err)

	// Verify stats were updated
	updated, err := suite.keeper.GetActivatedLizenz(suite.ctx, validator)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "1000000", updated.TotalRewardsEarned)
	require.Equal(suite.T(), "1000", updated.LastRewardBlock)
}

func (suite *RewardTrackerTestSuite) TestUpdateRewardStats_NoActivatedLizenz() {
	validator := "cosmos1nonexistent"
	
	err := suite.keeper.UpdateRewardStats(suite.ctx, validator, 1000000, 1000, 0.95, 1.0, 1000000)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "no activated LZN")
}

func (suite *RewardTrackerTestSuite) TestUpdateRewardStats_Accumulation() {
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
	// First create the activated lizenz
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Update reward stats (should accumulate)
	err = suite.keeper.UpdateRewardStats(suite.ctx, validator, 2000000, 2000, 1.0, 1.0, 2000000)
	require.NoError(suite.T(), err)

	// Verify accumulation
	updated, err := suite.keeper.GetActivatedLizenz(suite.ctx, validator)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "7000000", updated.TotalRewardsEarned) // 5000000 + 2000000
	require.Equal(suite.T(), "2000", updated.LastRewardBlock)
}

func (suite *RewardTrackerTestSuite) TestRecordRewardHistory() {
	validator := "cosmos1validator"
	
	// Create activated lizenz
	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:         validator,
		Amount:            "1000000",
		ActivationTime:    timestamppb.Now(),
		IdentityHash:      "test_identity_hash",
		TotalRewardsEarned: "0",
		LastRewardBlock:   "0",
		LastRewardTime:    timestamppb.Now(),
	}
	// First create the activated lizenz
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Record reward history
	record := RewardRecord{
		BlockHeight:    1000,
		RewardAmount:   "1000000",
		Timestamp:      suite.ctx.BlockTime().Unix(),
		MOACompliance:  0.95,
		PenaltyApplied: 1.0,
		BaseReward:     "1000000",
	}
	err = suite.keeper.RecordRewardHistory(suite.ctx, validator, record)
	require.NoError(suite.T(), err)

	// Verify history was recorded
	history, err := suite.keeper.GetRewardHistory(suite.ctx, validator)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), history, 1)
	require.Equal(suite.T(), uint64(1000), history[0].BlockHeight)
	require.Equal(suite.T(), "1000000", history[0].RewardAmount)
}

func (suite *RewardTrackerTestSuite) TestRecordRewardHistory_Multiple() {
	validator := "cosmos1validator"
	
	// Create activated lizenz
	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:         validator,
		Amount:            "1000000",
		ActivationTime:    timestamppb.Now(),
		IdentityHash:      "test_identity_hash",
		TotalRewardsEarned: "0",
		LastRewardBlock:   "0",
		LastRewardTime:    timestamppb.Now(),
	}
	// First create the activated lizenz
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Record multiple rewards
	for i := 0; i < 5; i++ {
		record := RewardRecord{
			BlockHeight:    uint64(1000 + i),
			RewardAmount:   "1000000",
			Timestamp:      suite.ctx.BlockTime().Unix(),
			MOACompliance:  0.95,
			PenaltyApplied: 1.0,
			BaseReward:     "1000000",
		}
		err = suite.keeper.RecordRewardHistory(suite.ctx, validator, record)
		require.NoError(suite.T(), err)
	}

	// Verify all records were saved
	history, err := suite.keeper.GetRewardHistory(suite.ctx, validator)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), history, 5)
}

func (suite *RewardTrackerTestSuite) TestGetRewardHistory() {
	validator := "cosmos1validator"
	
	// Create activated lizenz
	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:         validator,
		Amount:            "1000000",
		ActivationTime:    timestamppb.Now(),
		IdentityHash:      "test_identity_hash",
		TotalRewardsEarned: "0",
		LastRewardBlock:   "0",
		LastRewardTime:    timestamppb.Now(),
	}
	// First create the activated lizenz
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Get empty history
	history, err := suite.keeper.GetRewardHistory(suite.ctx, validator)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), history)

	// Record a reward
	record := RewardRecord{
		BlockHeight:    1000,
		RewardAmount:   "1000000",
		Timestamp:      suite.ctx.BlockTime().Unix(),
		MOACompliance:  0.95,
		PenaltyApplied: 1.0,
		BaseReward:     "1000000",
	}
	err = suite.keeper.RecordRewardHistory(suite.ctx, validator, record)
	require.NoError(suite.T(), err)

	// Get history
	history, err = suite.keeper.GetRewardHistory(suite.ctx, validator)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), history, 1)
}

func (suite *RewardTrackerTestSuite) TestGetRewardStats() {
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
	// First create the activated lizenz
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Record some reward history
	record := RewardRecord{
		BlockHeight:    1000,
		RewardAmount:   "1000000",
		Timestamp:      suite.ctx.BlockTime().Unix(),
		MOACompliance:  0.95,
		PenaltyApplied: 1.0,
		BaseReward:     "1000000",
	}
	err = suite.keeper.RecordRewardHistory(suite.ctx, validator, record)
	require.NoError(suite.T(), err)

	// Get reward stats
	stats, err := suite.keeper.GetRewardStats(suite.ctx, validator)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), stats)
	require.Equal(suite.T(), "5000000", stats.TotalRewardsEarned)
	require.Equal(suite.T(), "1000", stats.LastRewardBlock)
	require.Len(suite.T(), stats.RewardHistory, 1)
	require.Equal(suite.T(), 1, stats.TotalRewardsCount)
}

func (suite *RewardTrackerTestSuite) TestGetRewardStats_NoActivatedLizenz() {
	validator := "cosmos1nonexistent"
	
	_, err := suite.keeper.GetRewardStats(suite.ctx, validator)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "no activated LZN")
}

func (suite *RewardTrackerTestSuite) TestGetTotalRewardsEarned() {
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
	// First create the activated lizenz
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Get total rewards
	total, err := suite.keeper.GetTotalRewardsEarned(suite.ctx, validator)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "5000000", total)
}

func (suite *RewardTrackerTestSuite) TestGetTotalRewardsEarned_NoActivatedLizenz() {
	validator := "cosmos1nonexistent"
	
	_, err := suite.keeper.GetTotalRewardsEarned(suite.ctx, validator)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "no activated LZN")
}

