package types_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()

	require.NotNil(t, params)
	require.Equal(t, "1000000", params.MinLznAmount)
	require.Equal(t, "1000000000", params.MaxLznAmount)
	require.Equal(t, uint32(10), params.MaxActivatedPerValidator)
	require.Equal(t, time.Duration(7*24*time.Hour), params.InactivityPeriod)
	require.Equal(t, time.Duration(24*time.Hour), params.DeactivationPeriod)
	require.True(t, params.RequireIdentityVerification)
	require.Equal(t, "ulzn", params.LznDenom)
	require.Equal(t, "1.0", params.ActivityCoefficient)
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
			name: "invalid min amount - empty",
			params: types.Params{
				MinLznAmount:                "",
				MaxLznAmount:                "1000000000",
				MaxActivatedPerValidator:    10,
				InactivityPeriod:            7 * 24 * time.Hour,
				DeactivationPeriod:          24 * time.Hour,
				RequireIdentityVerification: true,
				LznDenom:                    "ulzn",
				ActivityCoefficient:         "1.0",
			},
			wantErr: true,
		},
		{
			name: "invalid max amount - empty",
			params: types.Params{
				MinLznAmount:                "1000000",
				MaxLznAmount:                "",
				MaxActivatedPerValidator:    10,
				InactivityPeriod:            7 * 24 * time.Hour,
				DeactivationPeriod:          24 * time.Hour,
				RequireIdentityVerification: true,
				LznDenom:                    "ulzn",
				ActivityCoefficient:         "1.0",
			},
			wantErr: true,
		},
		{
			name: "invalid max activated - zero",
			params: types.Params{
				MinLznAmount:                "1000000",
				MaxLznAmount:                "1000000000",
				MaxActivatedPerValidator:    0,
				InactivityPeriod:            7 * 24 * time.Hour,
				DeactivationPeriod:          24 * time.Hour,
				RequireIdentityVerification: true,
				LznDenom:                    "ulzn",
				ActivityCoefficient:         "1.0",
			},
			wantErr: true,
		},
		{
			name: "invalid inactivity period - zero",
			params: types.Params{
				MinLznAmount:                "1000000",
				MaxLznAmount:                "1000000000",
				MaxActivatedPerValidator:    10,
				InactivityPeriod:            0,
				DeactivationPeriod:          24 * time.Hour,
				RequireIdentityVerification: true,
				LznDenom:                    "ulzn",
				ActivityCoefficient:         "1.0",
			},
			wantErr: true,
		},
		{
			name: "invalid deactivation period - zero",
			params: types.Params{
				MinLznAmount:                "1000000",
				MaxLznAmount:                "1000000000",
				MaxActivatedPerValidator:    10,
				InactivityPeriod:            7 * 24 * time.Hour,
				DeactivationPeriod:          0,
				RequireIdentityVerification: true,
				LznDenom:                    "ulzn",
				ActivityCoefficient:         "1.0",
			},
			wantErr: true,
		},
		{
			name: "invalid denom - empty",
			params: types.Params{
				MinLznAmount:                "1000000",
				MaxLznAmount:                "1000000000",
				MaxActivatedPerValidator:    10,
				InactivityPeriod:            7 * 24 * time.Hour,
				DeactivationPeriod:          24 * time.Hour,
				RequireIdentityVerification: true,
				LznDenom:                    "",
				ActivityCoefficient:         "1.0",
			},
			wantErr: true,
		},
		{
			name: "invalid activity coefficient - empty",
			params: types.Params{
				MinLznAmount:                "1000000",
				MaxLznAmount:                "1000000000",
				MaxActivatedPerValidator:    10,
				InactivityPeriod:            7 * 24 * time.Hour,
				DeactivationPeriod:          24 * time.Hour,
				RequireIdentityVerification: true,
				LznDenom:                    "ulzn",
				ActivityCoefficient:         "",
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

func TestParamsString(t *testing.T) {
	params := types.DefaultParams()
	str := fmt.Sprintf("%+v", params)
	require.NotEmpty(t, str)
	require.Contains(t, str, "MinLznAmount")
	require.Contains(t, str, "MaxLznAmount")
}
