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

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
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

	// Set default params with higher limits for testing
	params := types.DefaultParams()
	params.MaxIdentitiesPerAddress = 1000
	suite.keeper.SetParams(suite.ctx, params)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// Test SetVerifiedAccount
func (suite *KeeperTestSuite) TestSetVerifiedAccount() {
	account := &identv1.VerifiedAccount{
		Address:              "cosmos1test",
		Role:                 identv1.Role_ROLE_CITIZEN,
		VerificationDate:     timestamppb.Now(),
		LastActive:           timestamppb.Now(),
		IsActive:             true,
		IdentityHash:         "hash123",
		VerificationProvider: "provider1",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Verify account was stored
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), account.Address, retrieved.Address)
	require.Equal(suite.T(), account.Role, retrieved.Role)
	require.Equal(suite.T(), account.IdentityHash, retrieved.IdentityHash)
}

func (suite *KeeperTestSuite) TestSetVerifiedAccount_Duplicate() {
	account := &identv1.VerifiedAccount{
		Address:              "cosmos1test",
		Role:                 identv1.Role_ROLE_CITIZEN,
		VerificationDate:     timestamppb.Now(),
		LastActive:           timestamppb.Now(),
		IsActive:             true,
		IdentityHash:         "hash123",
		VerificationProvider: "provider1",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Try to set duplicate
	err = suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAccountAlreadyExists, err)
}

func (suite *KeeperTestSuite) TestSetVerifiedAccount_InvalidAccount() {
	// Empty address
	account := &identv1.VerifiedAccount{
		Address:          "",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyAddress, err)

	// Empty identity hash
	account = &identv1.VerifiedAccount{
		Address:          "cosmos1test",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "",
	}

	err = suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyIdentityHash, err)
}

// Test GetVerifiedAccount
func (suite *KeeperTestSuite) TestGetVerifiedAccount() {
	account := &identv1.VerifiedAccount{
		Address:              "cosmos1test",
		Role:                 identv1.Role_ROLE_CITIZEN,
		VerificationDate:     timestamppb.Now(),
		LastActive:           timestamppb.Now(),
		IsActive:             true,
		IdentityHash:         "hash123",
		VerificationProvider: "provider1",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrieved)
	require.Equal(suite.T(), account.Address, retrieved.Address)
}

func (suite *KeeperTestSuite) TestGetVerifiedAccount_NotFound() {
	_, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAccountNotFound, err)
}

// Test UpdateVerifiedAccount
func (suite *KeeperTestSuite) TestUpdateVerifiedAccount() {
	account := &identv1.VerifiedAccount{
		Address:              "cosmos1test",
		Role:                 identv1.Role_ROLE_CITIZEN,
		VerificationDate:     timestamppb.Now(),
		LastActive:           timestamppb.Now(),
		IsActive:             true,
		IdentityHash:         "hash123",
		VerificationProvider: "provider1",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Update account
	account.IsActive = false
	err = suite.keeper.UpdateVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Verify update
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.False(suite.T(), retrieved.IsActive)
}

func (suite *KeeperTestSuite) TestUpdateVerifiedAccount_NotFound() {
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1notfound",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.UpdateVerifiedAccount(suite.ctx, account)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAccountNotFound, err)
}

// Test DeleteVerifiedAccount
func (suite *KeeperTestSuite) TestDeleteVerifiedAccount() {
	account := &identv1.VerifiedAccount{
		Address:              "cosmos1test",
		Role:                 identv1.Role_ROLE_CITIZEN,
		VerificationDate:     timestamppb.Now(),
		LastActive:           timestamppb.Now(),
		IsActive:             true,
		IdentityHash:         "hash123",
		VerificationProvider: "provider1",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Delete account
	err = suite.keeper.DeleteVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAccountNotFound, err)
}

// Test GetAllVerifiedAccounts
func (suite *KeeperTestSuite) TestGetAllVerifiedAccounts() {
	// Create multiple accounts
	for i := range 5 {
		account := &identv1.VerifiedAccount{
			Address:          "cosmos1test" + string(rune(i)),
			Role:             identv1.Role_ROLE_CITIZEN,
			VerificationDate: timestamppb.Now(),
			LastActive:       timestamppb.Now(),
			IsActive:         true,
			IdentityHash:     "hash" + string(rune(i)),
		}
		err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
		require.NoError(suite.T(), err)
	}

	accounts, err := suite.keeper.GetAllVerifiedAccounts(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), accounts, 5)
}

// Test GetVerifiedAccountsByRole
func (suite *KeeperTestSuite) TestGetVerifiedAccountsByRole() {
	// Create citizens
	for i := range 3 {
		account := &identv1.VerifiedAccount{
			Address:          "cosmos1citizen" + string(rune(i)),
			Role:             identv1.Role_ROLE_CITIZEN,
			VerificationDate: timestamppb.Now(),
			LastActive:       timestamppb.Now(),
			IsActive:         true,
			IdentityHash:     "hash_citizen" + string(rune(i)),
		}
		err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
		require.NoError(suite.T(), err)
	}

	// Create validators
	for i := range 2 {
		account := &identv1.VerifiedAccount{
			Address:          "cosmos1validator" + string(rune(i)),
			Role:             identv1.Role_ROLE_VALIDATOR,
			VerificationDate: timestamppb.Now(),
			LastActive:       timestamppb.Now(),
			IsActive:         true,
			IdentityHash:     "hash_validator" + string(rune(i)),
		}
		err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
		require.NoError(suite.T(), err)
	}

	// Get citizens
	citizens, err := suite.keeper.GetVerifiedAccountsByRole(suite.ctx, identv1.Role_ROLE_CITIZEN)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), citizens, 3)

	// Get validators
	validators, err := suite.keeper.GetVerifiedAccountsByRole(suite.ctx, identv1.Role_ROLE_VALIDATOR)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), validators, 2)
}

// Test ChangeAccountRole
func (suite *KeeperTestSuite) TestChangeAccountRole() {
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1test",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Change role
	err = suite.keeper.ChangeAccountRole(suite.ctx, "cosmos1test", identv1.Role_ROLE_VALIDATOR)
	require.NoError(suite.T(), err)

	// Verify role change
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_VALIDATOR, retrieved.Role)
}

func (suite *KeeperTestSuite) TestChangeAccountRole_InvalidRole() {
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1test",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Try to change to invalid role
	err = suite.keeper.ChangeAccountRole(suite.ctx, "cosmos1test", identv1.Role_ROLE_UNSPECIFIED)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrInvalidRole, err)
}

// Test UpdateAccountActivity
func (suite *KeeperTestSuite) TestUpdateAccountActivity() {
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1test",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.New(time.Now().Add(-24 * time.Hour)),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	oldTime := account.LastActive.AsTime()

	// Update activity
	err = suite.keeper.UpdateAccountActivity(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)

	// Verify activity was updated
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.True(suite.T(), retrieved.LastActive.AsTime().After(oldTime))
}

// Test BeginBlocker - account activity check
func (suite *KeeperTestSuite) TestBeginBlocker_InactiveAccounts() {
	// Set block time to current time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Create account with old last active time (older than 1 year for citizens)
	oldTime := currentTime.Add(-400 * 24 * time.Hour) // 400 days ago
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1test",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.New(oldTime),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Run BeginBlocker
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify account was downgraded to guest
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_GUEST, retrieved.Role)
}

// Test EndBlocker
func (suite *KeeperTestSuite) TestEndBlocker() {
	// Set block time to current time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	oldTime := currentTime.Add(-1 * time.Hour)
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1test",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.New(oldTime),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Run EndBlocker
	err = suite.keeper.EndBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify activity was updated
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.True(suite.T(), retrieved.LastActive.AsTime().After(oldTime))
}

// Test Role Migration
func (suite *KeeperTestSuite) TestSetRoleMigration() {
	migration := &identv1.RoleMigration{
		FromAddress:   "cosmos1from",
		ToAddress:     "cosmos1to",
		FromRole:      identv1.Role_ROLE_CITIZEN,
		ToRole:        identv1.Role_ROLE_VALIDATOR,
		MigrationHash: "hash123",
		ZkpProof:      "proof123",
		IsCompleted:   false,
	}

	err := suite.keeper.SetRoleMigration(suite.ctx, migration)
	require.NoError(suite.T(), err)

	// Verify migration was stored
	retrieved, err := suite.keeper.GetRoleMigration(suite.ctx, "cosmos1from", "cosmos1to")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), migration.FromAddress, retrieved.FromAddress)
	require.Equal(suite.T(), migration.ToAddress, retrieved.ToAddress)
}

func (suite *KeeperTestSuite) TestExecuteRoleMigration() {
	// Create source account
	sourceAccount := &identv1.VerifiedAccount{
		Address:          "cosmos1from",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, sourceAccount)
	require.NoError(suite.T(), err)

	// Create migration
	migration := &identv1.RoleMigration{
		FromAddress:   "cosmos1from",
		ToAddress:     "cosmos1to",
		FromRole:      identv1.Role_ROLE_CITIZEN,
		ToRole:        identv1.Role_ROLE_CITIZEN,
		MigrationHash: "hash123",
		ZkpProof:      "proof123",
		IsCompleted:   false,
	}

	err = suite.keeper.SetRoleMigration(suite.ctx, migration)
	require.NoError(suite.T(), err)

	// Execute migration
	err = suite.keeper.ExecuteRoleMigration(suite.ctx, "cosmos1from", "cosmos1to")
	require.NoError(suite.T(), err)

	// Verify source account is deactivated
	sourceRetrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1from")
	require.NoError(suite.T(), err)
	require.False(suite.T(), sourceRetrieved.IsActive)

	// Verify target account was created
	targetRetrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1to")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "cosmos1to", targetRetrieved.Address)
	require.Equal(suite.T(), identv1.Role_ROLE_CITIZEN, targetRetrieved.Role)
	require.True(suite.T(), targetRetrieved.IsActive)

	// Verify migration is marked as completed
	migrationRetrieved, err := suite.keeper.GetRoleMigration(suite.ctx, "cosmos1from", "cosmos1to")
	require.NoError(suite.T(), err)
	require.True(suite.T(), migrationRetrieved.IsCompleted)
}

func (suite *KeeperTestSuite) TestExecuteRoleMigration_AlreadyCompleted() {
	// Create completed migration
	migration := &identv1.RoleMigration{
		FromAddress:   "cosmos1from",
		ToAddress:     "cosmos1to",
		FromRole:      identv1.Role_ROLE_CITIZEN,
		ToRole:        identv1.Role_ROLE_CITIZEN,
		MigrationHash: "hash123",
		ZkpProof:      "proof123",
		IsCompleted:   true,
	}

	err := suite.keeper.SetRoleMigration(suite.ctx, migration)
	require.NoError(suite.T(), err)

	// Try to execute completed migration
	err = suite.keeper.ExecuteRoleMigration(suite.ctx, "cosmos1from", "cosmos1to")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrInvalidMigrationStatus, err)
}

// Test GetAllRoleMigrations
func (suite *KeeperTestSuite) TestGetAllRoleMigrations() {
	// Create multiple migrations
	for i := range 3 {
		migration := &identv1.RoleMigration{
			FromAddress:   "cosmos1from" + string(rune(i)),
			ToAddress:     "cosmos1to" + string(rune(i)),
			FromRole:      identv1.Role_ROLE_CITIZEN,
			ToRole:        identv1.Role_ROLE_VALIDATOR,
			MigrationHash: "hash" + string(rune(i)),
			ZkpProof:      "proof" + string(rune(i)),
			IsCompleted:   false,
		}
		err := suite.keeper.SetRoleMigration(suite.ctx, migration)
		require.NoError(suite.T(), err)
	}

	migrations, err := suite.keeper.GetAllRoleMigrations(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), migrations, 3)
}

// Test Params
func (suite *KeeperTestSuite) TestGetSetParams() {
	params := types.DefaultParams()
	params.MaxIdentitiesPerAddress = 100

	suite.keeper.SetParams(suite.ctx, params)

	retrieved := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), params.MaxIdentitiesPerAddress, retrieved.MaxIdentitiesPerAddress)
}

// Additional tests for uncovered methods

func (suite *KeeperTestSuite) TestSetVerifiedAccount_ExceedsLimit() {
	// Set low limit
	params := types.DefaultParams()
	params.MaxIdentitiesPerAddress = 1
	suite.keeper.SetParams(suite.ctx, params)

	// Create first account
	account1 := &identv1.VerifiedAccount{
		Address:          "cosmos1test1",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash1",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account1)
	require.NoError(suite.T(), err)

	// Try to create second account (should fail)
	account2 := &identv1.VerifiedAccount{
		Address:          "cosmos1test2",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash2",
	}

	err = suite.keeper.SetVerifiedAccount(suite.ctx, account2)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "account limit exceeded")
}

func (suite *KeeperTestSuite) TestChangeAccountRole_SameRole() {
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1test",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Change to same role (should succeed)
	err = suite.keeper.ChangeAccountRole(suite.ctx, "cosmos1test", identv1.Role_ROLE_CITIZEN)
	require.NoError(suite.T(), err)

	// Verify role unchanged
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_CITIZEN, retrieved.Role)
}

func (suite *KeeperTestSuite) TestChangeAccountRole_NotFound() {
	err := suite.keeper.ChangeAccountRole(suite.ctx, "cosmos1notfound", identv1.Role_ROLE_VALIDATOR)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAccountNotFound, err)
}

func (suite *KeeperTestSuite) TestUpdateAccountActivity_NotFound() {
	err := suite.keeper.UpdateAccountActivity(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAccountNotFound, err)
}

func (suite *KeeperTestSuite) TestGetRoleMigration_NotFound() {
	_, err := suite.keeper.GetRoleMigration(suite.ctx, "cosmos1from", "cosmos1to")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrRoleMigrationNotFound, err)
}

func (suite *KeeperTestSuite) TestExecuteRoleMigration_SourceNotFound() {
	// Create migration without source account
	migration := &identv1.RoleMigration{
		FromAddress:   "cosmos1notfound",
		ToAddress:     "cosmos1to",
		FromRole:      identv1.Role_ROLE_CITIZEN,
		ToRole:        identv1.Role_ROLE_CITIZEN,
		MigrationHash: "hash123",
		ZkpProof:      "proof123",
		IsCompleted:   false,
	}

	err := suite.keeper.SetRoleMigration(suite.ctx, migration)
	require.NoError(suite.T(), err)

	// Try to execute migration
	err = suite.keeper.ExecuteRoleMigration(suite.ctx, "cosmos1notfound", "cosmos1to")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAccountNotFound, err)
}

func (suite *KeeperTestSuite) TestGetVerifiedAccountsByRole_Empty() {
	accounts, err := suite.keeper.GetVerifiedAccountsByRole(suite.ctx, identv1.Role_ROLE_VALIDATOR)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), accounts)
}

func (suite *KeeperTestSuite) TestDeleteVerifiedAccount_NotFound() {
	err := suite.keeper.DeleteVerifiedAccount(suite.ctx, "cosmos1notfound")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrAccountNotFound, err)
}

func (suite *KeeperTestSuite) TestBeginBlocker_ActiveAccounts() {
	// Create account with recent activity
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1test",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Run BeginBlocker
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify account role unchanged (still active)
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_CITIZEN, retrieved.Role)
}

func (suite *KeeperTestSuite) TestEndBlocker_UpdatesActivity() {
	// Set block time to current time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Create account
	oldTime := currentTime.Add(-1 * time.Hour)
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1test",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.New(oldTime),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Run EndBlocker
	err = suite.keeper.EndBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify activity was updated
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.True(suite.T(), retrieved.LastActive.AsTime().After(oldTime))
}

// Additional tests for better coverage

func (suite *KeeperTestSuite) TestGetParams() {
	params := suite.keeper.GetParams(suite.ctx)
	require.NotNil(suite.T(), params)
	require.Greater(suite.T(), params.MaxIdentitiesPerAddress, uint64(0))
}

func (suite *KeeperTestSuite) TestSetParams() {
	params := types.DefaultParams()
	params.MaxIdentitiesPerAddress = 500

	suite.keeper.SetParams(suite.ctx, params)

	retrieved := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), uint64(500), retrieved.MaxIdentitiesPerAddress)
}

func (suite *KeeperTestSuite) TestBeginBlocker_ActiveAccounts_NoChange() {
	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Create account with recent activity
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1test",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Run BeginBlocker
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify account role unchanged
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1test")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_CITIZEN, retrieved.Role)
}

func (suite *KeeperTestSuite) TestBeginBlocker_ValidatorInactive() {
	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Set shorter activity period for testing
	params := suite.keeper.GetParams(suite.ctx)
	params.ValidatorActivityPeriod = 60 * 24 * time.Hour // 60 days
	suite.keeper.SetParams(suite.ctx, params)

	// Create validator with old activity (older than activity period)
	oldTime := currentTime.Add(-70 * 24 * time.Hour) // 70 days ago (older than 60 days)
	account := &identv1.VerifiedAccount{
		Address:          "cosmos1validator",
		Role:             identv1.Role_ROLE_VALIDATOR,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.New(oldTime),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Run BeginBlocker
	err = suite.keeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify validator was downgraded
	retrieved, err := suite.keeper.GetVerifiedAccount(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_GUEST, retrieved.Role)
}

func (suite *KeeperTestSuite) TestChangeAccountRole_ExceedsLimit() {
	// Set low limit
	params := types.DefaultParams()
	params.MaxIdentitiesPerAddress = 1
	suite.keeper.SetParams(suite.ctx, params)

	// Create one validator
	account1 := &identv1.VerifiedAccount{
		Address:          "cosmos1validator1",
		Role:             identv1.Role_ROLE_VALIDATOR,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash1",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, account1)
	require.NoError(suite.T(), err)

	// Create citizen and try to change to validator (should fail due to limit)
	account2 := &identv1.VerifiedAccount{
		Address:          "cosmos1citizen",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash2",
	}

	err = suite.keeper.SetVerifiedAccount(suite.ctx, account2)
	require.NoError(suite.T(), err)

	// Try to change citizen to validator (should fail)
	err = suite.keeper.ChangeAccountRole(suite.ctx, "cosmos1citizen", identv1.Role_ROLE_VALIDATOR)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "account limit exceeded")
}

func (suite *KeeperTestSuite) TestGetAllVerifiedAccounts_Empty() {
	accounts, err := suite.keeper.GetAllVerifiedAccounts(suite.ctx)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), accounts)
}

func (suite *KeeperTestSuite) TestGetAllRoleMigrations_Empty() {
	migrations, err := suite.keeper.GetAllRoleMigrations(suite.ctx)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), migrations)
}

func (suite *KeeperTestSuite) TestSetRoleMigration_Update() {
	migration := &identv1.RoleMigration{
		FromAddress:   "cosmos1from",
		ToAddress:     "cosmos1to",
		FromRole:      identv1.Role_ROLE_CITIZEN,
		ToRole:        identv1.Role_ROLE_VALIDATOR,
		MigrationHash: "hash123",
		ZkpProof:      "proof123",
		IsCompleted:   false,
	}

	err := suite.keeper.SetRoleMigration(suite.ctx, migration)
	require.NoError(suite.T(), err)

	// Update migration
	migration.IsCompleted = true
	err = suite.keeper.SetRoleMigration(suite.ctx, migration)
	require.NoError(suite.T(), err)

	// Verify update
	retrieved, err := suite.keeper.GetRoleMigration(suite.ctx, "cosmos1from", "cosmos1to")
	require.NoError(suite.T(), err)
	require.True(suite.T(), retrieved.IsCompleted)
}

func (suite *KeeperTestSuite) TestExecuteRoleMigration_TargetAlreadyExists() {
	// Create source account
	sourceAccount := &identv1.VerifiedAccount{
		Address:          "cosmos1from",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash123",
	}

	err := suite.keeper.SetVerifiedAccount(suite.ctx, sourceAccount)
	require.NoError(suite.T(), err)

	// Create target account (already exists)
	targetAccount := &identv1.VerifiedAccount{
		Address:          "cosmos1to",
		Role:             identv1.Role_ROLE_CITIZEN,
		VerificationDate: timestamppb.Now(),
		LastActive:       timestamppb.Now(),
		IsActive:         true,
		IdentityHash:     "hash456",
	}

	err = suite.keeper.SetVerifiedAccount(suite.ctx, targetAccount)
	require.NoError(suite.T(), err)

	// Create migration
	migration := &identv1.RoleMigration{
		FromAddress:   "cosmos1from",
		ToAddress:     "cosmos1to",
		FromRole:      identv1.Role_ROLE_CITIZEN,
		ToRole:        identv1.Role_ROLE_CITIZEN,
		MigrationHash: "hash123",
		ZkpProof:      "proof123",
		IsCompleted:   false,
	}

	err = suite.keeper.SetRoleMigration(suite.ctx, migration)
	require.NoError(suite.T(), err)

	// Try to execute migration (should fail because target exists)
	err = suite.keeper.ExecuteRoleMigration(suite.ctx, "cosmos1from", "cosmos1to")
	require.Error(suite.T(), err)
}
