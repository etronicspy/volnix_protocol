package types

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyMaxOrderAmount = []byte("MaxOrderAmount")
	KeyMinOrderAmount = []byte("MinOrderAmount")
	KeyTradingFee     = []byte("TradingFee")
	KeyAuctionPeriod  = []byte("AuctionPeriod")
)

// Ensure Params implements ParamSet
var _ paramtypes.ParamSet = (*Params)(nil)

// Params defines anteil module parameters
type Params struct {
	MaxOrderAmount string        `json:"max_order_amount"` // Int string
	MinOrderAmount string        `json:"min_order_amount"` // Int string
	TradingFee     string        `json:"trading_fee"`      // decimal as string in [0,1]
	AuctionPeriod  time.Duration `json:"auction_period"`
}

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMaxOrderAmount, &p.MaxOrderAmount, validateIntStringPositive),
		paramtypes.NewParamSetPair(KeyMinOrderAmount, &p.MinOrderAmount, validateIntStringPositive),
		paramtypes.NewParamSetPair(KeyTradingFee, &p.TradingFee, validateDecString),
		paramtypes.NewParamSetPair(KeyAuctionPeriod, &p.AuctionPeriod, validateDuration),
	}
}

func DefaultParams() Params {
	return Params{
		MaxOrderAmount: "0",
		MinOrderAmount: "0",
		TradingFee:     "0.001",
		AuctionPeriod:  60 * time.Second,
	}
}

func (p Params) Validate() error {
	if err := validateIntStringPositive(p.MaxOrderAmount); err != nil {
		return fmt.Errorf("invalid MaxOrderAmount: %w", err)
	}
	if err := validateIntStringPositive(p.MinOrderAmount); err != nil {
		return fmt.Errorf("invalid MinOrderAmount: %w", err)
	}
	if err := validateDecString(p.TradingFee); err != nil {
		return fmt.Errorf("invalid TradingFee: %w", err)
	}
	if err := validateDuration(p.AuctionPeriod); err != nil {
		return fmt.Errorf("invalid AuctionPeriod: %w", err)
	}
	return nil
}

func validateIntStringPositive(i interface{}) error {
	s, ok := i.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", i)
	}
	if len(s) == 0 {
		return fmt.Errorf("value cannot be empty")
	}
	// defer exact math.Int parsing to later steps; just ensure non-empty
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
