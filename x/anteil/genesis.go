package anteil

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	anteilv1 "github.com/helvetia-protocol/helvetia-protocol/proto/gen/go/helvetia/anteil/v1"
	"github.com/helvetia-protocol/helvetia-protocol/x/anteil/keeper"
	atypes "github.com/helvetia-protocol/helvetia-protocol/x/anteil/types"
)

func DefaultGenesis() *anteilv1.GenesisState {
	return &anteilv1.GenesisState{Params: atypes.DefaultParams().ToProto()}
}

func Validate(gen *anteilv1.GenesisState) error {
	if gen == nil {
		_, err := atypes.ParamsFromProto(&anteilv1.Params{})
		return err
	}
	p, err := atypes.ParamsFromProto(gen.Params)
	if err != nil {
		return err
	}
	return p.Validate()
}

func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState *anteilv1.GenesisState) {
	if genState == nil {
		genState = DefaultGenesis()
	}
	p, _ := atypes.ParamsFromProto(genState.Params)
	k.SetParams(ctx, p)
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *anteilv1.GenesisState {
	params := k.GetParams(ctx)
	return &anteilv1.GenesisState{Params: params.ToProto()}
}
