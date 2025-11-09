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

type KeeperTestSuite struct {
	suite.Suite

	ctx        sdk.Context
	keeper     *keeper.Keeper
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
}

func (suite *KeeperTestSuite) SetupTest() {
	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)

	// Create store keys
	suite.storeKey = storetypes.NewKVStoreKey(types.StoreKey)
	tKey := storetypes.NewTransientStoreKey("transient_test")

	// Create test context
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)

	// Create params keeper and subspace
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore = suite.paramStore.WithKeyTable(types.ParamKeyTable())

	// Create keeper
	suite.keeper = keeper.NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)

	// Set default params
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// Test SetActivatedLizenz
func (suite *KeeperTestSuite) TestSetActivatedLizenz() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Verify lizenz was stored
	retrieved, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), lizenz.Validator, retrieved.Validator)
	require.Equal(suite.T(), lizenz.Amount, retrieved.Amount)
}

func (suite *KeeperTestSuite) TestSetActivatedLizenz_Duplicate() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Try to set duplicate
	err = suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzAlreadyExists, err)
}

func (suite *KeeperTestSuite) TestSetActivatedLizenz_InvalidAmount() {
	// Amount below minimum
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "100",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrBelowMinAmount, err)
}

// Test GetActivatedLizenz
func (suite *KeeperTestSuite) TestGetActivatedLizenz() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	retrieved, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrieved)
	require.Equal(suite.T(), lizenz.Validator, retrieved.Validator)
}

func (suite *KeeperTestSuite) TestGetActivatedLizenz_NotFound() {
	_, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestGetActivatedLizenz_EmptyValidator() {
	_, err := suite.keeper.GetActivatedLizenz(suite.ctx, "")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidator, err)
}

// Test UpdateActivatedLizenz
func (suite *KeeperTestSuite) TestUpdateActivatedLizenz() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Update lizenz
	lizenz.IsEligibleForRewards = false
	err = suite.keeper.UpdateActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Verify update
	retrieved, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.False(suite.T(), retrieved.IsEligibleForRewards)
}

// Test DeleteActivatedLizenz
func (suite *KeeperTestSuite) TestDeleteActivatedLizenz() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Delete lizenz
	err = suite.keeper.DeleteActivatedLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

// Test GetAllActivatedLizenz
func (suite *KeeperTestSuite) TestGetAllActivatedLizenz() {
	// Create multiple lizenz
	for i := range 5 {
		lizenz := &lizenzv1.ActivatedLizenz{
			Validator:            "cosmos1validator" + string(rune(i)),
			Amount:               "1000000",
			ActivationTime:       timestamppb.Now(),
			LastActivity:         timestamppb.Now(),
			IsEligibleForRewards: true,
			IdentityHash:         "hash" + string(rune(i)),
		}
		err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
		require.NoError(suite.T(), err)
	}

	lizenzs, err := suite.keeper.GetAllActivatedLizenz(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), lizenzs, 5)
}

// Test ActivateLizenz
func (suite *KeeperTestSuite) TestActivateLizenz() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: false,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Activate lizenz
	err = suite.keeper.ActivateLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)

	// Verify activation
	retrieved, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.True(suite.T(), retrieved.IsEligibleForRewards)
}

// Test DeactivateLizenz
func (suite *KeeperTestSuite) TestDeactivateLizenz() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Deactivate lizenz
	err = suite.keeper.DeactivateLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)

	// Verify deactivation
	retrieved, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.False(suite.T(), retrieved.IsEligibleForRewards)
}

// Test TransferLizenz
func (suite *KeeperTestSuite) TestTransferLizenz() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator1",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Transfer lizenz
	err = suite.keeper.TransferLizenz(suite.ctx, "cosmos1validator1", "cosmos1validator2")
	require.NoError(suite.T(), err)

	// Verify old lizenz is deleted
	_, err = suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator1")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)

	// Verify new lizenz exists
	retrieved, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator2")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "cosmos1validator2", retrieved.Validator)
	require.Equal(suite.T(), "1000000", retrieved.Amount)
}

// Test MOA Status
func (suite *KeeperTestSuite) TestSetMOAStatus() {
	status := &lizenzv1.MOAStatus{
		Validator:    "cosmos1validator",
		CurrentMoa:   "1000000",
		RequiredMoa:  "500000",
		LastActivity: timestamppb.Now(),
		IsCompliant:  true,
	}

	err := suite.keeper.SetMOAStatus(suite.ctx, status)
	require.NoError(suite.T(), err)

	// Verify status was stored
	retrieved, err := suite.keeper.GetMOAStatus(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), status.Validator, retrieved.Validator)
	require.Equal(suite.T(), status.CurrentMoa, retrieved.CurrentMoa)
}

func (suite *KeeperTestSuite) TestGetMOAStatus_NotFound() {
	_, err := suite.keeper.GetMOAStatus(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestGetAllMOAStatus() {
	// Create multiple MOA statuses
	for i := range 3 {
		status := &lizenzv1.MOAStatus{
			Validator:    "cosmos1validator" + string(rune(i)),
			CurrentMoa:   "1000000",
			RequiredMoa:  "500000",
			LastActivity: timestamppb.Now(),
			IsCompliant:  true,
		}
		err := suite.keeper.SetMOAStatus(suite.ctx, status)
		require.NoError(suite.T(), err)
	}

	statuses, err := suite.keeper.GetAllMOAStatus(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), statuses, 3)
}

// Test CheckMOA
func (suite *KeeperTestSuite) TestCheckMOA_Compliant() {
	status := &lizenzv1.MOAStatus{
		Validator:    "cosmos1validator",
		CurrentMoa:   "1000000",
		RequiredMoa:  "500000",
		LastActivity: timestamppb.Now(),
		IsCompliant:  true,
	}

	err := suite.keeper.SetMOAStatus(suite.ctx, status)
	require.NoError(suite.T(), err)

	isCompliant, err := suite.keeper.CheckMOA(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.True(suite.T(), isCompliant)
}

func (suite *KeeperTestSuite) TestCheckMOA_NonCompliant() {
	status := &lizenzv1.MOAStatus{
		Validator:    "cosmos1validator",
		CurrentMoa:   "300000",
		RequiredMoa:  "500000",
		LastActivity: timestamppb.Now(),
		IsCompliant:  false,
	}

	err := suite.keeper.SetMOAStatus(suite.ctx, status)
	require.NoError(suite.T(), err)

	isCompliant, err := suite.keeper.CheckMOA(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.False(suite.T(), isCompliant)
}

// Test Deactivating Lizenz
func (suite *KeeperTestSuite) TestSetDeactivatingLizenz() {
	lizenz := &lizenzv1.DeactivatingLizenz{
		Validator:         "cosmos1validator",
		Amount:            "1000000",
		DeactivationEnd:   timestamppb.New(time.Now().Add(30 * 24 * time.Hour)),
		DeactivationStart: timestamppb.Now(),
		Reason:            "inactivity",
	}

	err := suite.keeper.SetDeactivatingLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Verify lizenz was stored
	retrieved, err := suite.keeper.GetDeactivatingLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), lizenz.Validator, retrieved.Validator)
	require.Equal(suite.T(), lizenz.Reason, retrieved.Reason)
}

func (suite *KeeperTestSuite) TestGetAllDeactivatingLizenz() {
	// Create multiple deactivating lizenz
	for i := range 3 {
		lizenz := &lizenzv1.DeactivatingLizenz{
			Validator:         "cosmos1validator" + string(rune(i)),
			Amount:            "1000000",
			DeactivationEnd:   timestamppb.New(time.Now().Add(30 * 24 * time.Hour)),
			DeactivationStart: timestamppb.Now(),
			Reason:            "inactivity",
		}
		err := suite.keeper.SetDeactivatingLizenz(suite.ctx, lizenz)
		require.NoError(suite.T(), err)
	}

	lizenzs, err := suite.keeper.GetAllDeactivatingLizenz(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), lizenzs, 3)
}

func (suite *KeeperTestSuite) TestDeleteDeactivatingLizenz() {
	lizenz := &lizenzv1.DeactivatingLizenz{
		Validator:         "cosmos1validator",
		Amount:            "1000000",
		DeactivationEnd:   timestamppb.New(time.Now().Add(30 * 24 * time.Hour)),
		DeactivationStart: timestamppb.Now(),
		Reason:            "inactivity",
	}

	err := suite.keeper.SetDeactivatingLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Delete lizenz
	err = suite.keeper.DeleteDeactivatingLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.keeper.GetDeactivatingLizenz(suite.ctx, "cosmos1validator")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

// Test BeginBlocker - CheckInactiveLizenz
func (suite *KeeperTestSuite) TestBeginBlocker_InactiveLizenz() {
	// Set block time to current time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Create lizenz with old last activity (older than InactivityPeriod)
	params := suite.keeper.GetParams(suite.ctx)
	oldTime := currentTime.Add(-params.InactivityPeriod - 24*time.Hour) // 1 day past threshold

	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.New(oldTime),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Run BeginBlocker
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify lizenz was moved to deactivating
	_, err = suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)

	// Verify deactivating lizenz exists
	deactivating, err := suite.keeper.GetDeactivatingLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "cosmos1validator", deactivating.Validator)
	require.Equal(suite.T(), "inactivity", deactivating.Reason)
}

// Test ProcessDeactivatingLizenz
func (suite *KeeperTestSuite) TestProcessDeactivatingLizenz() {
	// Set block time to current time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Get deactivation period from params
	params := suite.keeper.GetParams(suite.ctx)

	// Create deactivating lizenz with end date older than deactivation threshold
	// DeactivationEnd should be before (currentTime - DeactivationPeriod)
	pastTime := currentTime.Add(-params.DeactivationPeriod - 24*time.Hour) // 1 day past threshold

	lizenz := &lizenzv1.DeactivatingLizenz{
		Validator:         "cosmos1validator",
		Amount:            "1000000",
		DeactivationEnd:   timestamppb.New(pastTime),
		DeactivationStart: timestamppb.New(pastTime.Add(-30 * 24 * time.Hour)),
		Reason:            "inactivity",
	}

	err := suite.keeper.SetDeactivatingLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Run BeginBlocker (which calls ProcessDeactivatingLizenz)
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify deactivating lizenz was removed
	_, err = suite.keeper.GetDeactivatingLizenz(suite.ctx, "cosmos1validator")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

// Test UpdateLizenzActivity
func (suite *KeeperTestSuite) TestUpdateLizenzActivity() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.New(time.Now().Add(-24 * time.Hour)),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	oldTime := lizenz.LastActivity.AsTime()

	// Update activity
	err = suite.keeper.UpdateLizenzActivity(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)

	// Verify activity was updated
	retrieved, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.True(suite.T(), retrieved.LastActivity.AsTime().After(oldTime))
}

// Test Params
func (suite *KeeperTestSuite) TestGetSetParams() {
	params := types.DefaultParams()
	params.MinLznAmount = "2000000"

	suite.keeper.SetParams(suite.ctx, params)

	retrieved := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), params.MinLznAmount, retrieved.MinLznAmount)
}

// Additional tests for uncovered methods

func (suite *KeeperTestSuite) TestGetLizenz() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Get using alias method
	retrieved, err := suite.keeper.GetLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), lizenz.Validator, retrieved.Validator)
}

func (suite *KeeperTestSuite) TestUpdateLizenz() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Update using alias method
	lizenz.IsEligibleForRewards = false
	err = suite.keeper.UpdateLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Verify update
	retrieved, err := suite.keeper.GetLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.False(suite.T(), retrieved.IsEligibleForRewards)
}

func (suite *KeeperTestSuite) TestDeleteLizenz() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Delete using alias method
	err = suite.keeper.DeleteLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.keeper.GetLizenz(suite.ctx, "cosmos1validator")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestGetAllLizenzs() {
	// Create multiple lizenz
	for i := range 3 {
		lizenz := &lizenzv1.ActivatedLizenz{
			Validator:            "cosmos1validator" + string(rune(i)),
			Amount:               "1000000",
			ActivationTime:       timestamppb.Now(),
			LastActivity:         timestamppb.Now(),
			IsEligibleForRewards: true,
			IdentityHash:         "hash" + string(rune(i)),
		}
		err := suite.keeper.SetLizenz(suite.ctx, lizenz)
		require.NoError(suite.T(), err)
	}

	// Get all using alias method
	lizenzs, err := suite.keeper.GetAllLizenzs(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), lizenzs, 3)
}

func (suite *KeeperTestSuite) TestSetActivatedLizenz_ExceedsMaxAmount() {
	// Set params with low max amount
	params := types.DefaultParams()
	params.MaxLznAmount = "500000"
	suite.keeper.SetParams(suite.ctx, params)

	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000", // Exceeds max
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrExceedsMaxActivated, err)
}

func (suite *KeeperTestSuite) TestUpdateActivatedLizenz_NotFound() {
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1notfound",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.UpdateActivatedLizenz(suite.ctx, lizenz)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestDeleteActivatedLizenz_EmptyValidator() {
	err := suite.keeper.DeleteActivatedLizenz(suite.ctx, "")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidator, err)
}

func (suite *KeeperTestSuite) TestActivateLizenz_NotFound() {
	err := suite.keeper.ActivateLizenz(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestDeactivateLizenz_NotFound() {
	err := suite.keeper.DeactivateLizenz(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestTransferLizenz_NotFound() {
	err := suite.keeper.TransferLizenz(suite.ctx, "cosmos1notfound", "cosmos1to")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestCheckMOA_NotFound() {
	_, err := suite.keeper.CheckMOA(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestSetDeactivatingLizenz_Duplicate() {
	lizenz := &lizenzv1.DeactivatingLizenz{
		Validator:         "cosmos1validator",
		Amount:            "1000000",
		DeactivationEnd:   timestamppb.New(time.Now().Add(30 * 24 * time.Hour)),
		DeactivationStart: timestamppb.Now(),
		Reason:            "inactivity",
	}

	err := suite.keeper.SetDeactivatingLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Try to set duplicate
	err = suite.keeper.SetDeactivatingLizenz(suite.ctx, lizenz)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzAlreadyExists, err)
}

func (suite *KeeperTestSuite) TestGetDeactivatingLizenz_EmptyValidator() {
	_, err := suite.keeper.GetDeactivatingLizenz(suite.ctx, "")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidator, err)
}

func (suite *KeeperTestSuite) TestGetDeactivatingLizenz_NotFound() {
	_, err := suite.keeper.GetDeactivatingLizenz(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestDeleteDeactivatingLizenz_EmptyValidator() {
	err := suite.keeper.DeleteDeactivatingLizenz(suite.ctx, "")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidator, err)
}

func (suite *KeeperTestSuite) TestDeleteDeactivatingLizenz_NotFound() {
	err := suite.keeper.DeleteDeactivatingLizenz(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestSetMOAStatus_Duplicate() {
	status := &lizenzv1.MOAStatus{
		Validator:    "cosmos1validator",
		CurrentMoa:   "1000000",
		RequiredMoa:  "500000",
		LastActivity: timestamppb.Now(),
		IsCompliant:  true,
	}

	err := suite.keeper.SetMOAStatus(suite.ctx, status)
	require.NoError(suite.T(), err)

	// Update with new values (should succeed)
	status.CurrentMoa = "1500000"
	err = suite.keeper.SetMOAStatus(suite.ctx, status)
	require.NoError(suite.T(), err)
}

func (suite *KeeperTestSuite) TestGetMOAStatus_EmptyValidator() {
	_, err := suite.keeper.GetMOAStatus(suite.ctx, "")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidator, err)
}

func (suite *KeeperTestSuite) TestUpdateLizenzActivity_NotFound() {
	err := suite.keeper.UpdateLizenzActivity(suite.ctx, "cosmos1notfound")
	require.NoError(suite.T(), err) // Should not error, just skip
}

func (suite *KeeperTestSuite) TestUpdateLizenzActivity_Success() {
	// Create activated lizenz
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.New(time.Now().Add(-24 * time.Hour)),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Create MOA status
	status := &lizenzv1.MOAStatus{
		Validator:    "cosmos1validator",
		CurrentMoa:   "1000000",
		RequiredMoa:  "500000",
		LastActivity: timestamppb.New(time.Now().Add(-24 * time.Hour)),
		IsCompliant:  true,
	}

	err = suite.keeper.SetMOAStatus(suite.ctx, status)
	require.NoError(suite.T(), err)

	oldTime := lizenz.LastActivity.AsTime()

	// Update activity
	err = suite.keeper.UpdateLizenzActivity(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)

	// Verify activity was updated
	retrieved, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.True(suite.T(), retrieved.LastActivity.AsTime().After(oldTime))
}

func (suite *KeeperTestSuite) TestBeginBlocker_ActiveLizenz() {
	// Create lizenz with recent activity
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Run BeginBlocker
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify lizenz still active (not moved to deactivating)
	retrieved, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "cosmos1validator", retrieved.Validator)
}

func (suite *KeeperTestSuite) TestProcessDeactivatingLizenz_NotExpired() {
	// Create deactivating lizenz with future end date
	futureTime := time.Now().Add(30 * 24 * time.Hour)
	lizenz := &lizenzv1.DeactivatingLizenz{
		Validator:         "cosmos1validator",
		Amount:            "1000000",
		DeactivationEnd:   timestamppb.New(futureTime),
		DeactivationStart: timestamppb.Now(),
		Reason:            "inactivity",
	}

	err := suite.keeper.SetDeactivatingLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Run BeginBlocker (which calls ProcessDeactivatingLizenz)
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify deactivating lizenz still exists (not removed)
	retrieved, err := suite.keeper.GetDeactivatingLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "cosmos1validator", retrieved.Validator)
}
