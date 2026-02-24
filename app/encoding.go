package app

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/cosmos/cosmos-sdk/x/auth"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/volnix-protocol/volnix-protocol/x/consensus"
	"github.com/volnix-protocol/volnix-protocol/x/ident"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz"
)

// EncodingConfig specifies the concrete encoding types to use for a given app.
type EncodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          TxConfig
	LegacyAmino       *codec.LegacyAmino
}

// TxConfig defines minimal transaction configuration
type TxConfig struct {
	TxDecoder sdk.TxDecoder
	TxEncoder sdk.TxEncoder
}

// MakeEncodingConfig creates an EncodingConfig with real protobuf tx codec
// and all module message types registered.
func MakeEncodingConfig() EncodingConfig {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	protoCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()

	// Register standard SDK types
	authtypes.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)

	// Register custom module types so transactions can be decoded
	basicManager := []interface{ RegisterInterfaces(codectypes.InterfaceRegistry) }{
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		ident.AppModuleBasic{},
		lizenz.AppModuleBasic{},
		consensus.ConsensusAppModuleBasic{},
	}
	for _, m := range basicManager {
		m.RegisterInterfaces(interfaceRegistry)
	}

	// Real protobuf tx encoder/decoder for transaction processing
	sdkTxConfig := authtx.NewTxConfig(protoCodec, []signingtypes.SignMode{
		signingtypes.SignMode_SIGN_MODE_DIRECT,
	})

	txConfig := TxConfig{
		TxDecoder: sdkTxConfig.TxDecoder(),
		TxEncoder: sdkTxConfig.TxEncoder(),
	}

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             protoCodec,
		TxConfig:          txConfig,
		LegacyAmino:       legacyAmino,
	}
}