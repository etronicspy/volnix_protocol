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

func validateNoop(i interface{}) error { return nil }

// ParamSetPairs get the params.ParamSet
func (p *ConsensusParams) ParamSetPairs() paramtypes.ParamSetPairs {
	if p.Params == nil {
		p.Params = DefaultParams()
	}
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyBaseBlockTime, &p.BaseBlockTime, validateNoop),
		paramtypes.NewParamSetPair(KeyBaseBlockReward, &p.BaseBlockReward, validateNoop),
		paramtypes.NewParamSetPair(KeyBurnCapLambda, &p.BurnCapLambda, validateNoop),
		paramtypes.NewParamSetPair(KeyFeePolicyBZero, &p.FeePolicyBZero, validateNoop),
		paramtypes.NewParamSetPair(KeyHalvingInterval, &p.HalvingInterval, validateNoop),
		paramtypes.NewParamSetPair(KeyMoaPenaltyThresholdHigh, &p.MoaPenaltyThresholdHigh, validateNoop),
		paramtypes.NewParamSetPair(KeyMoaPenaltyThresholdWarning, &p.MoaPenaltyThresholdWarning, validateNoop),
		paramtypes.NewParamSetPair(KeyMoaPenaltyThresholdMedium, &p.MoaPenaltyThresholdMedium, validateNoop),
		paramtypes.NewParamSetPair(KeyMoaPenaltyThresholdLow, &p.MoaPenaltyThresholdLow, validateNoop),
		paramtypes.NewParamSetPair(KeyMaxBlockGas, &p.MaxBlockGas, validateNoop),
		paramtypes.NewParamSetPair(KeyMaxBlockBytes, &p.MaxBlockBytes, validateNoop),
	}
}

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(NewConsensusParams(nil))
}
