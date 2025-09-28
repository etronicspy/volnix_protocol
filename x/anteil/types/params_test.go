package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultParams(t *testing.T) {
	params := DefaultParams()

	require.Equal(t, "1000000", params.MinAntAmount)
	require.Equal(t, "1000000000", params.MaxAntAmount)
	require.Equal(t, "0.001", params.TradingFeeRate)
	require.Equal(t, "100000", params.MinOrderSize)
	require.Equal(t, "100000000", params.MaxOrderSize)
	require.Equal(t, 24*time.Hour, params.OrderExpiry)
	require.Equal(t, true, params.RequireIdentityVerification)
	require.Equal(t, "uant", params.AntDenom)
	require.Equal(t, uint32(10), params.MaxOpenOrders)
	require.Equal(t, "0.000001", params.PricePrecision)
}

func TestParamsValidation(t *testing.T) {
	params := DefaultParams()

	// Test valid params
	err := params.Validate()
	require.NoError(t, err)

	// Test invalid params - empty MinAntAmount
	invalidParams := params
	invalidParams.MinAntAmount = ""
	err = invalidParams.Validate()
	require.Error(t, err)

	// Test invalid params - empty MaxAntAmount
	invalidParams = params
	invalidParams.MaxAntAmount = ""
	err = invalidParams.Validate()
	require.Error(t, err)

	// Test invalid params - empty TradingFeeRate
	invalidParams = params
	invalidParams.TradingFeeRate = ""
	err = invalidParams.Validate()
	require.Error(t, err)

	// Test invalid params - empty MinOrderSize
	invalidParams = params
	invalidParams.MinOrderSize = ""
	err = invalidParams.Validate()
	require.Error(t, err)

	// Test invalid params - empty MaxOrderSize
	invalidParams = params
	invalidParams.MaxOrderSize = ""
	err = invalidParams.Validate()
	require.Error(t, err)

	// Test invalid params - zero OrderExpiry
	invalidParams = params
	invalidParams.OrderExpiry = 0
	err = invalidParams.Validate()
	require.Error(t, err)

	// Test invalid params - empty AntDenom
	invalidParams = params
	invalidParams.AntDenom = ""
	err = invalidParams.Validate()
	require.Error(t, err)

	// Test invalid params - zero MaxOpenOrders
	invalidParams = params
	invalidParams.MaxOpenOrders = 0
	err = invalidParams.Validate()
	require.Error(t, err)

	// Test invalid params - empty PricePrecision
	invalidParams = params
	invalidParams.PricePrecision = ""
	err = invalidParams.Validate()
	require.Error(t, err)
}

func TestParamKeyTable(t *testing.T) {
	keyTable := ParamKeyTable()
	require.NotNil(t, keyTable)
}

func TestParamSetPairs(t *testing.T) {
	params := DefaultParams()
	pairs := params.ParamSetPairs()

	// Test that all expected pairs are present
	expectedKeys := []string{
		"MinAntAmount",
		"MaxAntAmount",
		"TradingFeeRate",
		"MinOrderSize",
		"MaxOrderSize",
		"OrderExpiry",
		"RequireIdentityVerification",
		"AntDenom",
		"MaxOpenOrders",
		"PricePrecision",
		// New economic parameters
		"MarketMakerRewardRate",
		"StakingRewardRate",
		"LiquidityPoolFee",
		"MaxSlippage",
		"MinLiquidityThreshold",
	}

	require.Len(t, pairs, len(expectedKeys))

	// Test that all keys are present
	keyMap := make(map[string]bool)
	for _, pair := range pairs {
		keyMap[string(pair.Key)] = true
	}

	for _, expectedKey := range expectedKeys {
		require.True(t, keyMap[expectedKey], "Key %s not found", expectedKey)
	}
}

func TestValidationFunctions(t *testing.T) {
	// Test validateString
	err := validateString("test")
	require.NoError(t, err)

	err = validateString(123)
	require.Error(t, err)

	// Test validateDuration
	err = validateDuration(time.Hour)
	require.NoError(t, err)

	err = validateDuration("invalid")
	require.Error(t, err)

	// Test validateBool
	err = validateBool(true)
	require.NoError(t, err)

	err = validateBool("invalid")
	require.Error(t, err)

	// Test validateUint32
	err = validateUint32(uint32(10))
	require.NoError(t, err)

	err = validateUint32(uint32(0))
	require.Error(t, err)

	err = validateUint32("invalid")
	require.Error(t, err)
}
