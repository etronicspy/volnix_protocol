package ident

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

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

// AppModuleBasic implements the basic methods for the ident module.
type AppModuleBasic struct{}

var _ module.AppModuleBasic = AppModuleBasic{}

func (AppModuleBasic) Name() string { return identtypes.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

func (AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Interface registration temporarily disabled for integration testing
	// identv1.RegisterInterfaces(registry)
}

func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	bz, _ := json.Marshal(DefaultGenesis())
	return bz
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gen identv1.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gen); err != nil {
		return err
	}
	return Validate(&gen)
}

// Required by module.AppModuleBasic in Cosmos SDK v0.53
func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *gatewayruntime.ServeMux) {}

// AppModule implements an application module for the ident module.
type AppModule struct {
	AppModuleBasic

	keeper *keeper.Keeper
}

var _ module.AppModule = AppModule{}

func NewAppModule(k *keeper.Keeper) AppModule {
	return AppModule{keeper: k}
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	// Services temporarily disabled for integration testing
	// identv1.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(am.keeper))
	// identv1.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gen identv1.GenesisState
	cdc.MustUnmarshalJSON(data, &gen)
	InitGenesis(ctx, am.keeper, &gen)
	return []abci.ValidatorUpdate{}
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gen := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gen)
}

// Marker method required by module.AppModule in Cosmos SDK v0.53
func (AppModule) IsAppModule() {}

// Marker method required by module.AppModule in Cosmos SDK v0.53
func (AppModule) IsOnePerModuleType() {}

// Dependencies wiring
func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, ps paramtypes.Subspace) *keeper.Keeper {
	return keeper.NewKeeper(cdc, key, ps)
}
