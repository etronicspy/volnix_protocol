package types

import (
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ToProto converts Go Params to the protobuf Params.
// NOTE: After running `buf generate`, add fields 16-21 (citizen_ant_*, market_making_*)
// from the updated types.proto to keep proto and Go params in sync.
func (p Params) ToProto() *anteilv1.Params {
	return &anteilv1.Params{
		MinAntAmount:                p.MinAntAmount,
		MaxAntAmount:                p.MaxAntAmount,
		TradingFeeRate:              p.TradingFeeRate,
		MinOrderSize:                p.MinOrderSize,
		MaxOrderSize:                p.MaxOrderSize,
		OrderExpiry:                 durationpb.New(p.OrderExpiry),
		RequireIdentityVerification: p.RequireIdentityVerification,
		AntDenom:                    p.AntDenom,
		MaxOpenOrders:               p.MaxOpenOrders,
		PricePrecision:              p.PricePrecision,
		MarketMakerRewardRate:       p.MarketMakerRewardRate,
		StakingRewardRate:           p.StakingRewardRate,
		LiquidityPoolFee:            p.LiquidityPoolFee,
		MaxSlippage:                 p.MaxSlippage,
		MinLiquidityThreshold:       p.MinLiquidityThreshold,
	}
}

// ParamsFromProto converts protobuf Params to Go Params.
// Fields not present in proto (citizen_ant_*, market_making_*) keep Go defaults.
func ParamsFromProto(pp *anteilv1.Params) (Params, error) {
	if pp == nil {
		return DefaultParams(), nil
	}

	defaults := DefaultParams()

	orderExpiry := defaults.OrderExpiry
	if pp.OrderExpiry != nil {
		orderExpiry = pp.OrderExpiry.AsDuration()
	}

	return Params{
		MinAntAmount:                pp.MinAntAmount,
		MaxAntAmount:                pp.MaxAntAmount,
		TradingFeeRate:              pp.TradingFeeRate,
		MinOrderSize:                pp.MinOrderSize,
		MaxOrderSize:                pp.MaxOrderSize,
		OrderExpiry:                 orderExpiry,
		RequireIdentityVerification: pp.RequireIdentityVerification,
		AntDenom:                    pp.AntDenom,
		MaxOpenOrders:               pp.MaxOpenOrders,
		PricePrecision:              pp.PricePrecision,
		MarketMakerRewardRate:       pp.MarketMakerRewardRate,
		StakingRewardRate:           pp.StakingRewardRate,
		LiquidityPoolFee:            pp.LiquidityPoolFee,
		MaxSlippage:                 pp.MaxSlippage,
		MinLiquidityThreshold:       pp.MinLiquidityThreshold,
		CitizenAntRewardRate:        defaults.CitizenAntRewardRate,
		CitizenAntAccumulationLimit: defaults.CitizenAntAccumulationLimit,
		CitizenAntDistributionPeriod: defaults.CitizenAntDistributionPeriod,
		MarketMakingBuyDiscount:     defaults.MarketMakingBuyDiscount,
		MarketMakingSellPremium:     defaults.MarketMakingSellPremium,
		MarketMakingOrderSize:       defaults.MarketMakingOrderSize,
	}, nil
}
