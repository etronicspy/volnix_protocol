package types

import (
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ToProto converts local Params to protobuf Params
func (p Params) ToProto() *identv1.Params {
	return &identv1.Params{
		CitizenActivityPeriod:        durationpb.New(p.CitizenActivityPeriod),
		ValidatorActivityPeriod:      durationpb.New(p.ValidatorActivityPeriod),
		MaxIdentitiesPerAddress:      p.MaxIdentitiesPerAddress,
		RequireIdentityVerification:  p.RequireIdentityVerification,
		DefaultVerificationProvider:  p.DefaultVerificationProvider,
		VerificationCost:             &p.VerificationCost,
		MigrationFee:                 &p.MigrationFee,
		RoleChangeFee:                &p.RoleChangeFee,
	}
}

// ParamsFromProto converts protobuf Params to local Params
func ParamsFromProto(pp *identv1.Params) Params {
	if pp == nil {
		return DefaultParams()
	}
	return Params{
		CitizenActivityPeriod:        pp.CitizenActivityPeriod.AsDuration(),
		ValidatorActivityPeriod:      pp.ValidatorActivityPeriod.AsDuration(),
		MaxIdentitiesPerAddress:      pp.MaxIdentitiesPerAddress,
		RequireIdentityVerification:  pp.RequireIdentityVerification,
		DefaultVerificationProvider:  pp.DefaultVerificationProvider,
		VerificationCost:             *pp.VerificationCost,
		MigrationFee:                 *pp.MigrationFee,
		RoleChangeFee:                *pp.RoleChangeFee,
	}
}
