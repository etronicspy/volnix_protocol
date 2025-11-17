package types

import (
	"fmt"
	"strconv"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	// KeyMaxActivatedPerValidator defines the key for max activated per validator
	KeyMaxActivatedPerValidator = []byte("MaxActivatedPerValidator")

	// KeyActivityCoefficient defines the key for activity coefficient
	KeyActivityCoefficient = []byte("ActivityCoefficient")

	// KeyDeactivationPeriod defines the key for deactivation period
	KeyDeactivationPeriod = []byte("DeactivationPeriod")

	// KeyInactivityPeriod defines the key for inactivity period
	KeyInactivityPeriod = []byte("InactivityPeriod")

	// KeyMinLznAmount defines the key for minimum LZN amount
	KeyMinLznAmount = []byte("MinLznAmount")

	// KeyMaxLznAmount defines the key for maximum LZN amount
	KeyMaxLznAmount = []byte("MaxLznAmount")

	// KeyRequireIdentityVerification defines the key for identity verification requirement
	KeyRequireIdentityVerification = []byte("RequireIdentityVerification")

	// KeyLznDenom defines the key for LZN denomination
	KeyLznDenom = []byte("LznDenom")
)

// ParamKeyTable returns the parameter key table
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// Params defines the parameters for the lizenz module
type Params struct {
	MaxActivatedPerValidator    uint32        `json:"max_activated_per_validator"`
	ActivityCoefficient         string        `json:"activity_coefficient"`
	DeactivationPeriod          time.Duration `json:"deactivation_period"`
	InactivityPeriod            time.Duration `json:"inactivity_period"`
	MinLznAmount                string        `json:"min_lzn_amount"`
	MaxLznAmount                string        `json:"max_lzn_amount"`
	RequireIdentityVerification bool          `json:"require_identity_verification"`
	LznDenom                    string        `json:"lzn_denom"`
}

// ParamSetPairs returns the parameter set pairs
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMaxActivatedPerValidator, &p.MaxActivatedPerValidator, validateUint32),
		paramtypes.NewParamSetPair(KeyActivityCoefficient, &p.ActivityCoefficient, validateString),
		paramtypes.NewParamSetPair(KeyDeactivationPeriod, &p.DeactivationPeriod, validateDuration),
		paramtypes.NewParamSetPair(KeyInactivityPeriod, &p.InactivityPeriod, validateDuration),
		paramtypes.NewParamSetPair(KeyMinLznAmount, &p.MinLznAmount, validateString),
		paramtypes.NewParamSetPair(KeyMaxLznAmount, &p.MaxLznAmount, validateString),
		paramtypes.NewParamSetPair(KeyRequireIdentityVerification, &p.RequireIdentityVerification, validateBool),
		paramtypes.NewParamSetPair(KeyLznDenom, &p.LznDenom, validateString),
	}
}

// DefaultParams returns the default parameters for the lizenz module
func DefaultParams() Params {
	return Params{
		MaxActivatedPerValidator:    10,
		ActivityCoefficient:         "1.0",
		DeactivationPeriod:          24 * time.Hour,     // 24 hours
		InactivityPeriod:            7 * 24 * time.Hour, // 7 days
		MinLznAmount:                "1000000",          // 1 LZN in micro units
		MaxLznAmount:                "1000000000",       // 1000 LZN in micro units
		RequireIdentityVerification: true,
		LznDenom:                    "ulzn",
	}
}

// Validate validates the parameters
func (p *Params) Validate() error {
	if p.MaxActivatedPerValidator == 0 {
		return fmt.Errorf("MaxActivatedPerValidator must be greater than 0")
	}
	if p.ActivityCoefficient == "" {
		return fmt.Errorf("ActivityCoefficient cannot be empty")
	}
	// Validate ActivityCoefficient is a valid float between 0.0 and 1.0 (or > 1.0 for bonus)
	activityCoeff, err := strconv.ParseFloat(p.ActivityCoefficient, 64)
	if err != nil {
		return fmt.Errorf("ActivityCoefficient must be a valid number: %w", err)
	}
	if activityCoeff < 0 {
		return fmt.Errorf("ActivityCoefficient must be >= 0, got %f", activityCoeff)
	}
	if p.DeactivationPeriod <= 0 {
		return fmt.Errorf("DeactivationPeriod must be greater than 0")
	}
	if p.InactivityPeriod <= 0 {
		return fmt.Errorf("InactivityPeriod must be greater than 0")
	}
	if p.MinLznAmount == "" {
		return fmt.Errorf("MinLznAmount cannot be empty")
	}
	// Validate MinLznAmount is a valid positive integer
	minAmount, err := strconv.ParseInt(p.MinLznAmount, 10, 64)
	if err != nil {
		return fmt.Errorf("MinLznAmount must be a valid integer: %w", err)
	}
	if minAmount <= 0 {
		return fmt.Errorf("MinLznAmount must be greater than 0")
	}
	if p.MaxLznAmount == "" {
		return fmt.Errorf("MaxLznAmount cannot be empty")
	}
	// Validate MaxLznAmount is a valid positive integer
	maxAmount, err := strconv.ParseInt(p.MaxLznAmount, 10, 64)
	if err != nil {
		return fmt.Errorf("MaxLznAmount must be a valid integer: %w", err)
	}
	if maxAmount <= 0 {
		return fmt.Errorf("MaxLznAmount must be greater than 0")
	}
	// Validate MaxLznAmount >= MinLznAmount
	if maxAmount < minAmount {
		return fmt.Errorf("MaxLznAmount (%d) must be >= MinLznAmount (%d)", maxAmount, minAmount)
	}
	if p.LznDenom == "" {
		return fmt.Errorf("LznDenom cannot be empty")
	}
	return nil
}

// Validation functions
func validateUint32(i interface{}) error {
	u, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("expected uint32, got %T", i)
	}
	if u == 0 {
		return fmt.Errorf("value must be greater than 0")
	}
	return nil
}

func validateString(i interface{}) error {
	_, ok := i.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", i)
	}
	return nil
}

func validateDuration(i interface{}) error {
	_, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("expected time.Duration, got %T", i)
	}
	return nil
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("expected bool, got %T", i)
	}
	return nil
}
