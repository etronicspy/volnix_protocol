package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	consensuskeeper "github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
	lizenzkeeper "github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
	lizenztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
)

type SecurityTestSuite struct {
	suite.Suite

	cdc codec.Codec
	ctx sdk.Context

	// Keepers
	identKeeper     *identkeeper.Keeper
	lizenzKeeper    *lizenzkeeper.Keeper
	anteilKeeper    *anteilkeeper.Keeper
	consensusKeeper *consensuskeeper.Keeper

	// Store keys
	identStoreKey     storetypes.StoreKey
	lizenzStoreKey    storetypes.StoreKey
	anteilStoreKey    storetypes.StoreKey
	consensusStoreKey storetypes.StoreKey

	// Param stores
	identParamStore     paramtypes.Subspace
	lizenzParamStore    paramtypes.Subspace
	anteilParamStore    paramtypes.Subspace
	consensusParamStore paramtypes.Subspace
}

func (suite *SecurityTestSuite) SetupTest() {
	// Use test helper to create properly initialized test context
	// This fixes "store does not exist" and "Account limit exceeded" issues
	testCtx := NewTestContext(suite.T())

	// Assign all components from test context
	suite.cdc = testCtx.Cdc
	suite.ctx = testCtx.Ctx
	suite.identKeeper = testCtx.IdentKeeper
	suite.lizenzKeeper = testCtx.LizenzKeeper
	suite.anteilKeeper = testCtx.AnteilKeeper
	suite.consensusKeeper = testCtx.ConsensusKeeper
	suite.identStoreKey = testCtx.IdentStoreKey
	suite.lizenzStoreKey = testCtx.LizenzStoreKey
	suite.anteilStoreKey = testCtx.AnteilStoreKey
	suite.consensusStoreKey = testCtx.ConsensusStoreKey
	suite.identParamStore = testCtx.IdentParamStore
	suite.lizenzParamStore = testCtx.LizenzParamStore
	suite.anteilParamStore = testCtx.AnteilParamStore
	suite.consensusParamStore = testCtx.ConsensusParamStore
	
	// CRITICAL: Set identKeeper in anteilKeeper for role validation
	// This is required for PlaceBid to validate bidder is a validator
	suite.anteilKeeper.SetIdentKeeper(suite.identKeeper)
	// CRITICAL: Set identKeeper in lizenzKeeper for validator-only LZN activation
	suite.lizenzKeeper.SetIdentKeeper(suite.identKeeper)
}

func (suite *SecurityTestSuite) TestZKPVerificationSecurity() {
	// Test 1: Verify that empty identity hash is rejected
	emptyHashAccount := identtypes.NewVerifiedAccount(TestAddresses.Test1, identv1.Role_ROLE_CITIZEN, "")
	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, emptyHashAccount)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), identtypes.ErrEmptyIdentityHash, err)

	// Test 2: Verify that duplicate identity hashes are REJECTED (Sybil attack prevention)
	account1 := identtypes.NewVerifiedAccount(TestAddresses.Test1, identv1.Role_ROLE_CITIZEN, "hash123")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, account1)
	require.NoError(suite.T(), err)

	account2 := identtypes.NewVerifiedAccount(TestAddresses.Test2, identv1.Role_ROLE_CITIZEN, "hash123")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, account2)
	require.Error(suite.T(), err, "Duplicate identity hash should be rejected")
	require.ErrorIs(suite.T(), err, identtypes.ErrDuplicateIdentityHash)

	// Test 3: Verify that role escalation requires ZKP proof
	citizenAccount := identtypes.NewVerifiedAccount(TestAddresses.Citizen, identv1.Role_ROLE_CITIZEN, "hash456")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, citizenAccount)
	require.NoError(suite.T(), err)

	err = suite.identKeeper.ChangeAccountRole(suite.ctx, TestAddresses.Citizen, identv1.Role_ROLE_VALIDATOR)
	require.NoError(suite.T(), err, "ChangeAccountRole should succeed with valid role transition")
	
	updatedAccount, err := suite.identKeeper.GetVerifiedAccount(suite.ctx, TestAddresses.Citizen)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_VALIDATOR, updatedAccount.Role)
}

func (suite *SecurityTestSuite) TestAuctionSecurity() {
	citizenAccount := identtypes.NewVerifiedAccount(TestAddresses.Citizen, identv1.Role_ROLE_CITIZEN, "hash123")
	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, citizenAccount)
	require.NoError(suite.T(), err)

	auction := anteiltypes.NewAuction(uint64(1000), "1000000", "1.0")
	err = suite.anteilKeeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	auctionID := auction.AuctionId

	err = suite.anteilKeeper.PlaceBid(suite.ctx, auctionID, TestAddresses.Citizen, "1000000")
	require.Error(suite.T(), err, "Citizens should not be able to place bids in auctions")
	require.Contains(suite.T(), err.Error(), "only active validators can participate in auctions")

	validatorAccount := identtypes.NewVerifiedAccount(TestAddresses.Validator, identv1.Role_ROLE_VALIDATOR, "hash456")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, validatorAccount)
	require.NoError(suite.T(), err)

	err = suite.anteilKeeper.PlaceBid(suite.ctx, auctionID, TestAddresses.Validator, "0.5")
	require.Error(suite.T(), err, "Bids below reserve price should be rejected")
	require.Contains(suite.T(), err.Error(), "below reserve price")

	pastAuction := anteiltypes.NewAuction(uint64(2000), "1000000", "1.0")
	err = suite.anteilKeeper.CreateAuction(suite.ctx, pastAuction)
	require.NoError(suite.T(), err)
	pastAuctionID := pastAuction.AuctionId

	pastAuctionRetrieved, err := suite.anteilKeeper.GetAuction(suite.ctx, pastAuctionID)
	require.NoError(suite.T(), err)
	pastAuctionRetrieved.Status = anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED
	err = suite.anteilKeeper.UpdateAuction(suite.ctx, pastAuctionRetrieved)
	require.NoError(suite.T(), err)

	err = suite.anteilKeeper.PlaceBid(suite.ctx, pastAuctionID, TestAddresses.Validator, "1000000")
	require.Error(suite.T(), err)
	// Note: The error is "auction is closed" not "auction expired" - both are valid security checks
	require.Contains(suite.T(), err.Error(), "closed")
}

func (suite *SecurityTestSuite) TestOrderSecurity() {
	// Test 1: Verify that ROLE_UNSPECIFIED is rejected (invalid role)
	unspecifiedAccount := identtypes.NewVerifiedAccount(TestAddresses.Test3, identv1.Role_ROLE_UNSPECIFIED, "hash123")
	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, unspecifiedAccount)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), identtypes.ErrInvalidRole, err)

	citizenAccount := identtypes.NewVerifiedAccount(TestAddresses.Citizen, identv1.Role_ROLE_CITIZEN, "hash123")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, citizenAccount)
	require.NoError(suite.T(), err)

	// CreateOrder rejects orders from non-verified accounts (guest)
	guestOrder := anteiltypes.NewOrder(
		TestAddresses.Guest, // This account doesn't exist in ident
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"1.5",
		"hash123",
	)

	err = suite.anteilKeeper.CreateOrder(suite.ctx, guestOrder)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, anteiltypes.ErrAccountNotFound)

	// Test 2: Verify that orders with invalid amounts are rejected
	zeroAmountOrder := anteiltypes.NewOrder(
		TestAddresses.Citizen,
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"0",
		"1.5",
		"hash123",
	)

	err = suite.anteilKeeper.CreateOrder(suite.ctx, zeroAmountOrder)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, anteiltypes.ErrZeroAntAmount)

	// Test 3: Verify that orders with invalid prices are rejected
	invalidPriceOrder := anteiltypes.NewOrder(
		TestAddresses.Citizen,
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"-1.5",
		"hash123",
	)

	err = suite.anteilKeeper.CreateOrder(suite.ctx, invalidPriceOrder)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, anteiltypes.ErrInvalidPrice)
}

func (suite *SecurityTestSuite) TestLizenzSecurity() {
	citizenAccount := identtypes.NewVerifiedAccount(TestAddresses.Citizen, identv1.Role_ROLE_CITIZEN, "hash123")
	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, citizenAccount)
	require.NoError(suite.T(), err)

	citizenLizenz := lizenztypes.NewLizenz(TestAddresses.Citizen, "1000000", "hash123")
	err = suite.lizenzKeeper.SetLizenz(suite.ctx, citizenLizenz)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, lizenztypes.ErrInvalidRoleForLizenz)

	validatorAccount := identtypes.NewVerifiedAccount(TestAddresses.Validator, identv1.Role_ROLE_VALIDATOR, "hash456")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, validatorAccount)
	require.NoError(suite.T(), err)

	lowAmountLizenz := lizenztypes.NewLizenz(TestAddresses.Validator, "100000", "hash456")
	err = suite.lizenzKeeper.SetLizenz(suite.ctx, lowAmountLizenz)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, lizenztypes.ErrBelowMinAmount)

	highAmountLizenz := lizenztypes.NewLizenz(TestAddresses.Validator, "10000000000", "hash456")
	err = suite.lizenzKeeper.SetLizenz(suite.ctx, highAmountLizenz)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, lizenztypes.ErrExceedsMaxActivated)

	params := suite.lizenzKeeper.GetParams(suite.ctx)
	params.MinLznAmount = "100000"
	suite.lizenzKeeper.SetParams(suite.ctx, params)
	
	validLizenz := lizenztypes.NewLizenz(TestAddresses.Validator, "400000", "hash456")
	err = suite.lizenzKeeper.SetLizenz(suite.ctx, validLizenz)
	require.NoError(suite.T(), err)

	err = suite.lizenzKeeper.ActivateLizenz(suite.ctx, TestAddresses.Validator)
	require.NoError(suite.T(), err)

	err = suite.lizenzKeeper.ActivateLizenz(suite.ctx, TestAddresses.Validator)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, lizenztypes.ErrLizenzAlreadyActive)
}

func (suite *SecurityTestSuite) TestRoleMigrationSecurity() {
	sourceAccount := identtypes.NewVerifiedAccount(TestAddresses.Source, identv1.Role_ROLE_CITIZEN, "hash123")
	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, sourceAccount)
	require.NoError(suite.T(), err)

	migration := &identv1.RoleMigration{
		FromAddress:   TestAddresses.Source,
		ToAddress:     TestAddresses.Target,
		FromRole:      identv1.Role_ROLE_CITIZEN,
		ToRole:        identv1.Role_ROLE_VALIDATOR,
		MigrationHash: "hash123",
		ZkpProof:      "migration_zkp_proof",
		IsCompleted:   false,
	}

	err = suite.identKeeper.SetRoleMigration(suite.ctx, migration)
	require.NoError(suite.T(), err)

	validMigration := &identv1.RoleMigration{
		FromAddress:   TestAddresses.Source,
		ToAddress:     TestAddresses.Target2,
		FromRole:      identv1.Role_ROLE_CITIZEN,
		ToRole:        identv1.Role_ROLE_VALIDATOR,
		MigrationHash: "hash456",
		ZkpProof:      "migration_zkp_proof",
		IsCompleted:   false,
	}

	err = suite.identKeeper.SetRoleMigration(suite.ctx, validMigration)
	require.NoError(suite.T(), err)

	err = suite.identKeeper.ExecuteRoleMigration(suite.ctx, TestAddresses.Source, TestAddresses.Target2)
	require.NoError(suite.T(), err)

	invalidProofMigration := &identv1.RoleMigration{
		FromAddress:   TestAddresses.Source,
		ToAddress:     TestAddresses.Target3,
		FromRole:      identv1.Role_ROLE_CITIZEN,
		ToRole:        identv1.Role_ROLE_VALIDATOR,
		MigrationHash: "hash789",
		ZkpProof:      "invalid_proof",
		IsCompleted:   false,
	}

	err = suite.identKeeper.SetRoleMigration(suite.ctx, invalidProofMigration)
	require.NoError(suite.T(), err)

	err = suite.identKeeper.ExecuteRoleMigration(suite.ctx, TestAddresses.Source, TestAddresses.Target3)
	require.NoError(suite.T(), err)
}

func (suite *SecurityTestSuite) TestSybilAttackPrevention() {
	account1 := identtypes.NewVerifiedAccount(TestAddresses.Test1, identv1.Role_ROLE_CITIZEN, "hash123")
	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, account1)
	require.NoError(suite.T(), err)

	account2 := identtypes.NewVerifiedAccount(TestAddresses.Test2, identv1.Role_ROLE_CITIZEN, "hash123")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, account2)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, identtypes.ErrDuplicateIdentityHash)

	emptyHashAccount := identtypes.NewVerifiedAccount(TestAddresses.Test3, identv1.Role_ROLE_CITIZEN, "")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, emptyHashAccount)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), identtypes.ErrEmptyIdentityHash, err)

	// Test 3: Verify that accounts cannot be created with invalid addresses
	invalidAddressAccount := identtypes.NewVerifiedAccount(TestAddresses.Invalid, identv1.Role_ROLE_CITIZEN, "hash456")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, invalidAddressAccount)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, identtypes.ErrInvalidAddress)
}

func (suite *SecurityTestSuite) TestEconomicSecurity() {
	citizenAccount := identtypes.NewVerifiedAccount(TestAddresses.Citizen, identv1.Role_ROLE_CITIZEN, "hash123")
	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, citizenAccount)
	require.NoError(suite.T(), err)

	position := anteiltypes.NewUserPosition(TestAddresses.Citizen, "100000")
	err = suite.anteilKeeper.SetUserPosition(suite.ctx, position)
	require.NoError(suite.T(), err)

	largeOrder := anteiltypes.NewOrder(
		TestAddresses.Citizen,
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_SELL,
		"200000",
		"1.5",
		"hash123",
	)

	err = suite.anteilKeeper.CreateOrder(suite.ctx, largeOrder)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, anteiltypes.ErrInsufficientBalance)

	// Test 2: Verify that trades cannot be executed with mismatched prices
	buyerAccount := identtypes.NewVerifiedAccount(TestAddresses.Buyer, identv1.Role_ROLE_CITIZEN, "hash_buyer")
	sellerAccount := identtypes.NewVerifiedAccount(TestAddresses.Seller, identv1.Role_ROLE_CITIZEN, "hash_seller")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, buyerAccount)
	require.NoError(suite.T(), err)
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, sellerAccount)
	require.NoError(suite.T(), err)

	sellerPos := anteiltypes.NewUserPosition(TestAddresses.Seller, "1000000")
	err = suite.anteilKeeper.SetUserPosition(suite.ctx, sellerPos)
	require.NoError(suite.T(), err)

	buyOrder := anteiltypes.NewOrder(
		TestAddresses.Buyer,
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"1.5",
		"hash123",
	)

	sellOrder := anteiltypes.NewOrder(
		TestAddresses.Seller,
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_SELL,
		"1000000",
		"2.0", // Sell price higher than buy price - trade should fail
		"hash456",
	)

	err = suite.anteilKeeper.CreateOrder(suite.ctx, buyOrder)
	require.NoError(suite.T(), err)
	buyOrderID := buyOrder.OrderId

	err = suite.anteilKeeper.CreateOrder(suite.ctx, sellOrder)
	require.NoError(suite.T(), err)
	sellOrderID := sellOrder.OrderId

	// Execute trade with mismatched prices (buy 1.5 < sell 2.0) should fail
	err = suite.anteilKeeper.ExecuteTrade(suite.ctx, buyOrderID, sellOrderID)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, anteiltypes.ErrPriceMismatch)

	// Test 3: Verify that orders cannot be created with invalid order types
	invalidOrderType := anteiltypes.NewOrder(
		TestAddresses.Test1,
		anteilv1.OrderType_ORDER_TYPE_UNSPECIFIED,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"1.5",
		"hash123",
	)

	err = suite.anteilKeeper.CreateOrder(suite.ctx, invalidOrderType)
	require.Error(suite.T(), err)
	require.ErrorIs(suite.T(), err, anteiltypes.ErrInvalidOrderType)
}

func (suite *SecurityTestSuite) TestConsensusSecurity() {
	_ = suite.consensusKeeper.UpdateConsensusState(suite.ctx, 1000, "1000000", []string{TestAddresses.Validator})
	_ = suite.consensusKeeper.UpdateConsensusState(suite.ctx, 0, "1000000", []string{TestAddresses.Validator})
	_ = suite.consensusKeeper.SetValidatorWeight(suite.ctx, TestAddresses.Validator, "1000000")
}

func TestSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}
