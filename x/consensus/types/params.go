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
	if p.Params == nil {
		p.Params = DefaultParams()
	}
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyBaseBlockTime, &p.BaseBlockTime, validateBaseBlockTime),
		paramtypes.NewParamSetPair(KeyHighActivityThreshold, &p.HighActivityThreshold, validateHighActivityThreshold),
		paramtypes.NewParamSetPair(KeyLowActivityThreshold, &p.LowActivityThreshold, validateLowActivityThreshold),
		paramtypes.NewParamSetPair(KeyMinBurnAmount, &p.MinBurnAmount, validateMinBurnAmount),
		paramtypes.NewParamSetPair(KeyMaxBurnAmount, &p.MaxBurnAmount, validateMaxBurnAmount),
		paramtypes.NewParamSetPair(KeyBlockCreatorSelectionRounds, &p.BlockCreatorSelectionRounds, validateBlockCreatorSelectionRounds),
		paramtypes.NewParamSetPair(KeyActivityDecayRate, &p.ActivityDecayRate, validateActivityDecayRate),
		paramtypes.NewParamSetPair(KeyMoaPenaltyRate, &p.MoaPenaltyRate, validateMoaPenaltyRate),
	}
}

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(NewConsensusParams(nil))
}
