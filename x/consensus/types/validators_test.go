package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/crypto"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
)

func TestInitialValidatorsToAbci_Empty(t *testing.T) {
	out := InitialValidatorsToAbci(nil)
	require.Nil(t, out)
	out = InitialValidatorsToAbci([]*consensusv1.InitialValidator{})
	require.Nil(t, out)
}

func TestInitialValidatorsToAbci_Valid(t *testing.T) {
	pk := &cmtproto.PublicKey{Sum: &cmtproto.PublicKey_Ed25519{Ed25519: make([]byte, 32)}}
	pkBz, _ := pk.Marshal()
	initial := []*consensusv1.InitialValidator{
		{PubKey: pkBz, Power: 10},
		{PubKey: pkBz, Power: 20},
	}
	out := InitialValidatorsToAbci(initial)
	require.Len(t, out, 2)
	require.Equal(t, int64(10), out[0].Power)
	require.Equal(t, int64(20), out[1].Power)
}

func TestAbciValidatorsToInitial_Empty(t *testing.T) {
	out := AbciValidatorsToInitial(nil)
	require.Nil(t, out)
	out = AbciValidatorsToInitial([]abci.ValidatorUpdate{})
	require.Nil(t, out)
}

func TestAbciValidatorsToInitial_Valid(t *testing.T) {
	pk := cmtproto.PublicKey{Sum: &cmtproto.PublicKey_Ed25519{Ed25519: make([]byte, 32)}}
	updates := []abci.ValidatorUpdate{
		{PubKey: pk, Power: 5},
		{PubKey: pk, Power: 15},
	}
	out := AbciValidatorsToInitial(updates)
	require.Len(t, out, 2)
	require.Equal(t, int64(5), out[0].Power)
	require.Equal(t, int64(15), out[1].Power)
	require.NotEmpty(t, out[0].PubKey)
	require.NotEmpty(t, out[1].PubKey)
}

func TestInitialValidatorsToAbci_RoundTrip(t *testing.T) {
	pk := &cmtproto.PublicKey{Sum: &cmtproto.PublicKey_Ed25519{Ed25519: make([]byte, 32)}}
	pkBz, _ := pk.Marshal()
	initial := []*consensusv1.InitialValidator{{PubKey: pkBz, Power: 100}}
	abciList := InitialValidatorsToAbci(initial)
	require.Len(t, abciList, 1)
	back := AbciValidatorsToInitial(abciList)
	require.Len(t, back, 1)
	require.Equal(t, int64(100), back[0].Power)
	require.Equal(t, pkBz, back[0].PubKey)
}
