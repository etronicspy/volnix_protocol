package anteil

import (
	"encoding/json"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	atypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

type AppModuleBasic struct{}

var _ module.AppModuleBasic = AppModuleBasic{}

func (AppModuleBasic) Name() string                                  { return atypes.ModuleName }
func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}
func (AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Register message interfaces
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&anteilv1.MsgPlaceOrder{},
		&anteilv1.MsgCancelOrder{},
		&anteilv1.MsgUpdateOrder{},
		&anteilv1.MsgPlaceBid{},
		&anteilv1.MsgSettleAuction{},
	)
}

func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	bz, _ := json.Marshal(DefaultGenesis())
	return bz
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gen anteilv1.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gen); err != nil {
		return err
	}
	return Validate(&gen)
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *gatewayruntime.ServeMux) {}

type AppModule struct {
	AppModuleBasic
	keeper *keeper.Keeper
}

var _ module.AppModule = AppModule{}

func NewAppModule(k *keeper.Keeper) AppModule { return AppModule{keeper: k} }

func (am AppModule) RegisterServices(cfg module.Configurator) {
	anteilv1.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(am.keeper))
	anteilv1.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gen anteilv1.GenesisState
	cdc.MustUnmarshalJSON(data, &gen)
	InitGenesis(ctx, am.keeper, &gen)
	return []abci.ValidatorUpdate{}
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gen := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gen)
}

func (AppModule) IsAppModule()        {}
func (AppModule) IsOnePerModuleType() {}

func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, ps paramtypes.Subspace) *keeper.Keeper {
	return keeper.NewKeeper(cdc, key, ps)
}
