package consensus

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

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/client/cli"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

// ConsensusAppModuleBasic implements the AppModuleBasic interface for the consensus module.
type ConsensusAppModuleBasic struct{}

// Name returns the consensus module's name.
func (ConsensusAppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the consensus module's types on the LegacyAmino codec.
func (ConsensusAppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the consensus module's interface types.
func (ConsensusAppModuleBasic) RegisterInterfaces(reg codectypes.InterfaceRegistry) {
	// Interface registration temporarily disabled for integration testing
	// types.RegisterInterfaces(reg)
	// consensusv1.RegisterInterfaces(reg)
}

// DefaultGenesis returns default genesis state as raw bytes for the consensus module.
func (ConsensusAppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the consensus module.
func (ConsensusAppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return types.ValidateGenesis(&genState)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the consensus module.
// Note: In Cosmos SDK v0.53, BaseApp automatically registers routes from proto annotations
// (google.api.http). The routes are:
// - GET /volnix/consensus/v1/params
// - GET /volnix/consensus/v1/validators
// This method is required by the interface but BaseApp handles the actual registration.
func (ConsensusAppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	// BaseApp automatically registers routes from proto annotations
	// If manual registration is needed, it would be done here using:
	// consensusv1.RegisterQueryHandlerClient(context.Background(), mux, consensusv1.NewQueryClient(clientCtx))
	// However, the generated code uses grpc-gateway/v2 while Cosmos SDK uses v1,
	// so we rely on BaseApp's automatic registration from proto annotations.
}

// GetTxCmd returns the root tx command for the consensus module.
func (ConsensusAppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the root query command for the consensus module.
func (ConsensusAppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// ConsensusAppModule implements the AppModule interface for the consensus module.
type ConsensusAppModule struct {
	ConsensusAppModuleBasic

	keeper keeper.Keeper
}

// NewConsensusAppModule creates a new ConsensusAppModule object.
func NewConsensusAppModule(cdc codec.Codec, keeper keeper.Keeper) ConsensusAppModule {
	return ConsensusAppModule{
		ConsensusAppModuleBasic: ConsensusAppModuleBasic{},
		keeper:                  keeper,
	}
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries.
func (am ConsensusAppModule) RegisterServices(cfg module.Configurator) {
	consensusv1.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServer(am.keeper))
	consensusv1.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))
}

// RegisterInvariants registers the consensus module invariants.
func (am ConsensusAppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	// Register invariants if needed
}

// InitGenesis performs genesis initialization for the consensus module.
func (am ConsensusAppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	cdc.MustUnmarshalJSON(gs, &genState)

	am.keeper.InitGenesis(ctx, &genState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the consensus module.
func (am ConsensusAppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(&genState)
}

// IsAppModule implements the module.AppModule interface.
func (am ConsensusAppModule) IsAppModule() {}

// IsOnePerModuleType implements the module.AppModule interface.
func (am ConsensusAppModule) IsOnePerModuleType() {}

// ConsensusAppModule implements the module.AppModule interface.
var _ module.AppModule = ConsensusAppModule{}

// ConsensusAppModuleBasic implements the module.AppModuleBasic interface.
var _ module.AppModuleBasic = ConsensusAppModuleBasic{}
