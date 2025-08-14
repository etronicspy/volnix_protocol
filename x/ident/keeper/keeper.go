package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	ps       paramtypes.Subspace

	// dependencies from other modules can be added here later (account, bank, etc.)
}

func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, ps paramtypes.Subspace) Keeper {
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(identtypes.ParamKeyTable())
	}
	return Keeper{
		cdc:      cdc,
		storeKey: key,
		ps:       ps,
	}
}

func (k Keeper) GetParams(ctx sdk.Context) (params identtypes.Params) {
	k.ps.GetParamSet(ctx, &params)
	return
}

func (k Keeper) SetParams(ctx sdk.Context, params identtypes.Params) {
	k.ps.SetParamSet(ctx, &params)
}
