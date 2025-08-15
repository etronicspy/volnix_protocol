package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

// RegisterLegacyAminoCodec registers the consensus types on the LegacyAmino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&Validator{}, "volnix/Validator", nil)
	cdc.RegisterConcrete(&Params{}, "volnix/Params", nil)
	cdc.RegisterConcrete(&GenesisState{}, "volnix/GenesisState", nil)
}

// RegisterInterfaces registers the consensus types on the interface registry.
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil)) // Add message types here when they are implemented

}

// RegisterQueryHandlerClient registers the gRPC Gateway routes for consensus module.
func RegisterQueryHandlerClient(ctx context.Context, mux *runtime.ServeMux, client interface{}) error {
	// Register query handlers here when they are implemented
	return nil
}

// NewQueryClient creates a new query client for consensus module.
func NewQueryClient(client interface{}) interface{} {
	// Return query client when implemented
	return nil
}
