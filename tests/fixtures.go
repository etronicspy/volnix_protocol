package tests

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	lizenztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

// TestAddresses provides standard test addresses for consistency
var TestAddresses = struct {
	Guest      string
	Citizen    string
	Citizen2   string
	Validator  string
	Validator2 string
	Inactive   string
}{
	Guest:      "cosmos1guest",
	Citizen:    "cosmos1citizen",
	Citizen2:   "cosmos1citizen2",
	Validator:  "cosmos1validator",
	Validator2: "cosmos1validator2",
	Inactive:   "cosmos1inactive",
}

// TestHashes provides standard identity hashes for consistency
var TestHashes = struct {
	Valid1   string
	Valid2   string
	Valid3   string
	Valid4   string
	Empty    string
	Short    string
	Duplicate string
}{
	Valid1:   "hash123",
	Valid2:   "hash456",
	Valid3:   "hash789",
	Valid4:   "hashabc",
	Empty:    "",
	Short:    "short",
	Duplicate: "duplicate",
}

// TestAmounts provides standard amounts for testing
var TestAmounts = struct {
	Small  string
	Medium string
	Large  string
	Zero   string
}{
	Small:  "1000000",      // 1 token
	Medium: "10000000",     // 10 tokens
	Large:  "100000000",    // 100 tokens
	Zero:   "0",
}

// NewTestVerifiedAccount creates a test verified account with standard values
func NewTestVerifiedAccount(role identv1.Role) *identv1.VerifiedAccount {
	var address, hash string
	switch role {
	case identv1.Role_ROLE_CITIZEN:
		address = TestAddresses.Citizen
		hash = TestHashes.Valid1
	case identv1.Role_ROLE_VALIDATOR:
		address = TestAddresses.Validator
		hash = TestHashes.Valid2
	default: // GUEST
		address = TestAddresses.Guest
		hash = TestHashes.Valid3
	}
	
	return identtypes.NewVerifiedAccount(address, role, hash)
}

// NewTestVerifiedAccountCustom creates a test account with custom address and hash
func NewTestVerifiedAccountCustom(address string, role identv1.Role, hash string) *identv1.VerifiedAccount {
	return identtypes.NewVerifiedAccount(address, role, hash)
}

// NewTestOrder creates a test order with standard values
func NewTestOrder(owner string, orderType anteilv1.OrderType, orderSide anteilv1.OrderSide) *anteilv1.Order {
	return anteiltypes.NewOrder(
		owner,
		orderType,
		orderSide,
		TestAmounts.Medium,
		"1.5",
		TestHashes.Valid1,
	)
}

// NewTestAuction creates a test auction with standard values
func NewTestAuction(blockHeight uint64) *anteilv1.Auction {
	return anteiltypes.NewAuction(blockHeight, TestAmounts.Medium, "10.0")
}

// NewTestAuctionCustom creates a test auction with custom values
func NewTestAuctionCustom(blockHeight uint64, antAmount string, reservePrice string) *anteilv1.Auction {
	return anteiltypes.NewAuction(blockHeight, antAmount, reservePrice)
}

// NewTestUserPosition creates a test user position
func NewTestUserPosition(owner string, antBalance string) *anteilv1.UserPosition {
	return anteiltypes.NewUserPosition(owner, antBalance)
}

// NewTestLizenz creates a test lizenz (ActivatedLizenz)
func NewTestLizenz(owner string, amount string) *lizenzv1.ActivatedLizenz {
	return lizenztypes.NewLizenz(owner, amount, "hash-lizenz")
}

// NewTestTrade creates a test trade
func NewTestTrade(buyer, seller string, antAmount, price string) *anteilv1.Trade {
	return anteiltypes.NewTrade(
		"buy-order-1",
		"sell-order-1",
		buyer,
		seller,
		antAmount,
		price,
		TestHashes.Valid1,
	)
}

// TimeNow returns current timestamp for consistency in tests
func TimeNow() *timestamppb.Timestamp {
	return timestamppb.Now()
}

// TimePast returns a timestamp in the past
func TimePast(duration time.Duration) *timestamppb.Timestamp {
	return timestamppb.New(time.Now().Add(-duration))
}

// TimeFuture returns a timestamp in the future
func TimeFuture(duration time.Duration) *timestamppb.Timestamp {
	return timestamppb.New(time.Now().Add(duration))
}

// RepeatRole creates a slice of the same role repeated n times
func RepeatRole(role identv1.Role, count int) []identv1.Role {
	roles := make([]identv1.Role, count)
	for i := 0; i < count; i++ {
		roles[i] = role
	}
	return roles
}

// GenerateTestAddresses generates n unique test addresses
func GenerateTestAddresses(prefix string, count int) []string {
	addresses := make([]string, count)
	for i := 0; i < count; i++ {
		addresses[i] = fmt.Sprintf("cosmos1%s%d", prefix, i)
	}
	return addresses
}

// GenerateTestHashes generates n unique test hashes
func GenerateTestHashes(prefix string, count int) []string {
	hashes := make([]string, count)
	for i := 0; i < count; i++ {
		hashes[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return hashes
}
