package types

import (
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

func (p Params) ToProto() *lizenzv1.Params {
	return &lizenzv1.Params{
		MaxActivatedPerValidator: p.MaxActivatedLZNPerValidator,
		ActivityCoefficient:      p.ActivityCoefficient,
		DeactivationPeriod:       durationpb.New(p.DeactivationPeriod),
		InactivityPeriod:         durationpb.New(p.InactivityPeriod),
	}
}

func ParamsFromProto(pp *lizenzv1.Params) (Params, error) {
	if pp == nil {
		return DefaultParams(), nil
	}
	return Params{
		MaxActivatedLZNPerValidator: pp.MaxActivatedPerValidator,
		ActivityCoefficient:         pp.ActivityCoefficient,
		DeactivationPeriod:          pp.DeactivationPeriod.AsDuration(),
		InactivityPeriod:            pp.InactivityPeriod.AsDuration(),
	}, nil
}
