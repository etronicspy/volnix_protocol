package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cosmosdb "github.com/cosmos/cosmos-db"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

func TestParamStore_SetGetHas(t *testing.T) {
	db := cosmosdb.NewMemDB()

	ps := NewParamStore(db)
	ctx := context.Background()

	has, err := ps.Has(ctx)
	require.NoError(t, err)
	require.False(t, has)

	params := cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{MaxBytes: 100, MaxGas: 200},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 10,
			MaxAgeDuration:  time.Second,
		},
		Validator: &cmtproto.ValidatorParams{PubKeyTypes: []string{"ed25519"}},
	}
	err = ps.Set(ctx, params)
	require.NoError(t, err)

	has, err = ps.Has(ctx)
	require.NoError(t, err)
	require.True(t, has)

	got, err := ps.Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, got.Block)
	require.Equal(t, int64(100), got.Block.MaxBytes)
	require.Equal(t, int64(200), got.Block.MaxGas)
	require.NotNil(t, got.Validator)
	require.Equal(t, []string{"ed25519"}, got.Validator.PubKeyTypes)
}

func TestParamStore_GetNotFound(t *testing.T) {
	db := cosmosdb.NewMemDB()

	ps := NewParamStore(db)
	ctx := context.Background()

	_, err := ps.Get(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
