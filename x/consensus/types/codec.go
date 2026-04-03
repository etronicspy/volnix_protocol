package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
)

// RegisterLegacyAminoCodec registers the consensus types on the LegacyAmino codec.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&Validator{}, "volnix/Validator", nil)
	cdc.RegisterConcrete(&Params{}, "volnix/Params", nil)
	cdc.RegisterConcrete(&GenesisState{}, "volnix/GenesisState", nil)
}

// RegisterInterfaces registers the consensus types on the interface registry.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&consensusv1.MsgDeclarePerHeightBurn{},
		&consensusv1.MsgUpdateConsensusState{},
		&consensusv1.MsgSetValidatorWeight{},
		&consensusv1.MsgRegisterConsensusMapping{},
	)

	registry.RegisterImplementations((*txtypes.MsgResponse)(nil),
		&consensusv1.MsgDeclarePerHeightBurnResponse{},
		&consensusv1.MsgUpdateConsensusStateResponse{},
		&consensusv1.MsgSetValidatorWeightResponse{},
		&consensusv1.MsgRegisterConsensusMappingResponse{},
	)
}
