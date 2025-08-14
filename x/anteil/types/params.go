package types

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	// KeyMinAntAmount defines the key for minimum ANT amount
	KeyMinAntAmount = []byte("MinAntAmount")

	// KeyMaxAntAmount defines the key for maximum ANT amount
	KeyMaxAntAmount = []byte("MaxAntAmount")

	// KeyTradingFeeRate defines the key for trading fee rate
	KeyTradingFeeRate = []byte("TradingFeeRate")

	// KeyMinOrderSize defines the key for minimum order size
	KeyMinOrderSize = []byte("MinOrderSize")

	// KeyMaxOrderSize defines the key for maximum order size
	KeyMaxOrderSize = []byte("MaxOrderSize")

	// KeyOrderExpiry defines the key for order expiry duration
	KeyOrderExpiry = []byte("OrderExpiry")

	// KeyRequireIdentityVerification defines the key for identity verification requirement
	KeyRequireIdentityVerification = []byte("RequireIdentityVerification")

	// KeyAntDenom defines the key for ANT denomination
	KeyAntDenom = []byte("AntDenom")

	// KeyMaxOpenOrders defines the key for maximum open orders
	KeyMaxOpenOrders = []byte("MaxOpenOrders")

	// KeyPricePrecision defines the key for price precision
	KeyPricePrecision = []byte("PricePrecision")
)

// ParamKeyTable returns the parameter key table
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// Params defines the parameters for the anteil module
type Params struct {
	MinAntAmount                string        `json:"min_ant_amount"`
	MaxAntAmount                string        `json:"max_ant_amount"`
	TradingFeeRate              string        `json:"trading_fee_rate"`
	MinOrderSize                string        `json:"min_order_size"`
	MaxOrderSize                string        `json:"max_order_size"`
	OrderExpiry                 time.Duration `json:"order_expiry"`
	RequireIdentityVerification bool          `json:"require_identity_verification"`
	AntDenom                    string        `json:"ant_denom"`
	MaxOpenOrders               uint32        `json:"max_open_orders"`
	PricePrecision              string        `json:"price_precision"`
}

// ParamSetPairs returns the parameter set pairs
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinAntAmount, &p.MinAntAmount, validateString),
		paramtypes.NewParamSetPair(KeyMaxAntAmount, &p.MaxAntAmount, validateString),
		paramtypes.NewParamSetPair(KeyTradingFeeRate, &p.TradingFeeRate, validateString),
		paramtypes.NewParamSetPair(KeyMinOrderSize, &p.MinOrderSize, validateString),
		paramtypes.NewParamSetPair(KeyMaxOrderSize, &p.MaxOrderSize, validateString),
		paramtypes.NewParamSetPair(KeyOrderExpiry, &p.OrderExpiry, validateDuration),
		paramtypes.NewParamSetPair(KeyRequireIdentityVerification, &p.RequireIdentityVerification, validateBool),
		paramtypes.NewParamSetPair(KeyAntDenom, &p.AntDenom, validateString),
		paramtypes.NewParamSetPair(KeyMaxOpenOrders, &p.MaxOpenOrders, validateUint32),
		paramtypes.NewParamSetPair(KeyPricePrecision, &p.PricePrecision, validateString),
	}
}

// DefaultParams returns the default parameters for the anteil module
func DefaultParams() Params {
	return Params{
		MinAntAmount:                "1000000",      // 1 ANT in micro units
		MaxAntAmount:                "1000000000",   // 1000 ANT in micro units
		TradingFeeRate:              "0.001",        // 0.1%
		MinOrderSize:                "100000",       // 0.1 ANT in micro units
		MaxOrderSize:                "100000000",    // 100 ANT in micro units
		OrderExpiry:                 24 * time.Hour, // 24 hours
		RequireIdentityVerification: true,
		AntDenom:                    "uant",
		MaxOpenOrders:               10,
		PricePrecision:              "0.000001", // 6 decimal places
	}
}

// Validate validates the parameters
func (p *Params) Validate() error {
	if p.MinAntAmount == "" {
		return fmt.Errorf("MinAntAmount cannot be empty")
	}
	if p.MaxAntAmount == "" {
		return fmt.Errorf("MaxAntAmount cannot be empty")
	}
	if p.TradingFeeRate == "" {
		return fmt.Errorf("TradingFeeRate cannot be empty")
	}
	if p.MinOrderSize == "" {
		return fmt.Errorf("MinOrderSize cannot be empty")
	}
	if p.MaxOrderSize == "" {
		return fmt.Errorf("MaxOrderSize cannot be empty")
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
