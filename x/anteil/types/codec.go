package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
)

func RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	reg.RegisterImplementations((*sdk.Msg)(nil),
		&anteilv1.MsgPlaceOrder{},
		&anteilv1.MsgCancelOrder{},
		&anteilv1.MsgPlaceBid{},
	)
	reg.RegisterImplementations((*txtypes.MsgResponse)(nil),
		&anteilv1.MsgPlaceOrderResponse{},
		&anteilv1.MsgCancelOrderResponse{},
		&anteilv1.MsgPlaceBidResponse{},
	)
}
