package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

func TestNewConsensusParams_Nil(t *testing.T) {
	cp := types.NewConsensusParams(nil)
	require.NotNil(t, cp)
	require.NotNil(t, cp.Params)
	require.Equal(t, types.DefaultParams().BaseBlockTime, cp.BaseBlockTime)
}

func TestNewConsensusParams_Valid(t *testing.T) {
	p := types.DefaultParams()
	cp := types.NewConsensusParams(p)
	require.NotNil(t, cp)
	require.Equal(t, p, cp.Params)
}

func TestConsensusParams_ParamSetPairs(t *testing.T) {
	cp := types.NewConsensusParams(nil)
	pairs := cp.ParamSetPairs()
	require.NotEmpty(t, pairs)
}

func TestParamKeyTable(t *testing.T) {
	kt := types.ParamKeyTable()
	require.NotNil(t, kt)
}
