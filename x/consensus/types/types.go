package types

import (
	"fmt"
	
	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
)

// Type aliases for convenience
type (
	Validator     = consensusv1.Validator
	ValidatorStatus = consensusv1.ValidatorStatus
	Params       = consensusv1.Params
	GenesisState = consensusv1.GenesisState
)

// DefaultGenesis returns default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: &Params{
			BaseBlockTime:         "5s",
			HighActivityThreshold: 1000,
			LowActivityThreshold:  100,
		},
		Validators: []*Validator{},
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
		BaseBlockTime:         "5s",
		HighActivityThreshold: 1000,
		LowActivityThreshold:  100,
	}
}

// ValidateParams performs basic validation on consensus parameters
func ValidateParams(p *Params) error {
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

// Param keys
var (
	KeyBaseBlockTime         = []byte("BaseBlockTime")
	KeyHighActivityThreshold = []byte("HighActivityThreshold")
	KeyLowActivityThreshold  = []byte("LowActivityThreshold")
)
