package types

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyCitizenInactivityPeriod   = []byte("CitizenInactivityPeriod")
	KeyValidatorInactivityPeriod = []byte("ValidatorInactivityPeriod")
)

// Ensure Params implements ParamSet
var _ paramtypes.ParamSet = (*Params)(nil)

// Params defines ident module parameters
type Params struct {
	CitizenInactivityPeriod   time.Duration `json:"citizen_inactivity_period"`
	ValidatorInactivityPeriod time.Duration `json:"validator_inactivity_period"`
}

// ParamKeyTable for ident module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs returns the parameter set pairs
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyCitizenInactivityPeriod, &p.CitizenInactivityPeriod, validateDuration),
		paramtypes.NewParamSetPair(KeyValidatorInactivityPeriod, &p.ValidatorInactivityPeriod, validateDuration),
	}
}

// DefaultParams returns default parameters
func DefaultParams() Params {
	return Params{
		CitizenInactivityPeriod:   365 * 24 * time.Hour,
		ValidatorInactivityPeriod: 180 * 24 * time.Hour,
	}
}

// Validate performs basic validation of params
func (p Params) Validate() error {
	if err := validateDuration(p.CitizenInactivityPeriod); err != nil {
		return fmt.Errorf("invalid CitizenInactivityPeriod: %w", err)
	}
	if err := validateDuration(p.ValidatorInactivityPeriod); err != nil {
		return fmt.Errorf("invalid ValidatorInactivityPeriod: %w", err)
	}
	return nil
}

func validateDuration(i interface{}) error {
	d, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("expected time.Duration, got %T", i)
	}
	if d <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	return nil
}
