package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

func TestErrors(t *testing.T) {
	// Test that all errors are defined
	require.NotNil(t, types.ErrAccountAlreadyExists)
	require.NotNil(t, types.ErrAccountNotFound)
	require.NotNil(t, types.ErrEmptyAddress)
	require.NotNil(t, types.ErrEmptyIdentityHash)
	require.NotNil(t, types.ErrInvalidRole)
	require.NotNil(t, types.ErrInvalidRoleChoice)
	require.NotNil(t, types.ErrDuplicateIdentityHash)
	require.NotNil(t, types.ErrRoleMigrationNotFound)
}

func TestErrorMessages(t *testing.T) {
	// Test error messages are not empty
	require.NotEmpty(t, types.ErrAccountAlreadyExists.Error())
	require.NotEmpty(t, types.ErrEmptyAddress.Error())
	require.NotEmpty(t, types.ErrInvalidRole.Error())
	require.NotEmpty(t, types.ErrDuplicateIdentityHash.Error())
}
