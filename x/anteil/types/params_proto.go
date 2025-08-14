package types

import (
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

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
	}
}

func ParamsFromProto(pp *anteilv1.Params) (Params, error) {
	if pp == nil {
		return DefaultParams(), nil
	}
	return Params{
		MinAntAmount:                pp.MinAntAmount,
		MaxAntAmount:                pp.MaxAntAmount,
		TradingFeeRate:              pp.TradingFeeRate,
		MinOrderSize:                pp.MinOrderSize,
		MaxOrderSize:                pp.MaxOrderSize,
		OrderExpiry:                 pp.OrderExpiry.AsDuration(),
		RequireIdentityVerification: pp.RequireIdentityVerification,
		AntDenom:                    pp.AntDenom,
		MaxOpenOrders:               pp.MaxOpenOrders,
		PricePrecision:              pp.PricePrecision,
	}, nil
}
