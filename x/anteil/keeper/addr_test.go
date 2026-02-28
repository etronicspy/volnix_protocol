package keeper_test

import sdk "github.com/cosmos/cosmos-sdk/types"

func init() {
	cfg := sdk.GetConfig()
	if cfg.GetBech32AccountAddrPrefix() != "cosmos" {
		cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
	}
}

func mustAddr(hex string) string {
	a, err := sdk.AccAddressFromHexUnsafe(hex)
	if err != nil {
		panic(err)
	}
	return a.String()
}

// Test addresses for anteil keeper tests (valid bech32)
var (
	addrTest       = mustAddr("0000000000000000000000000000000000000001")
	addrBuyer      = mustAddr("0000000000000000000000000000000000000002")
	addrSeller     = mustAddr("0000000000000000000000000000000000000003")
	addrOwner1     = mustAddr("0000000000000000000000000000000000000004")
	addrOwner2     = mustAddr("0000000000000000000000000000000000000005")
	addrBidder     = mustAddr("0000000000000000000000000000000000000006")
	addrBidder1    = mustAddr("0000000000000000000000000000000000000007")
	addrBidder2    = mustAddr("0000000000000000000000000000000000000008")
	addrNotFound   = mustAddr("0000000000000000000000000000000000000009")
	addrBuyer1     = mustAddr("000000000000000000000000000000000000000a")
	addrBuyer2     = mustAddr("000000000000000000000000000000000000000b")
	addrNonexistent = mustAddr("000000000000000000000000000000000000000c")
	addrCitizen     = mustAddr("000000000000000000000000000000000000000d")
	addrValidator   = mustAddr("000000000000000000000000000000000000000e")
	addrGuest       = mustAddr("000000000000000000000000000000000000000f")
	addrInactive    = mustAddr("0000000000000000000000000000000000000010")
	addrCitizen1    = mustAddr("0000000000000000000000000000000000000011")
	addrCitizen2    = mustAddr("0000000000000000000000000000000000000012")
	addrTest2       = mustAddr("0000000000000000000000000000000000000013")
	addrNewUser     = mustAddr("0000000000000000000000000000000000000014")
)
