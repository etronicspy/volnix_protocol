package types

import (
	"fmt"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
)

// Type aliases for convenience
type (
	Validator          = consensusv1.Validator
	ValidatorStatus    = consensusv1.ValidatorStatus
	PerHeightBurn      = consensusv1.PerHeightBurn
	HeightBurnSummary  = consensusv1.HeightBurnSummary
	HalvingInfo        = consensusv1.HalvingInfo
	ConsensusState     = consensusv1.ConsensusState
	ValidatorWeight    = consensusv1.ValidatorWeight
	Params             = consensusv1.Params
	GenesisState       = consensusv1.GenesisState
	InitialValidator   = consensusv1.InitialValidator
)

// DefaultGenesis returns default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		Validators: []*Validator{},
	}
}

// ValidateGenesis performs basic validation on genesis state
func ValidateGenesis(gs *GenesisState) error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return ValidateParams(gs.Params)
}

// DefaultParams returns default consensus parameters
func DefaultParams() *Params {
	return &Params{
		BaseBlockTime:              "5s",
		BaseBlockReward:            "50000000uwrt",
		BurnCapLambda:              "0.333333333333333333",
		FeePolicyBZero:             "community_pool", // deprecated: B=0 impossible with min burn threshold (v4.17)
		HalvingInterval:            210000,
		MoaPenaltyThresholdHigh:    "1.0",
		MoaPenaltyThresholdWarning: "0.9",
		MoaPenaltyThresholdMedium:  "0.7",
		MoaPenaltyThresholdLow:     "0.5",
		MaxBlockGas:                40000000,
		MaxBlockBytes:              22020096,
	}
}

// ValidateParams performs basic validation on consensus parameters
func ValidateParams(p *Params) error {
	if p == nil {
		return fmt.Errorf("params cannot be nil")
	}

	if p.BaseBlockTime == "" {
		return fmt.Errorf("base block time cannot be empty")
	}

	if p.BaseBlockReward == "" {
		return fmt.Errorf("base block reward cannot be empty")
	}

	if p.BurnCapLambda == "" {
		return fmt.Errorf("burn cap lambda cannot be empty")
	}

	if p.HalvingInterval == 0 {
		return fmt.Errorf("halving interval must be positive")
	}

	return nil
}

// Param keys
var (
	KeyBaseBlockTime              = []byte("BaseBlockTime")
	KeyBaseBlockReward            = []byte("BaseBlockReward")
	KeyBurnCapLambda              = []byte("BurnCapLambda")
	KeyFeePolicyBZero             = []byte("FeePolicyBZero")
	KeyHalvingInterval            = []byte("HalvingInterval")
	KeyMoaPenaltyThresholdHigh    = []byte("MoaPenaltyThresholdHigh")
	KeyMoaPenaltyThresholdWarning = []byte("MoaPenaltyThresholdWarning")
	KeyMoaPenaltyThresholdMedium  = []byte("MoaPenaltyThresholdMedium")
	KeyMoaPenaltyThresholdLow     = []byte("MoaPenaltyThresholdLow")
	KeyMaxBlockGas                = []byte("MaxBlockGas")
	KeyMaxBlockBytes              = []byte("MaxBlockBytes")
)
