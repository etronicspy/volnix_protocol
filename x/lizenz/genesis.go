package lizenz

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
	lztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

func DefaultGenesis() *lizenzv1.GenesisState {
	return &lizenzv1.GenesisState{
		Params:    lztypes.DefaultParams().ToProto(),
		Activated: []*lizenzv1.ActivatedLizenz{},
	}
}

func Validate(gen *lizenzv1.GenesisState) error {
	if gen == nil {
		_, err := lztypes.ParamsFromProto((&lizenzv1.Params{}))
		return err
	}
	p, err := lztypes.ParamsFromProto(gen.Params)
	if err != nil {
		return err
	}
	return p.Validate()
}

func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState *lizenzv1.GenesisState) {
	if genState == nil {
		genState = DefaultGenesis()
	}
	p, _ := lztypes.ParamsFromProto(genState.Params)
	k.SetParams(ctx, p)
}

func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *lizenzv1.GenesisState {
	params := k.GetParams(ctx)
	return &lizenzv1.GenesisState{
		Params:    params.ToProto(),
		Activated: []*lizenzv1.ActivatedLizenz{},
	}
}
