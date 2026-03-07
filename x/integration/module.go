package integration

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/volnix-protocol/volnix-protocol/x/integration/keeper"
)

var (
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic defines the basic application module used by the integration module.
type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return "integration"
}

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

func (AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return []byte("{}")
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	if len(bz) == 0 {
		return fmt.Errorf("integration genesis cannot be empty")
	}
	if !json.Valid(bz) {
		return fmt.Errorf("integration genesis is not valid JSON")
	}
	return nil
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}

func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil
}

// AppModule implements an application module for the integration module.
type AppModule struct {
	AppModuleBasic
	keeper *keeper.Keeper
}

func NewAppModule(k *keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
	}
}

func (am AppModule) IsAppModule() {}

func (am AppModule) RegisterServices(cfg module.Configurator) {}

func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {}

// InitGenesis initializes all module health statuses.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	im := am.keeper.GetIntegrationManager()
	for name := range im.Modules {
		im.UpdateModuleHealth(name, 100, "", ctx.BlockTime())
	}
	ctx.Logger().Info("integration module genesis initialized")
}

// ExportGenesis exports the module health map as JSON.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	healthMap := am.keeper.GetAllModulesHealth()

	type exportedHealth struct {
		Name   string `json:"name"`
		Health int64  `json:"health"`
	}
	export := make([]exportedHealth, 0, len(healthMap))
	for name, mod := range healthMap {
		export = append(export, exportedHealth{Name: name, Health: mod.HealthScore})
	}

	bz, err := json.Marshal(export)
	if err != nil {
		ctx.Logger().Error("failed to marshal integration genesis", "error", err)
		return []byte("[]")
	}
	return bz
}

func (AppModule) ConsensusVersion() uint64 { return 1 }

func (am AppModule) BeginBlock(ctx sdk.Context, _ interface{}) {}

func (am AppModule) EndBlock(ctx sdk.Context, _ interface{}) []interface{} {
	return nil
}
