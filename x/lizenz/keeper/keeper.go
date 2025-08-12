package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	lztypes "github.com/helvetia-protocol/helvetia-protocol/x/lizenz/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	ps       paramtypes.Subspace
}

func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, ps paramtypes.Subspace) Keeper {
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(lztypes.ParamKeyTable())
	}
	return Keeper{cdc: cdc, storeKey: key, ps: ps}
}

func (k Keeper) GetParams(ctx sdk.Context) (params lztypes.Params) {
	k.ps.GetParamSet(ctx, &params)
	return
}

func (k Keeper) SetParams(ctx sdk.Context, params lztypes.Params) {
	k.ps.SetParamSet(ctx, &params)
}
