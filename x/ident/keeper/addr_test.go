package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

func mustAddr(hex string) string {
	a, err := sdk.AccAddressFromHexUnsafe(hex)
	if err != nil {
		panic(err)
	}
	return a.String()
}
