package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	atypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	ps       paramtypes.Subspace
}

func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, ps paramtypes.Subspace) Keeper {
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(atypes.ParamKeyTable())
	}
	return Keeper{cdc: cdc, storeKey: key, ps: ps}
}

func (k Keeper) GetParams(ctx sdk.Context) (params atypes.Params) {
	k.ps.GetParamSet(ctx, &params)
	return
}

func (k Keeper) SetParams(ctx sdk.Context, params atypes.Params) {
	k.ps.SetParamSet(ctx, &params)
}
