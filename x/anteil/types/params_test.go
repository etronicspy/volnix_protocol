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
	require.Equal(t, "0.001", params.TradingFeeRate)
	require.Equal(t, "100000", params.MinOrderSize)
	require.Equal(t, "100000000", params.MaxOrderSize)
	require.Equal(t, 24*time.Hour, params.OrderExpiry)
	require.True(t, params.RequireIdentityVerification)
	require.Equal(t, "uant", params.AntDenom)
	require.Equal(t, uint32(10), params.MaxOpenOrders)
	require.Equal(t, "0.000001", params.PricePrecision)
	require.Equal(t, types.DefaultEpochLength, params.EpochLength)
	require.Equal(t, "1.0", params.EpochCoefficient)
	require.Equal(t, "0.75", params.EpochCoefficientMin)
	require.Equal(t, "1.50", params.EpochCoefficientMax)
	require.Equal(t, "100000000", params.SupplierEpochAntLimit)
	require.Equal(t, types.DefaultCitizenAntDistributionPeriod, params.CitizenAntDistributionPeriod)
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
			name: "empty min order size",
			params: func() types.Params {
				p := types.DefaultParams()
				p.MinOrderSize = ""
				return p
			}(),
			wantErr: true,
		},
		{
			name: "empty max order size",
			params: func() types.Params {
				p := types.DefaultParams()
				p.MaxOrderSize = ""
				return p
			}(),
			wantErr: true,
		},
		{
			name: "zero order expiry",
			params: func() types.Params {
				p := types.DefaultParams()
				p.OrderExpiry = 0
				return p
			}(),
			wantErr: true,
		},
		{
			name: "empty ant denom",
			params: func() types.Params {
				p := types.DefaultParams()
				p.AntDenom = ""
				return p
			}(),
			wantErr: true,
		},
		{
			name: "zero max open orders",
			params: func() types.Params {
				p := types.DefaultParams()
				p.MaxOpenOrders = 0
				return p
			}(),
			wantErr: true,
		},
		{
			name: "zero epoch length",
			params: func() types.Params {
				p := types.DefaultParams()
				p.EpochLength = 0
				return p
			}(),
			wantErr: true,
		},
		{
			name: "empty epoch coefficient",
			params: func() types.Params {
				p := types.DefaultParams()
				p.EpochCoefficient = ""
				return p
			}(),
			wantErr: true,
		},
		{
			name: "empty supplier epoch ant limit",
			params: func() types.Params {
				p := types.DefaultParams()
				p.SupplierEpochAntLimit = ""
				return p
			}(),
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

func TestValidateDuration(t *testing.T) {
	params := types.DefaultParams()

	err := params.Validate()
	require.NoError(t, err)

	params.OrderExpiry = 0
	err = params.Validate()
	require.Error(t, err)
}

func TestValidateBool(t *testing.T) {
	params := types.DefaultParams()

	params.RequireIdentityVerification = true
	err := params.Validate()
	require.NoError(t, err)

	params.RequireIdentityVerification = false
	err = params.Validate()
	require.NoError(t, err)
}

func TestValidateUint32(t *testing.T) {
	params := types.DefaultParams()

	err := params.Validate()
	require.NoError(t, err)

	params.MaxOpenOrders = 0
	err = params.Validate()
	require.Error(t, err)
}
