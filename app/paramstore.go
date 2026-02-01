package app

import (
	"context"
	"encoding/json"
	"errors"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
)

// Consensus params key in the param store DB (single key per DB).
var paramStoreKey = []byte("consensus_params")

// paramStore implements baseapp.ParamStore using a cosmosdb.DB (e.g. LevelDB or MemDB).
type paramStore struct {
	db cosmosdb.DB
}

// Ensure paramStore implements baseapp.ParamStore.
var _ baseapp.ParamStore = (*paramStore)(nil)

// NewParamStore returns a ParamStore backed by the given DB.
func NewParamStore(db cosmosdb.DB) baseapp.ParamStore {
	return &paramStore{db: db}
}

func (ps *paramStore) Set(_ context.Context, cp cmtproto.ConsensusParams) error {
	bz, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	return ps.db.Set(paramStoreKey, bz)
}

func (ps *paramStore) Has(_ context.Context) (bool, error) {
	return ps.db.Has(paramStoreKey)
}

func (ps *paramStore) Get(_ context.Context) (cmtproto.ConsensusParams, error) {
	bz, err := ps.db.Get(paramStoreKey)
	if err != nil {
		return cmtproto.ConsensusParams{}, err
	}
	if len(bz) == 0 {
		return cmtproto.ConsensusParams{}, errors.New("params not found")
	}
	var params cmtproto.ConsensusParams
	if err := json.Unmarshal(bz, &params); err != nil {
		return cmtproto.ConsensusParams{}, err
	}
	return params, nil
}
