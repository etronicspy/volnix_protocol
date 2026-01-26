package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
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

func TestRegisterLegacyAminoCodec(t *testing.T) {
	cdc := codec.NewLegacyAmino()
	
	// Should not panic
	require.NotPanics(t, func() {
		types.RegisterLegacyAminoCodec(cdc)
	})
	
	// Codec should not be nil after registration
	require.NotNil(t, cdc)
}
