package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
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
func RegisterInterfaces(registry types.InterfaceRegistry) {
	// Register all Msg types
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&consensusv1.MsgSelectBlockCreator{},
		&consensusv1.MsgUpdateConsensusState{},
		&consensusv1.MsgSetValidatorWeight{},
		&consensusv1.MsgProcessHalving{},
		&consensusv1.MsgSelectBlockProducer{},
		&consensusv1.MsgCalculateBlockTime{},
		&consensusv1.MsgCommitBid{},
		&consensusv1.MsgRevealBid{},
	)

	// Register all MsgResponse types
	registry.RegisterImplementations((*txtypes.MsgResponse)(nil),
		&consensusv1.MsgSelectBlockCreatorResponse{},
		&consensusv1.MsgUpdateConsensusStateResponse{},
		&consensusv1.MsgSetValidatorWeightResponse{},
		&consensusv1.MsgProcessHalvingResponse{},
		&consensusv1.MsgSelectBlockProducerResponse{},
		&consensusv1.MsgCalculateBlockTimeResponse{},
		&consensusv1.MsgCommitBidResponse{},
		&consensusv1.MsgRevealBidResponse{},
	)
}
