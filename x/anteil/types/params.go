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
	
	// New economic parameter keys
	KeyMarketMakerRewardRate = []byte("MarketMakerRewardRate")
	KeyStakingRewardRate     = []byte("StakingRewardRate")
	KeyLiquidityPoolFee      = []byte("LiquidityPoolFee")
	KeyMaxSlippage           = []byte("MaxSlippage")
	KeyMinLiquidityThreshold = []byte("MinLiquidityThreshold")
	
	// Citizen ANT distribution parameter keys
	KeyCitizenAntRewardRate      = []byte("CitizenAntRewardRate")
	KeyCitizenAntAccumulationLimit = []byte("CitizenAntAccumulationLimit")
	KeyCitizenAntDistributionPeriod = []byte("CitizenAntDistributionPeriod")
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
	
	// New economic parameters
	MarketMakerRewardRate string `json:"market_maker_reward_rate"`
	StakingRewardRate     string `json:"staking_reward_rate"`
	LiquidityPoolFee      string `json:"liquidity_pool_fee"`
	MaxSlippage           string `json:"max_slippage"`
	MinLiquidityThreshold uint64 `json:"min_liquidity_threshold"`
	
	// Citizen ANT distribution parameters
	CitizenAntRewardRate       string        `json:"citizen_ant_reward_rate"`        // Base rate (e.g., "10" ANT per day)
	CitizenAntAccumulationLimit string        `json:"citizen_ant_accumulation_limit"` // Max accumulation (e.g., "1000" ANT)
	CitizenAntDistributionPeriod time.Duration `json:"citizen_ant_distribution_period"` // Distribution period (e.g., 24 hours)
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
		
		// New economic parameter pairs
		paramtypes.NewParamSetPair(KeyMarketMakerRewardRate, &p.MarketMakerRewardRate, validateString),
		paramtypes.NewParamSetPair(KeyStakingRewardRate, &p.StakingRewardRate, validateString),
		paramtypes.NewParamSetPair(KeyLiquidityPoolFee, &p.LiquidityPoolFee, validateString),
		paramtypes.NewParamSetPair(KeyMaxSlippage, &p.MaxSlippage, validateString),
		paramtypes.NewParamSetPair(KeyMinLiquidityThreshold, &p.MinLiquidityThreshold, validateUint64),
		
		// Citizen ANT distribution parameter pairs
		paramtypes.NewParamSetPair(KeyCitizenAntRewardRate, &p.CitizenAntRewardRate, validateString),
		paramtypes.NewParamSetPair(KeyCitizenAntAccumulationLimit, &p.CitizenAntAccumulationLimit, validateString),
		paramtypes.NewParamSetPair(KeyCitizenAntDistributionPeriod, &p.CitizenAntDistributionPeriod, validateDuration),
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
		
		// New economic parameters
		MarketMakerRewardRate: "0.002", // 0.2%
		StakingRewardRate:     "0.05",  // 5%
		LiquidityPoolFee:      "0.003", // 0.3%
		MaxSlippage:           "0.05",  // 5%
		MinLiquidityThreshold: 1000000, // 1 ANT in micro units
		
		// Citizen ANT distribution parameters (default: 10 ANT per day, 1000 ANT limit, 24h period)
		CitizenAntRewardRate:       "10000000",        // 10 ANT in micro units (10 * 1,000,000)
		CitizenAntAccumulationLimit: "1000000000",     // 1000 ANT in micro units (1000 * 1,000,000)
		CitizenAntDistributionPeriod: 24 * time.Hour, // 24 hours
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
	
	// Validate new economic parameters
	if p.MarketMakerRewardRate == "" {
		return fmt.Errorf("MarketMakerRewardRate cannot be empty")
	}
	if p.StakingRewardRate == "" {
		return fmt.Errorf("StakingRewardRate cannot be empty")
	}
	if p.LiquidityPoolFee == "" {
		return fmt.Errorf("LiquidityPoolFee cannot be empty")
	}
	if p.MaxSlippage == "" {
		return fmt.Errorf("MaxSlippage cannot be empty")
	}
	if p.MinLiquidityThreshold == 0 {
		return fmt.Errorf("MinLiquidityThreshold must be greater than 0")
	}
	
	// Validate citizen ANT distribution parameters
	if p.CitizenAntRewardRate == "" {
		return fmt.Errorf("CitizenAntRewardRate cannot be empty")
	}
	if p.CitizenAntAccumulationLimit == "" {
		return fmt.Errorf("CitizenAntAccumulationLimit cannot be empty")
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
