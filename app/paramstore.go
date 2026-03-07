package app

import (
	"context"
	"encoding/json"
	"fmt"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"

	sdkerrors "cosmossdk.io/errors"
)

// Consensus params key in the param store DB (single key per DB).
var paramStoreKey = []byte("consensus_params")

var errParamsNotFound = sdkerrors.Register("paramstore", 1, "consensus params not found")

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
		return fmt.Errorf("paramstore: failed to marshal consensus params: %w", err)
	}
	if err := ps.db.Set(paramStoreKey, bz); err != nil {
		return fmt.Errorf("paramstore: failed to persist consensus params: %w", err)
	}
	return nil
}

func (ps *paramStore) Has(_ context.Context) (bool, error) {
	has, err := ps.db.Has(paramStoreKey)
	if err != nil {
		return false, fmt.Errorf("paramstore: failed to check consensus params existence: %w", err)
	}
	return has, nil
}

func (ps *paramStore) Get(_ context.Context) (cmtproto.ConsensusParams, error) {
	bz, err := ps.db.Get(paramStoreKey)
	if err != nil {
		return cmtproto.ConsensusParams{}, fmt.Errorf("paramstore: failed to read consensus params: %w", err)
	}
	if len(bz) == 0 {
		return cmtproto.ConsensusParams{}, errParamsNotFound
	}
	var params cmtproto.ConsensusParams
	if err := json.Unmarshal(bz, &params); err != nil {
		return cmtproto.ConsensusParams{}, fmt.Errorf("paramstore: failed to unmarshal consensus params: %w", err)
	}
	return params, nil
}
