package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmath "cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyCitizenActivityPeriod        = []byte("CitizenActivityPeriod")
	KeyValidatorActivityPeriod      = []byte("ValidatorActivityPeriod")
	KeyMaxIdentitiesPerAddress      = []byte("MaxIdentitiesPerAddress")
	KeyRequireIdentityVerification  = []byte("RequireIdentityVerification")
	KeyDefaultVerificationProvider  = []byte("DefaultVerificationProvider")
	KeyVerificationCost             = []byte("VerificationCost")
	KeyMigrationFee                 = []byte("MigrationFee")
	KeyRoleChangeFee                = []byte("RoleChangeFee")
)

// Ensure Params implements ParamSet
var _ paramtypes.ParamSet = (*Params)(nil)

// Params defines ident module parameters
type Params struct {
	CitizenActivityPeriod        time.Duration `json:"citizen_activity_period"`
	ValidatorActivityPeriod      time.Duration `json:"validator_activity_period"`
	MaxIdentitiesPerAddress      uint64        `json:"max_identities_per_address"`
	RequireIdentityVerification  bool          `json:"require_identity_verification"`
	DefaultVerificationProvider  string        `json:"default_verification_provider"`
	VerificationCost             sdk.Coin      `json:"verification_cost"`
	MigrationFee                 sdk.Coin      `json:"migration_fee"`
	RoleChangeFee                sdk.Coin      `json:"role_change_fee"`
}

// ParamKeyTable for ident module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs returns the parameter set pairs
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyCitizenActivityPeriod, &p.CitizenActivityPeriod, validateDuration),
		paramtypes.NewParamSetPair(KeyValidatorActivityPeriod, &p.ValidatorActivityPeriod, validateDuration),
		paramtypes.NewParamSetPair(KeyMaxIdentitiesPerAddress, &p.MaxIdentitiesPerAddress, validateUint64),
		paramtypes.NewParamSetPair(KeyRequireIdentityVerification, &p.RequireIdentityVerification, validateBool),
		paramtypes.NewParamSetPair(KeyDefaultVerificationProvider, &p.DefaultVerificationProvider, validateString),
		paramtypes.NewParamSetPair(KeyVerificationCost, &p.VerificationCost, validateCoin),
		paramtypes.NewParamSetPair(KeyMigrationFee, &p.MigrationFee, validateCoin),
		paramtypes.NewParamSetPair(KeyRoleChangeFee, &p.RoleChangeFee, validateCoin),
	}
}

// DefaultParams returns default parameters
func DefaultParams() Params {
	return Params{
		CitizenActivityPeriod:        365 * 24 * time.Hour, // 1 year
		ValidatorActivityPeriod:      180 * 24 * time.Hour, // 6 months
		MaxIdentitiesPerAddress:      1,
		RequireIdentityVerification:  true,
		DefaultVerificationProvider:  "",
		VerificationCost:             sdk.NewCoin("uvx", sdkmath.NewInt(1000000)),
		MigrationFee:                 sdk.NewCoin("uvx", sdkmath.NewInt(500000)),
		RoleChangeFee:                sdk.NewCoin("uvx", sdkmath.NewInt(100000)),
	}
}

// Validate performs basic validation of params
func (p Params) Validate() error {
	if err := validateDuration(p.CitizenActivityPeriod); err != nil {
		return fmt.Errorf("invalid CitizenActivityPeriod: %w", err)
	}
	if err := validateDuration(p.ValidatorActivityPeriod); err != nil {
		return fmt.Errorf("invalid ValidatorActivityPeriod: %w", err)
	}
	if err := validateUint64(p.MaxIdentitiesPerAddress); err != nil {
		return fmt.Errorf("invalid MaxIdentitiesPerAddress: %w", err)
	}
	if err := validateBool(p.RequireIdentityVerification); err != nil {
		return fmt.Errorf("invalid RequireIdentityVerification: %w", err)
	}
	if err := validateString(p.DefaultVerificationProvider); err != nil {
		return fmt.Errorf("invalid DefaultVerificationProvider: %w", err)
	}
	if err := validateCoin(p.VerificationCost); err != nil {
		return fmt.Errorf("invalid VerificationCost: %w", err)
	}
	if err := validateCoin(p.MigrationFee); err != nil {
		return fmt.Errorf("invalid MigrationFee: %w", err)
	}
	if err := validateCoin(p.RoleChangeFee); err != nil {
		return fmt.Errorf("invalid RoleChangeFee: %w", err)
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

func validateCoin(i interface{}) error {
	coin, ok := i.(sdk.Coin)
	if !ok {
		return fmt.Errorf("expected sdk.Coin, got %T", i)
	}
	if err := coin.Validate(); err != nil {
		return fmt.Errorf("invalid coin: %w", err)
	}
	return nil
}
