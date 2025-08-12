package app

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	anteiltypes "github.com/helvetia-protocol/helvetia-protocol/x/anteil/types"
	identtypes "github.com/helvetia-protocol/helvetia-protocol/x/ident/types"
	lizenztypes "github.com/helvetia-protocol/helvetia-protocol/x/lizenz/types"
)

// EncodingConfig bundles the app-wide codec and interface registry
type EncodingConfig struct {
	InterfaceRegistry cdctypes.InterfaceRegistry
	Codec             codec.Codec
	LegacyAmino       *codec.LegacyAmino
	TxConfig          client.TxConfig
}

// MakeEncodingConfig constructs EncodingConfig and registers module interfaces
func MakeEncodingConfig() EncodingConfig {
	interfaceRegistry := cdctypes.NewInterfaceRegistry()

	// Register interfaces for custom modules
	identtypes.RegisterInterfaces(interfaceRegistry)
	lizenztypes.RegisterInterfaces(interfaceRegistry)
	anteiltypes.RegisterInterfaces(interfaceRegistry)

	// Create codecs
	protoCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()

	// TxConfig for signing and tx building
	txConfig := authtx.NewTxConfig(protoCodec, authtx.DefaultSignModes)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             protoCodec,
		LegacyAmino:       legacyAmino,
		TxConfig:          txConfig,
	}
}
