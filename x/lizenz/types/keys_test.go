package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

func TestGetActivatedLizenzKey(t *testing.T) {
	k := types.GetActivatedLizenzKey("validator1")
	require.NotEmpty(t, k)
	require.True(t, len(k) > len(types.ActivatedLizenzKeyPrefix))
}

func TestGetRewardHistoryKey(t *testing.T) {
	k := types.GetRewardHistoryKey("validator1")
	require.NotEmpty(t, k)
	require.True(t, len(k) > len(types.RewardHistoryKeyPrefix))
}

func TestGetDeactivatingLizenzKey(t *testing.T) {
	k := types.GetDeactivatingLizenzKey("validator1")
	require.NotEmpty(t, k)
	require.True(t, len(k) > len(types.DeactivatingLizenzKeyPrefix))
}

func TestGetMOAStatusKey(t *testing.T) {
	k := types.GetMOAStatusKey("validator1")
	require.NotEmpty(t, k)
	require.True(t, len(k) > len(types.MOAStatusKeyPrefix))
}
