package app

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	consensustypes "github.com/volnix-protocol/volnix-protocol/x/consensus/types"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
	lizenztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
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
	consensustypes.RegisterInterfaces(interfaceRegistry)

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
