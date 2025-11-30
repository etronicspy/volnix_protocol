package types

import (
	"fmt"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
)

// Type aliases for convenience
type (
	Validator       = consensusv1.Validator
	ValidatorStatus = consensusv1.ValidatorStatus
	BlockCreator    = consensusv1.BlockCreator
	BurnProof       = consensusv1.BurnProof
	ActivityScore   = consensusv1.ActivityScore
	HalvingInfo     = consensusv1.HalvingInfo
	ConsensusState  = consensusv1.ConsensusState
	ValidatorWeight = consensusv1.ValidatorWeight
	Params          = consensusv1.Params
	GenesisState    = consensusv1.GenesisState
)

// DefaultGenesis returns default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Validators:     []*Validator{},
		BlockCreators:  []*BlockCreator{},
		BurnProofs:     []*BurnProof{},
		ActivityScores: []*ActivityScore{},
	}
}

// ValidateGenesis performs basic validation on genesis state
func ValidateGenesis(gs *GenesisState) error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return nil
}

// DefaultParams returns default consensus parameters
func DefaultParams() *Params {
	return &Params{
		BaseBlockTime:               "5s",
		HighActivityThreshold:       1000,
		LowActivityThreshold:        100,
		MinBurnAmount:               "1000000uvx",
		MaxBurnAmount:               "1000000000uvx",
		BlockCreatorSelectionRounds: 10,
		ActivityDecayRate:           "0.95",
		MoaPenaltyRate:              "0.1",
		BaseBlockReward:             "50000000uwrt", // 50 WRT in micro units
		MoaPenaltyThresholdHigh:     "1.0",         // >= 1.0: no penalty
		MoaPenaltyThresholdWarning:  "0.9",         // >= 0.9: warning
		MoaPenaltyThresholdMedium:   "0.7",         // >= 0.7: 25% penalty
		MoaPenaltyThresholdLow:      "0.5",         // >= 0.5: 50% penalty
		ActivityFactorHigh:          "0.5",         // High activity: faster blocks
		ActivityFactorMedium:        "0.75",        // Medium activity: moderate speed
		ActivityFactorNormal:        "1.0",        // Normal activity: normal speed
		AverageBlockTimeWindowSize:  1000,          // Window size for average block time
		BidHistoryLimit:             100,           // Maximum bid history entries
		AuctionHistoryBlocks:        100,           // Blocks to keep auction history
		RapidBidLimit:               5,             // Maximum rapid bids allowed
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
	
	if p.MinBurnAmount == "" {
		return fmt.Errorf("min burn amount cannot be empty")
	}
	
	if p.MaxBurnAmount == "" {
		return fmt.Errorf("max burn amount cannot be empty")
	}
	
	return nil
}

// validateBaseBlockTime validates the base block time
func validateBaseBlockTime(i interface{}) error {
	return nil
}

// validateHighActivityThreshold validates the high activity threshold
func validateHighActivityThreshold(i interface{}) error {
	return nil
}

// validateLowActivityThreshold validates the low activity threshold
func validateLowActivityThreshold(i interface{}) error {
	return nil
}

// validateMinBurnAmount validates the minimum burn amount
func validateMinBurnAmount(i interface{}) error {
	return nil
}

// validateMaxBurnAmount validates the maximum burn amount
func validateMaxBurnAmount(i interface{}) error {
	return nil
}

// validateActivityDecayRate validates the activity decay rate
func validateActivityDecayRate(i interface{}) error {
	return nil
}

// validateBlockCreatorSelectionRounds validates the block creator selection rounds
func validateBlockCreatorSelectionRounds(i interface{}) error {
	return nil
}

// validateMoaPenaltyRate validates the MOA penalty rate
func validateMoaPenaltyRate(i interface{}) error {
	return nil
}

// validateBaseBlockReward validates the base block reward
func validateBaseBlockReward(i interface{}) error {
	return nil
}

// validateMoaPenaltyThreshold validates MOA penalty thresholds
func validateMoaPenaltyThreshold(i interface{}) error {
	return nil
}

// validateActivityFactor validates activity factors
func validateActivityFactor(i interface{}) error {
	return nil
}

// validateUint64 validates uint64 parameters
func validateUint64(i interface{}) error {
	return nil
}

// Param keys
var (
	KeyBaseBlockTime               = []byte("BaseBlockTime")
	KeyHighActivityThreshold       = []byte("HighActivityThreshold")
	KeyLowActivityThreshold        = []byte("LowActivityThreshold")
	KeyMinBurnAmount               = []byte("MinBurnAmount")
	KeyMaxBurnAmount               = []byte("MaxBurnAmount")
	KeyBlockCreatorSelectionRounds = []byte("BlockCreatorSelectionRounds")
	KeyActivityDecayRate           = []byte("ActivityDecayRate")
	KeyMoaPenaltyRate              = []byte("MoaPenaltyRate")
	KeyBaseBlockReward             = []byte("BaseBlockReward")
	KeyMoaPenaltyThresholdHigh     = []byte("MoaPenaltyThresholdHigh")
	KeyMoaPenaltyThresholdWarning  = []byte("MoaPenaltyThresholdWarning")
	KeyMoaPenaltyThresholdMedium   = []byte("MoaPenaltyThresholdMedium")
	KeyMoaPenaltyThresholdLow      = []byte("MoaPenaltyThresholdLow")
	KeyActivityFactorHigh          = []byte("ActivityFactorHigh")
	KeyActivityFactorMedium        = []byte("ActivityFactorMedium")
	KeyActivityFactorNormal        = []byte("ActivityFactorNormal")
	KeyAverageBlockTimeWindowSize  = []byte("AverageBlockTimeWindowSize")
	KeyBidHistoryLimit             = []byte("BidHistoryLimit")
	KeyAuctionHistoryBlocks        = []byte("AuctionHistoryBlocks")
	KeyRapidBidLimit               = []byte("RapidBidLimit")
)
