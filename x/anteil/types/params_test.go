package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()

	require.NotNil(t, params)
	require.Equal(t, "1000000", params.MinAntAmount)
	require.Equal(t, "1000000000", params.MaxAntAmount)
	require.Equal(t, "0.001", params.TradingFeeRate)
	require.Equal(t, "100000", params.MinOrderSize)
	require.Equal(t, "100000000", params.MaxOrderSize)
	require.Equal(t, 24*time.Hour, params.OrderExpiry)
	require.True(t, params.RequireIdentityVerification)
	require.Equal(t, "uant", params.AntDenom)
	require.Equal(t, uint32(10), params.MaxOpenOrders)
	require.Equal(t, "0.000001", params.PricePrecision)
	require.Equal(t, "0.002", params.MarketMakerRewardRate)
	require.Equal(t, "0.05", params.StakingRewardRate)
	require.Equal(t, "0.003", params.LiquidityPoolFee)
	require.Equal(t, "0.05", params.MaxSlippage)
	require.Equal(t, uint64(1000000), params.MinLiquidityThreshold)
}

func TestParamsValidate(t *testing.T) {
	tests := []struct {
		name    string
		params  types.Params
		wantErr bool
	}{
		{
			name:    "valid params",
			params:  types.DefaultParams(),
			wantErr: false,
		},
		{
			name: "empty min ant amount",
			params: types.Params{
				MinAntAmount:                "",
				MaxAntAmount:                "1000000000",
				TradingFeeRate:              "0.001",
				MinOrderSize:                "100000",
				MaxOrderSize:                "100000000",
				OrderExpiry:                 24 * time.Hour,
				RequireIdentityVerification: true,
				AntDenom:                    "uant",
				MaxOpenOrders:               10,
				PricePrecision:              "0.000001",
				MarketMakerRewardRate:       "0.002",
				StakingRewardRate:           "0.05",
				LiquidityPoolFee:            "0.003",
				MaxSlippage:                 "0.05",
				MinLiquidityThreshold:       1000000,
			},
			wantErr: true,
		},
		{
			name: "empty max ant amount",
			params: types.Params{
				MinAntAmount:                "1000000",
				MaxAntAmount:                "",
				TradingFeeRate:              "0.001",
				MinOrderSize:                "100000",
				MaxOrderSize:                "100000000",
				OrderExpiry:                 24 * time.Hour,
				RequireIdentityVerification: true,
				AntDenom:                    "uant",
				MaxOpenOrders:               10,
				PricePrecision:              "0.000001",
				MarketMakerRewardRate:       "0.002",
				StakingRewardRate:           "0.05",
				LiquidityPoolFee:            "0.003",
				MaxSlippage:                 "0.05",
				MinLiquidityThreshold:       1000000,
			},
			wantErr: true,
		},
		{
			name: "zero order expiry",
			params: types.Params{
				MinAntAmount:                "1000000",
				MaxAntAmount:                "1000000000",
				TradingFeeRate:              "0.001",
				MinOrderSize:                "100000",
				MaxOrderSize:                "100000000",
				OrderExpiry:                 0,
				RequireIdentityVerification: true,
				AntDenom:                    "uant",
				MaxOpenOrders:               10,
				PricePrecision:              "0.000001",
				MarketMakerRewardRate:       "0.002",
				StakingRewardRate:           "0.05",
				LiquidityPoolFee:            "0.003",
				MaxSlippage:                 "0.05",
				MinLiquidityThreshold:       1000000,
			},
			wantErr: true,
		},
		{
			name: "empty ant denom",
			params: types.Params{
				MinAntAmount:                "1000000",
				MaxAntAmount:                "1000000000",
				TradingFeeRate:              "0.001",
				MinOrderSize:                "100000",
				MaxOrderSize:                "100000000",
				OrderExpiry:                 24 * time.Hour,
				RequireIdentityVerification: true,
				AntDenom:                    "",
				MaxOpenOrders:               10,
				PricePrecision:              "0.000001",
				MarketMakerRewardRate:       "0.002",
				StakingRewardRate:           "0.05",
				LiquidityPoolFee:            "0.003",
				MaxSlippage:                 "0.05",
				MinLiquidityThreshold:       1000000,
			},
			wantErr: true,
		},
		{
			name: "zero max open orders",
			params: types.Params{
				MinAntAmount:                "1000000",
				MaxAntAmount:                "1000000000",
				TradingFeeRate:              "0.001",
				MinOrderSize:                "100000",
				MaxOrderSize:                "100000000",
				OrderExpiry:                 24 * time.Hour,
				RequireIdentityVerification: true,
				AntDenom:                    "uant",
				MaxOpenOrders:               0,
				PricePrecision:              "0.000001",
				MarketMakerRewardRate:       "0.002",
				StakingRewardRate:           "0.05",
				LiquidityPoolFee:            "0.003",
				MaxSlippage:                 "0.05",
				MinLiquidityThreshold:       1000000,
			},
			wantErr: true,
		},
		{
			name: "zero min liquidity threshold",
			params: types.Params{
				MinAntAmount:                "1000000",
				MaxAntAmount:                "1000000000",
				TradingFeeRate:              "0.001",
				MinOrderSize:                "100000",
				MaxOrderSize:                "100000000",
				OrderExpiry:                 24 * time.Hour,
				RequireIdentityVerification: true,
				AntDenom:                    "uant",
				MaxOpenOrders:               10,
				PricePrecision:              "0.000001",
				MarketMakerRewardRate:       "0.002",
				StakingRewardRate:           "0.05",
				LiquidityPoolFee:            "0.003",
				MaxSlippage:                 "0.05",
				MinLiquidityThreshold:       0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParamKeyTable(t *testing.T) {
	table := types.ParamKeyTable()
	require.NotNil(t, table)
}
