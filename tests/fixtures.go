package tests

import (
	"fmt"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	lizenztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

var ensureBech32Once sync.Once

// EnsureBech32Config sets the bech32 prefix for test addresses.
// Call from TestMain or SetupSuite instead of init() to avoid mutating global state.
func EnsureBech32Config() {
	ensureBech32Once.Do(func() {
		cfg := sdk.GetConfig()
		if cfg.GetBech32AccountAddrPrefix() != "cosmos" {
			cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
		}
	})
}

func init() {
	EnsureBech32Config()
}

// mustBech32 creates a valid bech32 address from hex (20 bytes = 40 hex chars)
// Ensures cosmos prefix is set before creating address (required for package init order)
func mustBech32(hexStr string) string {
	cfg := sdk.GetConfig()
	if cfg.GetBech32AccountAddrPrefix() != "cosmos" {
		cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
	}
	addr, err := sdk.AccAddressFromHexUnsafe(hexStr)
	if err != nil {
		panic(err)
	}
	return addr.String()
}

// TestAddresses provides standard test addresses (valid bech32)
var TestAddresses = struct {
	Guest      string
	Citizen    string
	Citizen2   string
	Citizen3   string
	Validator  string
	Validator2 string
	Inactive   string
	Buyer      string
	Seller     string
	Source     string
	Target     string
	Target2    string
	Target3    string
	Test1      string
	Test2      string
	Test3      string
	Invalid    string
}{
	Guest:      mustBech32("0000000000000000000000000000000000000001"),
	Citizen:    mustBech32("0000000000000000000000000000000000000002"),
	Citizen2:   mustBech32("0000000000000000000000000000000000000003"),
	Citizen3:   mustBech32("0000000000000000000000000000000000000013"),
	Validator:  mustBech32("0000000000000000000000000000000000000004"),
	Validator2: mustBech32("0000000000000000000000000000000000000005"),
	Inactive:   mustBech32("0000000000000000000000000000000000000006"),
	Buyer:      mustBech32("0000000000000000000000000000000000000007"),
	Seller:     mustBech32("0000000000000000000000000000000000000008"),
	Source:     mustBech32("0000000000000000000000000000000000000009"),
	Target:     mustBech32("000000000000000000000000000000000000000a"),
	Target2:    mustBech32("000000000000000000000000000000000000000b"),
	Target3:    mustBech32("000000000000000000000000000000000000000c"),
	Test1:      mustBech32("0000000000000000000000000000000000000010"),
	Test2:      mustBech32("0000000000000000000000000000000000000011"),
	Test3:      mustBech32("0000000000000000000000000000000000000012"),
	Invalid:    "invalid_address", // Not valid bech32 - for negative tests
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

// GenerateTestAddresses generates n unique valid bech32 test addresses
func GenerateTestAddresses(_ string, count int) []string {
	addresses := make([]string, count)
	for i := 0; i < count; i++ {
		hexStr := fmt.Sprintf("000000000000000000000000000000000000%04x", i+0x100)
		addresses[i] = mustBech32(hexStr)
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
