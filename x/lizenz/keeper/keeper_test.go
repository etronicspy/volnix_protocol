package keeper

import (
	"testing"
	"time"

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
	suite.storeKey = storetypes.NewKVStoreKey("test_lizenz")
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
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
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
	require.Equal(suite.T(), types.DefaultParams(), params)

	// Test setting new params
	newParams := types.DefaultParams()
	newParams.MinLznAmount = "500000"
	suite.keeper.SetParams(suite.ctx, newParams)

	// Verify params were set
	retrievedParams := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), newParams, retrievedParams)
}

func (suite *KeeperTestSuite) TestSetLizenz() {
	// Create a valid lizenz
	lizenz := types.NewLizenz("cosmos1test", "1000000", "hash123")

	// Test setting lizenz
	err := suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Test setting duplicate lizenz (should fail)
	err = suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzAlreadyExists, err)

	// Test setting lizenz with invalid amount
	invalidLizenz := types.NewLizenz("cosmos2test", "", "hash456")
	err = suite.keeper.SetLizenz(suite.ctx, invalidLizenz)
	require.Error(suite.T(), err)
}

func (suite *KeeperTestSuite) TestGetLizenz() {
	// Create and set lizenz
	lizenz := types.NewLizenz("cosmos1test", "1000000", "hash123")
	err := suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Test getting existing lizenz
	retrievedLizenz, err := suite.keeper.GetLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), lizenz.Validator, retrievedLizenz.Validator)
	require.Equal(suite.T(), lizenz.Amount, retrievedLizenz.Amount)

	// Test getting non-existent lizenz
	_, err = suite.keeper.GetLizenz(suite.ctx, "cosmos2test")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestUpdateLizenz() {
	// Create and set lizenz
	lizenz := types.NewLizenz("cosmos1test", "1000000", "hash123")
	err := suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Update lizenz
	updatedLizenz := types.NewLizenz("cosmos1test", "2000000", "hash123")
	err = suite.keeper.UpdateLizenz(suite.ctx, updatedLizenz)
	require.NoError(suite.T(), err)

	// Verify update
	retrievedLizenz, err := suite.keeper.GetLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "2000000", retrievedLizenz.Amount)

	// Test updating non-existent lizenz
	nonExistentLizenz := types.NewLizenz("cosmos2test", "1000000", "hash456")
	err = suite.keeper.UpdateLizenz(suite.ctx, nonExistentLizenz)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestDeleteLizenz() {
	// Create and set lizenz
	lizenz := types.NewLizenz("cosmos1test", "1000000", "hash123")
	err := suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Verify lizenz exists
	_, err = suite.keeper.GetLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)

	// Delete lizenz
	err = suite.keeper.DeleteLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)

	// Verify lizenz is deleted
	_, err = suite.keeper.GetLizenz(suite.ctx, "cosmos1test")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)

	// Test deleting non-existent lizenz
	err = suite.keeper.DeleteLizenz(suite.ctx, "cosmos2test")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestGetAllLizenzs() {
	// Create and set multiple lizenzs
	lizenz1 := types.NewLizenz("cosmos1test", "1000000", "hash123")
	lizenz2 := types.NewLizenz("cosmos2test", "2000000", "hash456")
	lizenz3 := types.NewLizenz("cosmos3test", "5000000", "hash789")

	err := suite.keeper.SetLizenz(suite.ctx, lizenz1)
	require.NoError(suite.T(), err)
	err = suite.keeper.SetLizenz(suite.ctx, lizenz2)
	require.NoError(suite.T(), err)
	err = suite.keeper.SetLizenz(suite.ctx, lizenz3)
	require.NoError(suite.T(), err)

	// Get all lizenzs
	lizenzs, err := suite.keeper.GetAllLizenzs(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), lizenzs, 3)

	// Verify all lizenzs are present
	owners := make([]string, 3)
	for i, liz := range lizenzs {
		owners[i] = liz.Validator
	}
	require.Contains(suite.T(), owners, "cosmos1test")
	require.Contains(suite.T(), owners, "cosmos2test")
	require.Contains(suite.T(), owners, "cosmos3test")
}

func (suite *KeeperTestSuite) TestActivateLizenz() {
	// Create and set lizenz
	lizenz := types.NewLizenz("cosmos1test", "1000000", "hash123")
	err := suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Test activating lizenz
	err = suite.keeper.ActivateLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)

	// Verify lizenz is activated
	retrievedLizenz, err := suite.keeper.GetLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), true, retrievedLizenz.IsEligibleForRewards)

	// Test activating non-existent lizenz
	err = suite.keeper.ActivateLizenz(suite.ctx, "cosmos2test")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestDeactivateLizenz() {
	// Create and set lizenz
	lizenz := types.NewLizenz("cosmos1test", "1000000", "hash123")
	err := suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Activate lizenz first
	err = suite.keeper.ActivateLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)

	// Test deactivating lizenz
	err = suite.keeper.DeactivateLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)

	// Verify lizenz is deactivated
	retrievedLizenz, err := suite.keeper.GetLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), false, retrievedLizenz.IsEligibleForRewards)

	// Test deactivating non-existent lizenz
	err = suite.keeper.DeactivateLizenz(suite.ctx, "cosmos2test")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestTransferLizenz() {
	// Create and activate lizenz
	lizenz := types.NewActivatedLizenz("cosmos1test", "1000000", "hash123")
	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Test transferring lizenz
	err = suite.keeper.TransferLizenz(suite.ctx, "cosmos1test", "cosmos2test")
	require.NoError(suite.T(), err)

	// Verify lizenz is transferred
	retrievedLizenz, err := suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos2test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "cosmos2test", retrievedLizenz.Validator)

	// Verify old owner no longer has lizenz
	_, err = suite.keeper.GetActivatedLizenz(suite.ctx, "cosmos1test")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)

	// Test transferring non-existent lizenz
	err = suite.keeper.TransferLizenz(suite.ctx, "cosmos3test", "cosmos4test")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestCheckMOA() {
	// Create and activate lizenz
	lizenz := types.NewActivatedLizenz("cosmos1test", "1000000", "hash123")
	err := suite.keeper.SetActivatedLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Create MOA status
	moaStatus := &lizenzv1.MOAStatus{
		Validator:    "cosmos1test",
		IsActive:     true,
		CurrentMoa:   "1000000",
		RequiredMoa:  "500000",
		LastActivity: timestamppb.Now(),
		NextCheck:    timestamppb.Now(),
	}
	err = suite.keeper.SetMOAStatus(suite.ctx, moaStatus)
	require.NoError(suite.T(), err)

	// Test MOA check
	isCompliant, err := suite.keeper.CheckMOA(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.True(suite.T(), isCompliant)

	// Test MOA check for non-existent lizenz
	_, err = suite.keeper.CheckMOA(suite.ctx, "cosmos2test")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrLizenzNotFound, err)
}

func (suite *KeeperTestSuite) TestProcessMOAViolations() {
	// Create and set lizenz
	lizenz := types.NewLizenz("cosmos1test", "1000000", "hash123")
	err := suite.keeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Activate lizenz
	err = suite.keeper.ActivateLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)

	// Run BeginBlocker to process MOA violations
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify lizenz still exists and is active
	retrievedLizenz, err := suite.keeper.GetLizenz(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), true, retrievedLizenz.IsEligibleForRewards)
}

func (suite *KeeperTestSuite) TestDefaultParams() {
	// Test that default params are valid
	params := types.DefaultParams()
	require.NoError(suite.T(), params.Validate())

	// Test specific values
	require.Equal(suite.T(), "1000000", params.MinLznAmount)
	require.Equal(suite.T(), "1000000000", params.MaxLznAmount)
	require.Equal(suite.T(), "1.0", params.ActivityCoefficient)
	require.Equal(suite.T(), "ulzn", params.LznDenom)
	require.Equal(suite.T(), 7*24*time.Hour, params.InactivityPeriod)
}

func (suite *KeeperTestSuite) TestParamsValidation() {
	// Test valid params
	params := types.DefaultParams()
	require.NoError(suite.T(), params.Validate())

	// Test invalid params
	invalidParams := params
	invalidParams.MinLznAmount = ""
	err := invalidParams.Validate()
	require.Error(suite.T(), err)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
