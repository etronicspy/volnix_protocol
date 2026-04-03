package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

func TestErrors(t *testing.T) {
	require.NotNil(t, types.ErrOrderNotFound)
	require.NotNil(t, types.ErrOrderAlreadyExists)
	require.NotNil(t, types.ErrTradeNotFound)
	require.NotNil(t, types.ErrTradeAlreadyExists)
	require.NotNil(t, types.ErrPositionNotFound)
	require.NotNil(t, types.ErrEmptyOwner)
	require.NotNil(t, types.ErrEmptyAntAmount)
	require.NotNil(t, types.ErrEmptyPrice)
}

func TestErrorMessages(t *testing.T) {
	require.NotEmpty(t, types.ErrOrderNotFound.Error())
	require.NotEmpty(t, types.ErrPositionNotFound.Error())
}
