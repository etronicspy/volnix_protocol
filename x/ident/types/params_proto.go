package types

import (
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ToProto converts local Params to protobuf Params
func (p Params) ToProto() *identv1.Params {
	return &identv1.Params{
		MoaSupplierWindow:            durationpb.New(p.MoaSupplierWindow),
		MoaValidatorWindow:           durationpb.New(p.MoaValidatorWindow),
		MaxActiveSuppliers:           p.MaxActiveSuppliers,
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
		MoaSupplierWindow:            pp.MoaSupplierWindow.AsDuration(),
		MoaValidatorWindow:           pp.MoaValidatorWindow.AsDuration(),
		MaxActiveSuppliers:           pp.MaxActiveSuppliers,
		RequireIdentityVerification:  pp.RequireIdentityVerification,
		DefaultVerificationProvider:  pp.DefaultVerificationProvider,
		VerificationCost:             *pp.VerificationCost,
		MigrationFee:                 *pp.MigrationFee,
		RoleChangeFee:                *pp.RoleChangeFee,
	}
}
