package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"cosmossdk.io/math"
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
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

// MockBankKeeperForLizenz is a mock implementation of BankKeeperInterface for lizenz module
type MockBankKeeperForLizenz struct {
	lockedCoins   map[string]sdk.Coins   // validator -> coins (locked)
	unlockedCoins map[string]sdk.Coins    // validator -> coins (unlocked)
	errors        map[string]error       // operation -> error
}

func NewMockBankKeeperForLizenz() *MockBankKeeperForLizenz {
	return &MockBankKeeperForLizenz{
		lockedCoins:   make(map[string]sdk.Coins),
		unlockedCoins: make(map[string]sdk.Coins),
		errors:        make(map[string]error),
	}
}

func (m *MockBankKeeperForLizenz) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	if err, ok := m.errors["SendCoins"]; ok {
		return err
	}
	return nil
}

func (m *MockBankKeeperForLizenz) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	if err, ok := m.errors["SendCoinsFromAccountToModule"]; ok {
		return err
	}
	if m.lockedCoins == nil {
		m.lockedCoins = make(map[string]sdk.Coins)
	}
	// Store by validator address string
	m.lockedCoins[senderAddr.String()] = amt
	// Also store by bech32 string for lookup
	return nil
}

func (m *MockBankKeeperForLizenz) SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	if err, ok := m.errors["SendCoinsFromModuleToAccount"]; ok {
		return err
	}
	if m.unlockedCoins == nil {
		m.unlockedCoins = make(map[string]sdk.Coins)
	}
	m.unlockedCoins[recipientAddr.String()] = amt
	return nil
}

func (m *MockBankKeeperForLizenz) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, math.ZeroInt())
}

func (m *MockBankKeeperForLizenz) GetLockedCoins(validator string) sdk.Coins {
	return m.lockedCoins[validator]
}

func (m *MockBankKeeperForLizenz) GetUnlockedCoins(validator string) sdk.Coins {
	return m.unlockedCoins[validator]
}

func (m *MockBankKeeperForLizenz) SetError(operation string, err error) {
	m.errors[operation] = err
}

type LizenzBankKeeperTestSuite struct {
	suite.Suite
	ctx            sdk.Context
	keeper         *keeper.Keeper
	mockBankKeeper *MockBankKeeperForLizenz
	cdc            codec.BinaryCodec
	storeKey       storetypes.StoreKey
	paramStore     paramtypes.Subspace
}

func TestLizenzBankKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(LizenzBankKeeperTestSuite))
}

func (suite *LizenzBankKeeperTestSuite) SetupTest() {
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
	
	suite.mockBankKeeper = NewMockBankKeeperForLizenz()
	suite.keeper.SetBankKeeper(suite.mockBankKeeper)
}

// TestLZNLockingOnActivation tests that LZN tokens are locked when activating a license
func (suite *LizenzBankKeeperTestSuite) TestLZNLockingOnActivation() {
	// Create a valid bech32 address
	validatorAddr := sdk.AccAddress("validator1_______________")
	validator := validatorAddr.String()
	amount := "1000000" // 1M LZN

	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:      validator,
		Amount:         amount,
		ActivationTime: timestamppb.Now(),
		LastActivity:   timestamppb.Now(),
		IdentityHash:   "test_identity_hash_123",
		IsEligibleForRewards: true,
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Check that coins were locked
	// Check all locked coins
	hasLockedCoins := false
	for addr, coins := range suite.mockBankKeeper.lockedCoins {
		if !coins.IsZero() && coins.AmountOf("ulzn").GT(math.ZeroInt()) {
			hasLockedCoins = true
			require.Equal(suite.T(), "ulzn", coins[0].Denom)
			require.Equal(suite.T(), math.NewIntFromUint64(1000000), coins[0].Amount)
			// Verify it's the correct validator address
			require.Equal(suite.T(), validatorAddr.String(), addr)
			break
		}
	}
	require.True(suite.T(), hasLockedCoins, "LZN tokens should be locked")
}

// TestLZNUnlockingOnDeactivation tests that LZN tokens are unlocked when deactivating a license
func (suite *LizenzBankKeeperTestSuite) TestLZNUnlockingOnDeactivation() {
	// Create a valid bech32 address
	validatorAddr := sdk.AccAddress("validator1_______________")
	validator := validatorAddr.String()
	amount := "1000000" // 1M LZN

	// First activate
	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:      validator,
		Amount:         amount,
		ActivationTime: timestamppb.Now(),
		LastActivity:   timestamppb.Now(),
		IdentityHash:   "test_identity_hash_123",
		IsEligibleForRewards: true,
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Then deactivate
	err = suite.keeper.DeleteActivatedLizenz(suite.ctx, validator)
	require.NoError(suite.T(), err)

	// Check that coins were unlocked
	// Check all unlocked coins
	hasUnlockedCoins := false
	for addr, coins := range suite.mockBankKeeper.unlockedCoins {
		if !coins.IsZero() && coins.AmountOf("ulzn").GT(math.ZeroInt()) {
			hasUnlockedCoins = true
			require.Equal(suite.T(), "ulzn", coins[0].Denom)
			require.Equal(suite.T(), math.NewIntFromUint64(1000000), coins[0].Amount)
			// Verify it's the correct validator address
			require.Equal(suite.T(), validatorAddr.String(), addr)
			break
		}
	}
	require.True(suite.T(), hasUnlockedCoins, "LZN tokens should be unlocked")
}

// TestLZNLockingError tests handling of locking errors
func (suite *LizenzBankKeeperTestSuite) TestLZNLockingError() {
	suite.mockBankKeeper.SetError("SendCoinsFromAccountToModule", types.ErrEmptyValidator)

	validatorAddr := sdk.AccAddress("validator1_______________")
	validator := validatorAddr.String()
	amount := "1000000"

	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:      validator,
		Amount:         amount,
		ActivationTime: timestamppb.Now(),
		LastActivity:   timestamppb.Now(),
		IdentityHash:   "test_identity_hash_123",
		IsEligibleForRewards: true,
	}

	// Should not fail activation even if locking fails
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)
}

// TestLZNUnlockingError tests handling of unlocking errors
func (suite *LizenzBankKeeperTestSuite) TestLZNUnlockingError() {
	validatorAddr := sdk.AccAddress("validator1_______________")
	validator := validatorAddr.String()
	amount := "1000000"

	// Activate first
	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:      validator,
		Amount:         amount,
		ActivationTime: timestamppb.Now(),
		LastActivity:   timestamppb.Now(),
		IdentityHash:   "test_identity_hash_123",
		IsEligibleForRewards: true,
	}

	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)

	// Set unlock error
	suite.mockBankKeeper.SetError("SendCoinsFromModuleToAccount", types.ErrEmptyValidator)

	// Should not fail deactivation even if unlocking fails
	err = suite.keeper.DeleteActivatedLizenz(suite.ctx, validator)
	require.NoError(suite.T(), err)
}

// TestLZNLockingNoBankKeeper tests that activation works without bank keeper
func (suite *LizenzBankKeeperTestSuite) TestLZNLockingNoBankKeeper() {
	suite.keeper.SetBankKeeper(nil)

	validatorAddr := sdk.AccAddress("validator1_______________")
	validator := validatorAddr.String()
	amount := "1000000"

	activatedLizenz := &lizenzv1.ActivatedLizenz{
		Validator:      validator,
		Amount:         amount,
		ActivationTime: timestamppb.Now(),
		LastActivity:   timestamppb.Now(),
		IdentityHash:   "test_identity_hash_123",
		IsEligibleForRewards: true,
	}

	// Should work without bank keeper (just won't lock tokens)
	err := suite.keeper.SetActivatedLizenz(suite.ctx, activatedLizenz)
	require.NoError(suite.T(), err)
}

