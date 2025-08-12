package types

import (
	anteilv1 "github.com/helvetia-protocol/helvetia-protocol/proto/gen/go/helvetia/anteil/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

func (p Params) ToProto() *anteilv1.Params {
	return &anteilv1.Params{
		MaxOrderAmount: p.MaxOrderAmount,
		MinOrderAmount: p.MinOrderAmount,
		TradingFee:     p.TradingFee,
		AuctionPeriod:  durationpb.New(p.AuctionPeriod),
	}
}

func ParamsFromProto(pp *anteilv1.Params) (Params, error) {
	if pp == nil {
		return DefaultParams(), nil
	}
	return Params{
		MaxOrderAmount: pp.MaxOrderAmount,
		MinOrderAmount: pp.MinOrderAmount,
		TradingFee:     pp.TradingFee,
		AuctionPeriod:  pp.AuctionPeriod.AsDuration(),
	}, nil
}
