package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

func TestErrors(t *testing.T) {
	// Test that all errors are defined
	require.NotNil(t, types.ErrLizenzNotFound)
	require.NotNil(t, types.ErrLizenzAlreadyExists)
	require.NotNil(t, types.ErrEmptyValidator)
	require.NotNil(t, types.ErrEmptyAmount)
	require.NotNil(t, types.ErrInvalidAmount)
}

func TestErrorMessages(t *testing.T) {
	// Test error messages are not empty
	require.NotEmpty(t, types.ErrLizenzNotFound.Error())
	require.NotEmpty(t, types.ErrLizenzAlreadyExists.Error())
	require.NotEmpty(t, types.ErrEmptyValidator.Error())
}
