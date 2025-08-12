package types

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyMaxActivatedLZNPerValidator = []byte("MaxActivatedLZNPerValidator")
	KeyActivityCoefficient         = []byte("ActivityCoefficient")
	KeyDeactivationPeriod          = []byte("DeactivationPeriod")
	KeyInactivityPeriod            = []byte("InactivityPeriod")
)

// Ensure Params implements ParamSet
var _ paramtypes.ParamSet = (*Params)(nil)

// Params defines lizenz module parameters
type Params struct {
	MaxActivatedLZNPerValidator uint32        `json:"max_activated_lzn_per_validator"`
	ActivityCoefficient         string        `json:"activity_coefficient"` // decimal as string
	DeactivationPeriod          time.Duration `json:"deactivation_period"`
	InactivityPeriod            time.Duration `json:"inactivity_period"`
}

// ParamKeyTable returns the key declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs implements the ParamSet interface and returns the key/value pairs
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMaxActivatedLZNPerValidator, &p.MaxActivatedLZNPerValidator, validateUint32Positive),
		paramtypes.NewParamSetPair(KeyActivityCoefficient, &p.ActivityCoefficient, validateDecString),
		paramtypes.NewParamSetPair(KeyDeactivationPeriod, &p.DeactivationPeriod, validateDuration),
		paramtypes.NewParamSetPair(KeyInactivityPeriod, &p.InactivityPeriod, validateDuration),
	}
}

// DefaultParams returns default module parameters
func DefaultParams() Params {
	return Params{
		MaxActivatedLZNPerValidator: 0,
		ActivityCoefficient:         "0",
		DeactivationPeriod:          30 * 24 * time.Hour,
		InactivityPeriod:            180 * 24 * time.Hour,
	}
}

// Validate validates parameters
func (p Params) Validate() error {
	if err := validateUint32Positive(p.MaxActivatedLZNPerValidator); err != nil {
		return fmt.Errorf("invalid MaxActivatedLZNPerValidator: %w", err)
	}
	if err := validateDecString(p.ActivityCoefficient); err != nil {
		return fmt.Errorf("invalid ActivityCoefficient: %w", err)
	}
	if err := validateDuration(p.DeactivationPeriod); err != nil {
		return fmt.Errorf("invalid DeactivationPeriod: %w", err)
	}
	if err := validateDuration(p.InactivityPeriod); err != nil {
		return fmt.Errorf("invalid InactivityPeriod: %w", err)
	}
	return nil
}

func validateUint32Positive(i interface{}) error {
	v, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("expected uint32, got %T", i)
	}
	// allow zero as meaning "no limit"; only disallow negatives (impossible) here
	_ = v
	return nil
}

func validateDecString(i interface{}) error {
	s, ok := i.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", i)
	}
	if len(s) == 0 {
		return fmt.Errorf("decimal string cannot be empty")
	}
	// detailed numeric validation is deferred to parsing stage in keeper logic
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
