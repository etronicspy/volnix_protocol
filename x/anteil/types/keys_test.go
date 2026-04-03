package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

func TestGetOrderKey(t *testing.T) {
	key := types.GetOrderKey("order123")
	require.NotNil(t, key)
	require.Greater(t, len(key), 0)

	key2 := types.GetOrderKey("order456")
	require.NotEqual(t, key, key2)
}

func TestGetTradeKey(t *testing.T) {
	key := types.GetTradeKey("trade123")
	require.NotNil(t, key)
	require.Greater(t, len(key), 0)

	key2 := types.GetTradeKey("trade456")
	require.NotEqual(t, key, key2)
}

func TestGetUserPositionKey(t *testing.T) {
	key := types.GetUserPositionKey("cosmos1test")
	require.NotNil(t, key)
	require.Greater(t, len(key), 0)

	key2 := types.GetUserPositionKey("cosmos1test2")
	require.NotEqual(t, key, key2)
}

func TestGetOrderPrefix(t *testing.T) {
	prefix := types.GetOrderPrefix()
	require.NotNil(t, prefix)
	require.Greater(t, len(prefix), 0)
}

func TestGetTradePrefix(t *testing.T) {
	prefix := types.GetTradePrefix()
	require.NotNil(t, prefix)
	require.Greater(t, len(prefix), 0)
}

func TestPrefixesAreUnique(t *testing.T) {
	orderPrefix := types.GetOrderPrefix()
	tradePrefix := types.GetTradePrefix()

	require.NotEqual(t, orderPrefix, tradePrefix)
}
