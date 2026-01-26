package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

func TestRegisterInterfaces(t *testing.T) {
	registry := cdctypes.NewInterfaceRegistry()
	
	// Should not panic
	require.NotPanics(t, func() {
		types.RegisterInterfaces(registry)
	})
	
	// Registry should not be nil after registration
	require.NotNil(t, registry)
}
