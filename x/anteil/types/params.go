package types

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// DefaultEpochLength is the default epoch duration (1 week).
const DefaultEpochLength = 7 * 24 * time.Hour

// DefaultCitizenAntDistributionPeriod kept for backward compat; semantics change to epoch.
const DefaultCitizenAntDistributionPeriod = 1 * time.Minute

var (
	KeyMinOrderSize                = []byte("MinOrderSize")
	KeyMaxOrderSize                = []byte("MaxOrderSize")
	KeyTradingFeeRate              = []byte("TradingFeeRate")
	KeyOrderExpiry                 = []byte("OrderExpiry")
	KeyRequireIdentityVerification = []byte("RequireIdentityVerification")
	KeyAntDenom                    = []byte("AntDenom")
	KeyMaxOpenOrders               = []byte("MaxOpenOrders")
	KeyPricePrecision              = []byte("PricePrecision")

	KeyEpochLength                   = []byte("EpochLength")
	KeyEpochCoefficient              = []byte("EpochCoefficient")
	KeyEpochCoefficientMin           = []byte("EpochCoefficientMin")
	KeyEpochCoefficientMax           = []byte("EpochCoefficientMax")
	KeySupplierEpochAntLimit         = []byte("SupplierEpochAntLimit")
	KeyCitizenAntDistributionPeriod  = []byte("CitizenAntDistributionPeriod")
)

// ParamKeyTable returns the parameter key table
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// Params defines the parameters for the anteil module
type Params struct {
	MinOrderSize                string        `json:"min_order_size"`
	MaxOrderSize                string        `json:"max_order_size"`
	TradingFeeRate              string        `json:"trading_fee_rate"`
	OrderExpiry                 time.Duration `json:"order_expiry"`
	RequireIdentityVerification bool          `json:"require_identity_verification"`
	AntDenom                    string        `json:"ant_denom"`
	MaxOpenOrders               uint32        `json:"max_open_orders"`
	PricePrecision              string        `json:"price_precision"`

	EpochLength                  time.Duration `json:"epoch_length"`
	EpochCoefficient             string        `json:"epoch_coefficient"`
	EpochCoefficientMin          string        `json:"epoch_coefficient_min"`
	EpochCoefficientMax          string        `json:"epoch_coefficient_max"`
	SupplierEpochAntLimit        string        `json:"supplier_epoch_ant_limit"`
	CitizenAntDistributionPeriod time.Duration `json:"citizen_ant_distribution_period"`
}

// ParamSetPairs returns the parameter set pairs
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinOrderSize, &p.MinOrderSize, validateString),
		paramtypes.NewParamSetPair(KeyMaxOrderSize, &p.MaxOrderSize, validateString),
		paramtypes.NewParamSetPair(KeyTradingFeeRate, &p.TradingFeeRate, validateString),
		paramtypes.NewParamSetPair(KeyOrderExpiry, &p.OrderExpiry, validateDuration),
		paramtypes.NewParamSetPair(KeyRequireIdentityVerification, &p.RequireIdentityVerification, validateBool),
		paramtypes.NewParamSetPair(KeyAntDenom, &p.AntDenom, validateString),
		paramtypes.NewParamSetPair(KeyMaxOpenOrders, &p.MaxOpenOrders, validateUint32),
		paramtypes.NewParamSetPair(KeyPricePrecision, &p.PricePrecision, validateString),
		paramtypes.NewParamSetPair(KeyEpochLength, &p.EpochLength, validateDuration),
		paramtypes.NewParamSetPair(KeyEpochCoefficient, &p.EpochCoefficient, validateString),
		paramtypes.NewParamSetPair(KeyEpochCoefficientMin, &p.EpochCoefficientMin, validateString),
		paramtypes.NewParamSetPair(KeyEpochCoefficientMax, &p.EpochCoefficientMax, validateString),
		paramtypes.NewParamSetPair(KeySupplierEpochAntLimit, &p.SupplierEpochAntLimit, validateString),
		paramtypes.NewParamSetPair(KeyCitizenAntDistributionPeriod, &p.CitizenAntDistributionPeriod, validateDuration),
	}
}

// DefaultParams returns the default parameters for the anteil module
func DefaultParams() Params {
	return Params{
		MinOrderSize:                "100000",
		MaxOrderSize:                "100000000",
		TradingFeeRate:              "0.001",
		OrderExpiry:                 24 * time.Hour,
		RequireIdentityVerification: true,
		AntDenom:                    "uant",
		MaxOpenOrders:               10,
		PricePrecision:              "0.000001",

		EpochLength:                  DefaultEpochLength,
		EpochCoefficient:             "1.0",
		EpochCoefficientMin:          "0.75",
		EpochCoefficientMax:          "1.50",
		SupplierEpochAntLimit:        "100000000",
		CitizenAntDistributionPeriod: DefaultCitizenAntDistributionPeriod,
	}
}

// Validate validates the parameters
func (p *Params) Validate() error {
	if p.MinOrderSize == "" {
		return fmt.Errorf("MinOrderSize cannot be empty")
	}
	if p.MaxOrderSize == "" {
		return fmt.Errorf("MaxOrderSize cannot be empty")
	}
	if p.TradingFeeRate == "" {
		return fmt.Errorf("TradingFeeRate cannot be empty")
	}
	if p.OrderExpiry <= 0 {
		return fmt.Errorf("OrderExpiry must be greater than 0")
	}
	if p.AntDenom == "" {
		return fmt.Errorf("AntDenom cannot be empty")
	}
	if p.MaxOpenOrders == 0 {
		return fmt.Errorf("MaxOpenOrders must be greater than 0")
	}
	if p.PricePrecision == "" {
		return fmt.Errorf("PricePrecision cannot be empty")
	}
	if p.EpochLength <= 0 {
		return fmt.Errorf("EpochLength must be greater than 0")
	}
	if p.EpochCoefficient == "" {
		return fmt.Errorf("EpochCoefficient cannot be empty")
	}
	if p.EpochCoefficientMin == "" {
		return fmt.Errorf("EpochCoefficientMin cannot be empty")
	}
	if p.EpochCoefficientMax == "" {
		return fmt.Errorf("EpochCoefficientMax cannot be empty")
	}
	if p.SupplierEpochAntLimit == "" {
		return fmt.Errorf("SupplierEpochAntLimit cannot be empty")
	}
	if p.CitizenAntDistributionPeriod <= 0 {
		return fmt.Errorf("CitizenAntDistributionPeriod must be greater than 0")
	}
	return nil
}

// Validation functions
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
