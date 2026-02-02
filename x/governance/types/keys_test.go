package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/volnix-protocol/volnix-protocol/x/governance/types"
)

func TestGetProposalKey(t *testing.T) {
	k := types.GetProposalKey(0)
	require.NotEmpty(t, k)
	require.True(t, len(k) > len(types.ProposalKeyPrefix))

	k1 := types.GetProposalKey(1)
	k2 := types.GetProposalKey(2)
	require.NotEqual(t, k1, k2)
}

func TestGetVoteKey(t *testing.T) {
	k := types.GetVoteKey(1, "cosmos1voter")
	require.NotEmpty(t, k)
	require.True(t, len(k) > len(types.VoteKeyPrefix))

	k2 := types.GetVoteKey(1, "cosmos1other")
	require.NotEqual(t, k, k2)
}
