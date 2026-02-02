package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
)

func TestRegisterInterfaces(t *testing.T) {
	reg := cdctypes.NewInterfaceRegistry()
	require.NotPanics(t, func() { types.RegisterInterfaces(reg) })
}

func TestRegisterInterfaces_Implementations(t *testing.T) {
	reg := cdctypes.NewInterfaceRegistry()
	types.RegisterInterfaces(reg)

	any, err := cdctypes.NewAnyWithValue(&lizenzv1.MsgActivateLZN{})
	require.NoError(t, err)
	require.NotNil(t, any)
}
