package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	lizenzkeeper "github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
)

// ============================================================================
// Identity Helpers
// ============================================================================

// AssertAccountRole checks account has expected role
func AssertAccountRole(t *testing.T, keeper *identkeeper.Keeper, ctx sdk.Context, address string, expectedRole identv1.Role) {
	account, err := keeper.GetVerifiedAccount(ctx, address)
	require.NoError(t, err, "Failed to get account %s", address)
	require.Equal(t, expectedRole, account.Role, "Account %s should have role %s", address, expectedRole.String())
}

// AssertAccountActive checks account is active
func AssertAccountActive(t *testing.T, keeper *identkeeper.Keeper, ctx sdk.Context, address string, expectedActive bool) {
	account, err := keeper.GetVerifiedAccount(ctx, address)
	require.NoError(t, err, "Failed to get account %s", address)
	require.Equal(t, expectedActive, account.IsActive, "Account %s active status mismatch", address)
}

// CreateTestAccounts creates multiple test accounts with specified roles
func CreateTestAccounts(t *testing.T, keeper *identkeeper.Keeper, ctx sdk.Context, roles []identv1.Role) []*identv1.VerifiedAccount {
	accounts := make([]*identv1.VerifiedAccount, len(roles))
	for i, role := range roles {
		acc := NewTestVerifiedAccountCustom(
			fmt.Sprintf("cosmos1test%d", i),
			role,
			fmt.Sprintf("hash%d", i),
		)
		err := keeper.SetVerifiedAccount(ctx, acc)
		require.NoError(t, err, "Failed to create test account %d", i)
		accounts[i] = acc
	}
	return accounts
}

// CreateInactiveAccount creates an inactive account for testing
func CreateInactiveAccount(t *testing.T, keeper *identkeeper.Keeper, ctx sdk.Context, address string, role identv1.Role) *identv1.VerifiedAccount {
	acc := NewTestVerifiedAccountCustom(address, role, "hash-inactive")
	acc.IsActive = false
	err := keeper.SetVerifiedAccount(ctx, acc)
	require.NoError(t, err, "Failed to create inactive account")
	return acc
}

// ============================================================================
// Anteil (Market) Helpers
// ============================================================================

// AssertOrderExists checks order exists in store
func AssertOrderExists(t *testing.T, keeper *anteilkeeper.Keeper, ctx sdk.Context, orderID string) *anteilv1.Order {
	order, err := keeper.GetOrder(ctx, orderID)
	require.NoError(t, err, "Order %s should exist", orderID)
	require.NotNil(t, order, "Order %s should not be nil", orderID)
	return order
}

// AssertOrderNotExists checks order does not exist
func AssertOrderNotExists(t *testing.T, keeper *anteilkeeper.Keeper, ctx sdk.Context, orderID string) {
	_, err := keeper.GetOrder(ctx, orderID)
	require.Error(t, err, "Order %s should not exist", orderID)
}

// AssertUserPosition checks user has expected ANT balance
func AssertUserPosition(t *testing.T, keeper *anteilkeeper.Keeper, ctx sdk.Context, address string, expectedBalance string) {
	position, err := keeper.GetUserPosition(ctx, address)
	require.NoError(t, err, "Failed to get position for %s", address)
	require.Equal(t, expectedBalance, position.AntBalance, "User %s should have balance %s", address, expectedBalance)
}

// CreateTestOrders creates multiple test orders
func CreateTestOrders(t *testing.T, keeper *anteilkeeper.Keeper, ctx sdk.Context, count int, orderSide anteilv1.OrderSide) []*anteilv1.Order {
	orders := make([]*anteilv1.Order, count)
	for i := 0; i < count; i++ {
		order := NewTestOrder(
			fmt.Sprintf("cosmos1owner%d", i),
			anteilv1.OrderType_ORDER_TYPE_LIMIT,
			orderSide,
		)
		err := keeper.SetOrder(ctx, order)
		require.NoError(t, err, "Failed to create test order %d", i)
		orders[i] = order
	}
	return orders
}

// SetupAuctionWithValidators creates auction and validator accounts
func SetupAuctionWithValidators(t *testing.T, anteilKeeper *anteilkeeper.Keeper, identKeeper *identkeeper.Keeper, ctx sdk.Context, numValidators int) (string, []*identv1.VerifiedAccount) {
	// Create validators
	validators := make([]*identv1.VerifiedAccount, numValidators)
	for i := 0; i < numValidators; i++ {
		val := NewTestVerifiedAccountCustom(
			fmt.Sprintf("cosmos1validator%d", i),
			identv1.Role_ROLE_VALIDATOR,
			fmt.Sprintf("hash-val-%d", i),
		)
		err := identKeeper.SetVerifiedAccount(ctx, val)
		require.NoError(t, err, "Failed to create validator %d", i)
		validators[i] = val
	}
	
	// Create auction
	auction := NewTestAuction(1000)
	err := anteilKeeper.CreateAuction(ctx, auction)
	require.NoError(t, err, "Failed to create auction")
	
	return auction.AuctionId, validators
}

// ============================================================================
// Lizenz Helpers
// ============================================================================

// AssertLizenzExists checks lizenz exists for validator
func AssertLizenzExists(t *testing.T, keeper *lizenzkeeper.Keeper, ctx sdk.Context, validator string) interface{} {
	lizenz, err := keeper.GetLizenz(ctx, validator)
	require.NoError(t, err, "Lizenz for %s should exist", validator)
	require.NotNil(t, lizenz, "Lizenz should not be nil")
	return lizenz
}

// CreateTestLizenz creates and stores test lizenz
func CreateTestLizenz(t *testing.T, keeper *lizenzkeeper.Keeper, ctx sdk.Context, owner string, amount string) interface{} {
	lizenz := NewTestLizenz(owner, amount)
	err := keeper.SetLizenz(ctx, lizenz)
	require.NoError(t, err)
	return lizenz
}

// ============================================================================
// General Test Helpers
// ============================================================================

// RequireErrorContains checks error contains expected substring
func RequireErrorContains(t *testing.T, err error, expectedMsg string) {
	require.Error(t, err, "Expected error but got nil")
	require.Contains(t, err.Error(), expectedMsg, "Error message should contain '%s'", expectedMsg)
}

// RequireNoErrorf is a formatted version of require.NoError
func RequireNoErrorf(t *testing.T, err error, format string, args ...interface{}) {
	if err != nil {
		t.Errorf(format+": %v", append(args, err)...)
	}
}

// RequireEventEmitted checks that specific event was emitted
func RequireEventEmitted(t *testing.T, ctx sdk.Context, eventType string) {
	events := ctx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == eventType {
			found = true
			break
		}
	}
	require.True(t, found, "Event %s should be emitted", eventType)
}

// RequireEventNotEmitted checks that specific event was NOT emitted
func RequireEventNotEmitted(t *testing.T, ctx sdk.Context, eventType string) {
	events := ctx.EventManager().Events()
	for _, event := range events {
		require.NotEqual(t, eventType, event.Type, "Event %s should not be emitted", eventType)
	}
}

// GetEventAttribute retrieves attribute value from event
func GetEventAttribute(t *testing.T, ctx sdk.Context, eventType string, attributeKey string) string {
	events := ctx.EventManager().Events()
	for _, event := range events {
		if event.Type == eventType {
			for _, attr := range event.Attributes {
				if string(attr.Key) == attributeKey {
					return string(attr.Value)
				}
			}
		}
	}
	require.Fail(t, "Event attribute not found", "Event: %s, Attribute: %s", eventType, attributeKey)
	return ""
}

// CountEvents counts number of events of specific type
func CountEvents(ctx sdk.Context, eventType string) int {
	events := ctx.EventManager().Events()
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

// ============================================================================
// Assertion Builders (fluent interface)
// ============================================================================

// AccountAssertion provides fluent interface for account assertions
type AccountAssertion struct {
	t       *testing.T
	keeper  *identkeeper.Keeper
	ctx     sdk.Context
	address string
}

func AssertAccount(t *testing.T, keeper *identkeeper.Keeper, ctx sdk.Context, address string) *AccountAssertion {
	return &AccountAssertion{t: t, keeper: keeper, ctx: ctx, address: address}
}

func (a *AccountAssertion) HasRole(role identv1.Role) *AccountAssertion {
	AssertAccountRole(a.t, a.keeper, a.ctx, a.address, role)
	return a
}

func (a *AccountAssertion) IsActive() *AccountAssertion {
	AssertAccountActive(a.t, a.keeper, a.ctx, a.address, true)
	return a
}

func (a *AccountAssertion) IsInactive() *AccountAssertion {
	AssertAccountActive(a.t, a.keeper, a.ctx, a.address, false)
	return a
}

func (a *AccountAssertion) Exists() *AccountAssertion {
	_, err := a.keeper.GetVerifiedAccount(a.ctx, a.address)
	require.NoError(a.t, err, "Account %s should exist", a.address)
	return a
}

func (a *AccountAssertion) NotExists() *AccountAssertion {
	_, err := a.keeper.GetVerifiedAccount(a.ctx, a.address)
	require.Error(a.t, err, "Account %s should not exist", a.address)
	return a
}
