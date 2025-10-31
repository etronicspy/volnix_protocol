package app

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EncodingConfig specifies the concrete encoding types to use for a given app.
type EncodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          TxConfig
	LegacyAmino       *codec.LegacyAmino
}

// TxConfig defines minimal transaction configuration
type TxConfig struct {
	TxDecoder sdk.TxDecoder
	TxEncoder sdk.TxEncoder
}

// MakeEncodingConfig creates an EncodingConfig for the app.
func MakeEncodingConfig() EncodingConfig {
	interfaceRegistry := types.NewInterfaceRegistry()
	protoCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	
	// Create basic tx encoder/decoder
	txConfig := TxConfig{
		TxDecoder: func(txBytes []byte) (sdk.Tx, error) {
			// Basic implementation - in real app would decode protobuf
			return nil, nil
		},
		TxEncoder: func(tx sdk.Tx) ([]byte, error) {
			// Basic implementation - in real app would encode protobuf
			return []byte{}, nil
		},
	}

	// Register standard interfaces
	std.RegisterLegacyAminoCodec(legacyAmino)
	std.RegisterInterfaces(interfaceRegistry)

	// Register module interfaces
	ModuleBasics.RegisterLegacyAminoCodec(legacyAmino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             protoCodec,
		TxConfig:          txConfig,
		LegacyAmino:       legacyAmino,
	}
}