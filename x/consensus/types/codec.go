package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterLegacyAminoCodec registers the consensus types on the LegacyAmino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&Validator{}, "volnix/Validator", nil)
	cdc.RegisterConcrete(&Params{}, "volnix/Params", nil)
	cdc.RegisterConcrete(&GenesisState{}, "volnix/GenesisState", nil)
}

// RegisterInterfaces registers the consensus types on the interface registry.
func RegisterInterfaces(registry types.InterfaceRegistry) {
	// Temporarily disabled for CometBFT integration
}
