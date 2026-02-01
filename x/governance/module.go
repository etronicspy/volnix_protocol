package governance

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
	"github.com/volnix-protocol/volnix-protocol/x/governance/client/cli"
	"github.com/volnix-protocol/volnix-protocol/x/governance/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/governance/types"
)

// AppModuleBasic implements the AppModuleBasic interface for the governance module
type AppModuleBasic struct{}

var _ module.AppModuleBasic = AppModuleBasic{}

// Name returns the governance module's name
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the governance module's types on the LegacyAmino codec.
// No legacy amino message types are used yet; register here when adding CLI/tx encoding.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// No types to register until legacy amino tx is used for governance messages
}

// RegisterInterfaces registers the governance module's interface types
func (AppModuleBasic) RegisterInterfaces(reg codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns default genesis state as raw bytes for the governance module
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := DefaultGenesis()
	bz, err := json.Marshal(genState)
	if err != nil {
		panic(fmt.Errorf("failed to marshal governance genesis state: %w", err))
	}
	return bz
}

// ValidateGenesis performs genesis state validation for the governance module
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := json.Unmarshal(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return ValidateGenesis(&genState)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the governance module
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	// BaseApp automatically registers routes from proto annotations
	// If manual registration is needed, it would be done here using:
	// governancev1.RegisterQueryHandlerClient(context.Background(), mux, governancev1.NewQueryClient(clientCtx))
}

// GetTxCmd returns the root tx command for the governance module
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the root query command for the governance module
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// AppModule implements the AppModule interface for the governance module
type AppModule struct {
	AppModuleBasic
	keeper *keeper.Keeper
}

var _ module.AppModule = AppModule{}

// NewAppModule creates a new AppModule object
func NewAppModule(keeper *keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:          keeper,
	}
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(cfg module.Configurator) {
	governancev1.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(am.keeper))
	governancev1.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))
}

// RegisterInvariants registers the governance module invariants
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	// Register invariants if needed
}

// InitGenesis performs genesis initialization for the governance module
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	if err := json.Unmarshal(gs, &genState); err != nil {
		panic(fmt.Errorf("failed to unmarshal governance genesis state: %w", err))
	}
	InitGenesis(ctx, am.keeper, &genState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the governance module
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	bz, err := json.Marshal(genState)
	if err != nil {
		panic(fmt.Errorf("failed to marshal governance genesis state: %w", err))
	}
	return bz
}

// IsAppModule implements the module.AppModule interface
func (am AppModule) IsAppModule() {}

// IsOnePerModuleType implements the module.AppModule interface
func (am AppModule) IsOnePerModuleType() {}

