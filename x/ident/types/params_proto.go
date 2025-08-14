package types

import (
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ToProto converts local Params to protobuf Params
func (p Params) ToProto() *identv1.Params {
	return &identv1.Params{
		CitizenInactivityPeriod:   durationpb.New(p.CitizenInactivityPeriod),
		ValidatorInactivityPeriod: durationpb.New(p.ValidatorInactivityPeriod),
	}
}

// ParamsFromProto converts protobuf Params to local Params
func ParamsFromProto(pp *identv1.Params) Params {
	if pp == nil {
		return DefaultParams()
	}
	return Params{
		CitizenInactivityPeriod:   pp.CitizenInactivityPeriod.AsDuration(),
		ValidatorInactivityPeriod: pp.ValidatorInactivityPeriod.AsDuration(),
	}
}
