package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
)

// RegisterInterfaces registers the governance module's Msg and MsgResponse types on the interface registry.
func RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	reg.RegisterImplementations((*sdk.Msg)(nil),
		&governancev1.MsgSubmitProposal{},
		&governancev1.MsgVote{},
		&governancev1.MsgExecuteProposal{},
	)
	reg.RegisterImplementations((*txtypes.MsgResponse)(nil),
		&governancev1.MsgSubmitProposalResponse{},
		&governancev1.MsgVoteResponse{},
		&governancev1.MsgExecuteProposalResponse{},
	)
}
