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
		&anteilv1.MsgUpdateOrder{},
		&anteilv1.MsgPlaceBid{},
		&anteilv1.MsgSettleAuction{},
		&anteilv1.MsgRegisterMarketMaker{},
		&anteilv1.MsgProvideLiquidity{},
		&anteilv1.MsgWithdrawLiquidity{},
		&anteilv1.MsgStakeANT{},
		&anteilv1.MsgUnstakeANTResponse{},
		&anteilv1.MsgClaimRewards{},
	)
	reg.RegisterImplementations((*txtypes.MsgResponse)(nil),
		&anteilv1.MsgPlaceOrderResponse{},
		&anteilv1.MsgCancelOrderResponse{},
		&anteilv1.MsgUpdateOrderResponse{},
		&anteilv1.MsgPlaceBidResponse{},
		&anteilv1.MsgSettleAuctionResponse{},
		&anteilv1.MsgRegisterMarketMakerResponse{},
		&anteilv1.MsgProvideLiquidityResponse{},
		&anteilv1.MsgWithdrawLiquidityResponse{},
		&anteilv1.MsgStakeANTResponse{},
		&anteilv1.MsgUnstakeANTResponse{},
		&anteilv1.MsgClaimRewardsResponse{},
	)
}
