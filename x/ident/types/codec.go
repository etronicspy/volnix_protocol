package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	identv1 "github.com/helvetia-protocol/helvetia-protocol/proto/gen/go/helvetia/ident/v1"
)

// RegisterInterfaces registers module concrete types on the given InterfaceRegistry.
// Registers request and response messages for routing and amino/json compatibility.
func RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	// sdk.Msg requests
	reg.RegisterImplementations((*sdk.Msg)(nil),
		&identv1.MsgVerifyIdentity{},
		&identv1.MsgMigrateRole{},
		&identv1.MsgChangeRole{},
	)

	// tx.MsgResponse responses
	reg.RegisterImplementations((*txtypes.MsgResponse)(nil),
		&identv1.MsgVerifyIdentityResponse{},
		&identv1.MsgMigrateRoleResponse{},
		&identv1.MsgChangeRoleResponse{},
	)
}
