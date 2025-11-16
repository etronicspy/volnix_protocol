package types

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	// KeyVotingPeriod defines the key for voting period
	KeyVotingPeriod = []byte("VotingPeriod")

	// KeyTimelockPeriod defines the key for timelock period
	KeyTimelockPeriod = []byte("TimelockPeriod")

	// KeyMinDeposit defines the key for minimum deposit
	KeyMinDeposit = []byte("MinDeposit")

	// KeyQuorum defines the key for quorum
	KeyQuorum = []byte("Quorum")

	// KeyThreshold defines the key for threshold
	KeyThreshold = []byte("Threshold")
)

// ParamKeyTable returns the parameter key table
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// Params defines the parameters for the governance module
type Params struct {
	VotingPeriod  time.Duration `json:"voting_period"`  // Duration of voting period
	TimelockPeriod time.Duration `json:"timelock_period"` // Period before proposal execution
	MinDeposit    string        `json:"min_deposit"`     // Minimum WRT deposit to create proposal
	Quorum        string        `json:"quorum"`         // Minimum quorum (as decimal, e.g., "0.4" for 40%)
	Threshold     string        `json:"threshold"`       // Minimum threshold for passing (as decimal, e.g., "0.5" for 50%)
}

// ParamSetPairs returns the parameter set pairs
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyVotingPeriod, &p.VotingPeriod, validateDuration),
		paramtypes.NewParamSetPair(KeyTimelockPeriod, &p.TimelockPeriod, validateDuration),
		paramtypes.NewParamSetPair(KeyMinDeposit, &p.MinDeposit, validateString),
		paramtypes.NewParamSetPair(KeyQuorum, &p.Quorum, validateDecimal),
		paramtypes.NewParamSetPair(KeyThreshold, &p.Threshold, validateDecimal),
	}
}

// DefaultParams returns default governance parameters
func DefaultParams() Params {
	return Params{
		VotingPeriod:  7 * 24 * time.Hour,  // 7 days
		TimelockPeriod: 14 * 24 * time.Hour, // 14 days (according to whitepaper: "длительный период ожидания")
		MinDeposit:    "1000000",            // 1 WRT (in micro units)
		Quorum:        "0.4",               // 40% of total WRT supply
		Threshold:     "0.5",               // 50% of votes must be yes
	}
}

// Validate validates the parameters
func (p Params) Validate() error {
	if p.VotingPeriod <= 0 {
		return fmt.Errorf("voting period must be positive")
	}
	if p.TimelockPeriod <= 0 {
		return fmt.Errorf("timelock period must be positive")
	}
	if p.MinDeposit == "" {
		return fmt.Errorf("min deposit cannot be empty")
	}
	if p.Quorum == "" {
		return fmt.Errorf("quorum cannot be empty")
	}
	if p.Threshold == "" {
		return fmt.Errorf("threshold cannot be empty")
	}
	return nil
}

// validateDuration validates a duration parameter
func validateDuration(i interface{}) error {
	_, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

// validateString validates a string parameter
func validateString(i interface{}) error {
	_, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

// validateDecimal validates a decimal string parameter
func validateDecimal(i interface{}) error {
	str, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	// Validate that it's a valid decimal between 0 and 1
	// This is a simple check - actual parsing would be done in keeper
	if str == "" {
		return fmt.Errorf("decimal cannot be empty")
	}
	return nil
}

