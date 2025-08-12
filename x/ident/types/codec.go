package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterInterfaces registers module concrete types on the given InterfaceRegistry.
// Add interfaces and implementations here when Msg/Query types are implemented.
func RegisterInterfaces(reg cdctypes.InterfaceRegistry) {}
