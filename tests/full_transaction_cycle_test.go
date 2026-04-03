package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
	lizenzkeeper "github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
)

// FullTransactionCycleTestSuite tests the complete transaction flow
// from identity verification → role change → LZN activation → ANT market
// using actual Msg handlers (msg_server) to simulate real blockchain transactions
type FullTransactionCycleTestSuite struct {
	suite.Suite

	// Keepers
	identKeeper  *identkeeper.Keeper
	lizenzKeeper *lizenzkeeper.Keeper
	anteilKeeper *anteilkeeper.Keeper

	// Msg servers (for real transaction simulation)
	identMsgServer  identv1.MsgServer
	lizenzMsgServer lizenzv1.MsgServer
	anteilMsgServer anteilv1.MsgServer

	// Test context
	testCtx *TestContext
}

func (suite *FullTransactionCycleTestSuite) SetupTest() {
	suite.testCtx = NewTestContext(suite.T())
	suite.identKeeper = suite.testCtx.IdentKeeper
	suite.lizenzKeeper = suite.testCtx.LizenzKeeper
	suite.anteilKeeper = suite.testCtx.AnteilKeeper

	// Initialize msg servers
	suite.identMsgServer = identkeeper.NewMsgServer(suite.identKeeper)
	suite.lizenzMsgServer = lizenzkeeper.NewMsgServer(suite.lizenzKeeper)
	suite.anteilMsgServer = anteilkeeper.NewMsgServer(suite.anteilKeeper)

	// Set up keeper dependencies
	suite.anteilKeeper.SetIdentKeeper(suite.identKeeper)
}

// TestCompleteUserJourney tests the full user journey:
// 1. Identity Verification (Guest → Supplier)
// 2. Role Change (Supplier → Validator) with ZKP
// 3. LZN Activation (Validator activates mining license)
// 4. ANT Market Participation (Create orders, place bids)
func (suite *FullTransactionCycleTestSuite) TestCompleteUserJourney() {
	ctx := suite.testCtx.Ctx
	userAddr := TestAddresses.Test1

	suite.T().Log("=== Phase 1: Identity Verification ===")

	// Step 1: Verify identity as Guest (becomes Supplier)
	// Note: In real scenario, verification provider would be registered first
	// For test, we'll use empty string or skip provider validation
	verifyMsg := &identv1.MsgVerifyIdentity{
		Address:              userAddr,
		ZkpProof:             "zkp_proof_verification_123456789012345678901234567890123456789012345678901234567890", // Must be >= 64 bytes
		VerificationProvider: "",                                                                                    // Empty for test (will be validated in keeper)
		VerificationCost:     nil,                                                                                   // Optional
		DesiredRole:          identv1.Role_ROLE_SUPPLIER,
	}

	verifyResp, err := suite.identMsgServer.VerifyIdentity(ctx, verifyMsg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), verifyResp)
	suite.T().Log("✓ Identity verified, user is now Supplier")

	// Verify account was created
	account, err := suite.identKeeper.GetVerifiedAccount(ctx, userAddr)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_SUPPLIER, account.Role)
	require.True(suite.T(), account.IsActive)

	suite.T().Log("=== Phase 2: Role Change (Supplier → Validator) ===")

	// Step 2: Change role from Supplier to Validator (requires ZKP proof)
	changeRoleMsg := &identv1.MsgChangeRole{
		Address:   userAddr,
		NewRole:   identv1.Role_ROLE_VALIDATOR,
		ZkpProof:  "zkp_proof_role_change_123456789012345678901234567890123456789012345678901234567890", // Must be >= 64 bytes
		ChangeFee: nil,                                                                                  // Optional
	}

	changeRoleResp, err := suite.identMsgServer.ChangeRole(ctx, changeRoleMsg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), changeRoleResp)
	suite.T().Log("✓ Role changed from Supplier to Validator")

	// Verify role was changed
	account, err = suite.identKeeper.GetVerifiedAccount(ctx, userAddr)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_VALIDATOR, account.Role)

	suite.T().Log("=== Phase 3: LZN License Activation ===")

	// Step 3: Activate LZN license directly
	// Note: In real scenario, LZN would be minted/transferred first, then activated
	// For test, we activate directly which creates both LZN and ActivatedLizenz
	activateMsg := &lizenzv1.MsgActivateLZN{
		Validator:    userAddr,
		Amount:       "1000000", // 1M LZN
		IdentityHash: "hash_user_123",
	}

	activateResp, err := suite.lizenzMsgServer.ActivateLZN(ctx, activateMsg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), activateResp)
	suite.T().Log("✓ LZN license activated")

	// Verify LZN is activated
	activatedLizenz, err := suite.lizenzKeeper.GetActivatedLizenz(ctx, userAddr)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), activatedLizenz)

	suite.T().Log("=== Phase 4: ANT Market Participation ===")

	// Step 5: Create ANT position for user (simulate receiving ANT)
	// In real scenario, ANT would be distributed or transferred
	position := anteiltypes.NewUserPosition(userAddr, "10000000") // 10M ANT
	err = suite.anteilKeeper.SetUserPosition(ctx, position)
	require.NoError(suite.T(), err)
	suite.T().Log("✓ ANT position created")

	// Step 6: Create sell order (user selling ANT)
	sellOrderMsg := &anteilv1.MsgPlaceOrder{
		Owner:        userAddr,
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_SELL,
		AntAmount:    "1000000", // 1M ANT
		Price:        "1.5",     // 1.5 WRT per ANT
		IdentityHash: "hash_user_123",
	}

	sellOrderResp, err := suite.anteilMsgServer.PlaceOrder(ctx, sellOrderMsg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), sellOrderResp)
	require.NotEmpty(suite.T(), sellOrderResp.OrderId)
	suite.T().Logf("✓ Sell order created: %s", sellOrderResp.OrderId)

	// Verify order was created
	order, err := suite.anteilKeeper.GetOrder(ctx, sellOrderResp.OrderId)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.OrderStatus_ORDER_STATUS_OPEN, order.Status)
	require.Equal(suite.T(), userAddr, order.Owner)

	suite.T().Log("=== Phase 5: Verification ===")

	// Final verification: Check all states
	account, err = suite.identKeeper.GetVerifiedAccount(ctx, userAddr)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_VALIDATOR, account.Role)
	require.True(suite.T(), account.IsActive)

	activatedLizenz, err = suite.lizenzKeeper.GetActivatedLizenz(ctx, userAddr)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), activatedLizenz)

	positionRetrieved, err := suite.anteilKeeper.GetUserPosition(ctx, userAddr)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "10000000", positionRetrieved.AntBalance)

	orders, err := suite.anteilKeeper.GetAllOrders(ctx)
	require.NoError(suite.T(), err)
	require.Greater(suite.T(), len(orders), 0)

	suite.T().Log("🎉 Complete user journey test passed!")
	suite.T().Log("✓ Identity verified")
	suite.T().Log("✓ Role changed (Supplier → Validator)")
	suite.T().Log("✓ LZN activated")
	suite.T().Log("✓ ANT market operations completed")
}

// TestRoleChangeValidation tests that role changes follow proper validation rules
func (suite *FullTransactionCycleTestSuite) TestRoleChangeValidation() {
	ctx := suite.testCtx.Ctx
	userAddr := TestAddresses.Test2

	// Step 1: Create Supplier account
	citizenAccount := identtypes.NewVerifiedAccount(userAddr, identv1.Role_ROLE_SUPPLIER, "hash_456")
	err := suite.identKeeper.SetVerifiedAccount(ctx, citizenAccount)
	require.NoError(suite.T(), err)

	// Step 2: Try to change role directly to Validator (should require ZKP)
	changeRoleMsg := &identv1.MsgChangeRole{
		Address:   userAddr,
		NewRole:   identv1.Role_ROLE_VALIDATOR,
		ZkpProof:  "zkp_proof_valid_123456789012345678901234567890123456789012345678901234567890", // Must be >= 64 bytes
		ChangeFee: nil,
	}

	changeRoleResp, err := suite.identMsgServer.ChangeRole(ctx, changeRoleMsg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), changeRoleResp)

	// Verify role was changed
	account, err := suite.identKeeper.GetVerifiedAccount(ctx, userAddr)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_VALIDATOR, account.Role)

	// Step 3: Try invalid role change (Validator → Guest) - should fail
	invalidChangeMsg := &identv1.MsgChangeRole{
		Address:   userAddr,
		NewRole:   identv1.Role_ROLE_GUEST,
		ZkpProof:  "zkp_proof_invalid_123456789012345678901234567890123456789012345678901234567890", // Must be >= 64 bytes
		ChangeFee: nil,
	}

	_, err = suite.identMsgServer.ChangeRole(ctx, invalidChangeMsg)
	require.Error(suite.T(), err) // Should fail - cannot downgrade directly
	suite.T().Log("✓ Invalid role downgrade correctly rejected")

	// Verify role was NOT changed
	account, err = suite.identKeeper.GetVerifiedAccount(ctx, userAddr)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_VALIDATOR, account.Role) // Still validator
}

func TestFullTransactionCycleTestSuite(t *testing.T) {
	suite.Run(t, new(FullTransactionCycleTestSuite))
}
