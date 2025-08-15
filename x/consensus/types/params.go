package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// ConsensusParams wraps the protobuf Params to implement ParamSet interface
type ConsensusParams struct {
	*Params
}

// NewConsensusParams creates a new ConsensusParams instance
func NewConsensusParams(params *Params) *ConsensusParams {
	if params == nil {
		params = DefaultParams()
	}
	return &ConsensusParams{Params: params}
}

// ParamSetPairs get the params.ParamSet
func (p *ConsensusParams) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyBaseBlockTime, &p.BaseBlockTime, validateBaseBlockTime),
		paramtypes.NewParamSetPair(KeyHighActivityThreshold, &p.HighActivityThreshold, validateHighActivityThreshold),
		paramtypes.NewParamSetPair(KeyLowActivityThreshold, &p.LowActivityThreshold, validateLowActivityThreshold),
	}
}

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(NewConsensusParams(nil))
}
