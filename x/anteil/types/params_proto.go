package types

import (
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// DefaultProtoParams returns sensible proto-level defaults.
func DefaultProtoParams() *anteilv1.Params {
	d := DefaultParams()
	return d.ToProto()
}

// ToProto converts Go Params to the protobuf Params.
func (p Params) ToProto() *anteilv1.Params {
	return &anteilv1.Params{
		MinOrderSize:                p.MinOrderSize,
		MaxOrderSize:                p.MaxOrderSize,
		TradingFeeRate:              p.TradingFeeRate,
		OrderExpiry:                 durationpb.New(p.OrderExpiry),
		RequireIdentityVerification: p.RequireIdentityVerification,
		AntDenom:                    p.AntDenom,
		MaxOpenOrders:               p.MaxOpenOrders,
		PricePrecision:              p.PricePrecision,

		EpochLength:                  durationpb.New(p.EpochLength),
		EpochCoefficient:             p.EpochCoefficient,
		EpochCoefficientMin:          p.EpochCoefficientMin,
		EpochCoefficientMax:          p.EpochCoefficientMax,
		SupplierEpochAntLimit:        p.SupplierEpochAntLimit,
		CitizenAntDistributionPeriod: durationpb.New(p.CitizenAntDistributionPeriod),
	}
}

// ParamsFromProto converts protobuf Params to Go Params.
func ParamsFromProto(pp *anteilv1.Params) (Params, error) {
	if pp == nil {
		return DefaultParams(), nil
	}

	defaults := DefaultParams()

	orderExpiry := defaults.OrderExpiry
	if pp.OrderExpiry != nil {
		orderExpiry = pp.OrderExpiry.AsDuration()
	}

	epochLength := defaults.EpochLength
	if pp.EpochLength != nil {
		epochLength = pp.EpochLength.AsDuration()
	}

	citizenDistPeriod := defaults.CitizenAntDistributionPeriod
	if pp.CitizenAntDistributionPeriod != nil {
		citizenDistPeriod = pp.CitizenAntDistributionPeriod.AsDuration()
	}

	epochCoefficient := defaults.EpochCoefficient
	if pp.EpochCoefficient != "" {
		epochCoefficient = pp.EpochCoefficient
	}

	epochCoefficientMin := defaults.EpochCoefficientMin
	if pp.EpochCoefficientMin != "" {
		epochCoefficientMin = pp.EpochCoefficientMin
	}

	epochCoefficientMax := defaults.EpochCoefficientMax
	if pp.EpochCoefficientMax != "" {
		epochCoefficientMax = pp.EpochCoefficientMax
	}

	supplierLimit := defaults.SupplierEpochAntLimit
	if pp.SupplierEpochAntLimit != "" {
		supplierLimit = pp.SupplierEpochAntLimit
	}

	return Params{
		MinOrderSize:                pp.MinOrderSize,
		MaxOrderSize:                pp.MaxOrderSize,
		TradingFeeRate:              pp.TradingFeeRate,
		OrderExpiry:                 orderExpiry,
		RequireIdentityVerification: pp.RequireIdentityVerification,
		AntDenom:                    pp.AntDenom,
		MaxOpenOrders:               pp.MaxOpenOrders,
		PricePrecision:              pp.PricePrecision,

		EpochLength:                  epochLength,
		EpochCoefficient:             epochCoefficient,
		EpochCoefficientMin:          epochCoefficientMin,
		EpochCoefficientMax:          epochCoefficientMax,
		SupplierEpochAntLimit:        supplierLimit,
		CitizenAntDistributionPeriod: citizenDistPeriod,
	}, nil
}
