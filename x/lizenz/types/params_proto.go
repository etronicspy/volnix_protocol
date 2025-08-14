package types

import (
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

func (p Params) ToProto() *lizenzv1.Params {
	return &lizenzv1.Params{
		MaxActivatedPerValidator:    p.MaxActivatedPerValidator,
		ActivityCoefficient:         p.ActivityCoefficient,
		DeactivationPeriod:          durationpb.New(p.DeactivationPeriod),
		InactivityPeriod:            durationpb.New(p.InactivityPeriod),
		MinLznAmount:                p.MinLznAmount,
		MaxLznAmount:                p.MaxLznAmount,
		RequireIdentityVerification: p.RequireIdentityVerification,
		LznDenom:                    p.LznDenom,
	}
}

func ParamsFromProto(pp *lizenzv1.Params) (Params, error) {
	if pp == nil {
		return DefaultParams(), nil
	}
	return Params{
		MaxActivatedPerValidator:    pp.MaxActivatedPerValidator,
		ActivityCoefficient:         pp.ActivityCoefficient,
		DeactivationPeriod:          pp.DeactivationPeriod.AsDuration(),
		InactivityPeriod:            pp.InactivityPeriod.AsDuration(),
		MinLznAmount:                pp.MinLznAmount,
		MaxLznAmount:                pp.MaxLznAmount,
		RequireIdentityVerification: pp.RequireIdentityVerification,
		LznDenom:                    pp.LznDenom,
	}, nil
}
