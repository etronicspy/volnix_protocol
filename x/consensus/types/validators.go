package types

import (
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/crypto"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
)

// InitialValidatorsToAbci converts genesis InitialValidators to abci.ValidatorUpdate slice.
// Used by consensus module InitGenesis (HasABCIGenesis) for CometBFT handshake.
func InitialValidatorsToAbci(initial []*consensusv1.InitialValidator) []abci.ValidatorUpdate {
	if len(initial) == 0 {
		return nil
	}
	out := make([]abci.ValidatorUpdate, len(initial))
	for i, v := range initial {
		if v == nil {
			continue
		}
		var pubKey cmtproto.PublicKey
		if len(v.PubKey) > 0 {
			if err := pubKey.Unmarshal(v.PubKey); err != nil {
				continue
			}
		}
		out[i] = abci.ValidatorUpdate{
			PubKey: pubKey,
			Power:  v.Power,
		}
	}
	return out
}

// AbciValidatorsToInitial converts req.Validators (abci) to genesis InitialValidators.
// Used by app InitChainer when building default consensus genesis.
func AbciValidatorsToInitial(validators []abci.ValidatorUpdate) []*consensusv1.InitialValidator {
	if len(validators) == 0 {
		return nil
	}
	out := make([]*consensusv1.InitialValidator, len(validators))
	for i, v := range validators {
		var pubKeyBz []byte
		if n := v.PubKey.Size(); n > 0 {
			pubKeyBz = make([]byte, n)
			v.PubKey.MarshalToSizedBuffer(pubKeyBz)
		}
		out[i] = &consensusv1.InitialValidator{
			PubKey: pubKeyBz,
			Power:  v.Power,
		}
	}
	return out
}
