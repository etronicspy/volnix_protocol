package types

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyCitizenInactivityPeriod      = []byte("CitizenInactivityPeriod")
	KeyValidatorInactivityPeriod    = []byte("ValidatorInactivityPeriod")
	KeyMaxCitizenAccounts          = []byte("MaxCitizenAccounts")
	KeyMaxValidatorAccounts        = []byte("MaxValidatorAccounts")
	KeyRequireIdentityVerification = []byte("RequireIdentityVerification")
	KeyIdentityProviderAddress     = []byte("IdentityProviderAddress")
)

// Ensure Params implements ParamSet
var _ paramtypes.ParamSet = (*Params)(nil)

// Params defines ident module parameters
type Params struct {
	CitizenInactivityPeriod      time.Duration `json:"citizen_inactivity_period"`
	ValidatorInactivityPeriod    time.Duration `json:"validator_inactivity_period"`
	MaxCitizenAccounts          uint64        `json:"max_citizen_accounts"`
	MaxValidatorAccounts        uint64        `json:"max_validator_accounts"`
	RequireIdentityVerification bool          `json:"require_identity_verification"`
	IdentityProviderAddress     string        `json:"identity_provider_address"`
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
		paramtypes.NewParamSetPair(KeyMaxCitizenAccounts, &p.MaxCitizenAccounts, validateUint64),
		paramtypes.NewParamSetPair(KeyMaxValidatorAccounts, &p.MaxValidatorAccounts, validateUint64),
		paramtypes.NewParamSetPair(KeyRequireIdentityVerification, &p.RequireIdentityVerification, validateBool),
		paramtypes.NewParamSetPair(KeyIdentityProviderAddress, &p.IdentityProviderAddress, validateString),
	}
}

// DefaultParams returns default parameters
func DefaultParams() Params {
	return Params{
		CitizenInactivityPeriod:      365 * 24 * time.Hour, // 1 year
		ValidatorInactivityPeriod:    180 * 24 * time.Hour, // 6 months
		MaxCitizenAccounts:           10000,
		MaxValidatorAccounts:         1000,
		RequireIdentityVerification:  true,
		IdentityProviderAddress:      "",
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
	if err := validateUint64(p.MaxCitizenAccounts); err != nil {
		return fmt.Errorf("invalid MaxCitizenAccounts: %w", err)
	}
	if err := validateUint64(p.MaxValidatorAccounts); err != nil {
		return fmt.Errorf("invalid MaxValidatorAccounts: %w", err)
	}
	if err := validateBool(p.RequireIdentityVerification); err != nil {
		return fmt.Errorf("invalid RequireIdentityVerification: %w", err)
	}
	if err := validateString(p.IdentityProviderAddress); err != nil {
		return fmt.Errorf("invalid IdentityProviderAddress: %w", err)
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

func validateUint64(i interface{}) error {
	u, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("expected uint64, got %T", i)
	}
	if u == 0 {
		return fmt.Errorf("value must be greater than 0")
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

func validateString(i interface{}) error {
	_, ok := i.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", i)
	}
	// Allow empty string for optional fields
	return nil
}
