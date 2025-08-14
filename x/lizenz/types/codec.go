package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
)

func RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	reg.RegisterImplementations((*sdk.Msg)(nil),
		&lizenzv1.MsgActivateLZN{},
		&lizenzv1.MsgDeactivateLZN{},
	)
	reg.RegisterImplementations((*txtypes.MsgResponse)(nil),
		&lizenzv1.MsgActivateLZNResponse{},
		&lizenzv1.MsgDeactivateLZNResponse{},
	)
}
