package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultParams(t *testing.T) {
	params := DefaultParams()
	
	require.Equal(t, 7*24*time.Hour, params.VotingPeriod)
	require.Equal(t, 14*24*time.Hour, params.TimelockPeriod)
	require.Equal(t, "1000000", params.MinDeposit)
	require.Equal(t, "0.4", params.Quorum)
	require.Equal(t, "0.5", params.Threshold)
}

func TestParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  Params
		wantErr bool
	}{
		{
			name:    "valid params",
			params:  DefaultParams(),
			wantErr: false,
		},
		{
			name: "zero voting period",
			params: Params{
				VotingPeriod:  0,
				TimelockPeriod: 14 * 24 * time.Hour,
				MinDeposit:    "1000000",
				Quorum:        "0.4",
				Threshold:     "0.5",
			},
			wantErr: true,
		},
		{
			name: "zero timelock period",
			params: Params{
				VotingPeriod:  7 * 24 * time.Hour,
				TimelockPeriod: 0,
				MinDeposit:    "1000000",
				Quorum:        "0.4",
				Threshold:     "0.5",
			},
			wantErr: true,
		},
		{
			name: "empty min deposit",
			params: Params{
				VotingPeriod:  7 * 24 * time.Hour,
				TimelockPeriod: 14 * 24 * time.Hour,
				MinDeposit:    "",
				Quorum:        "0.4",
				Threshold:     "0.5",
			},
			wantErr: true,
		},
		{
			name: "empty quorum",
			params: Params{
				VotingPeriod:  7 * 24 * time.Hour,
				TimelockPeriod: 14 * 24 * time.Hour,
				MinDeposit:    "1000000",
				Quorum:        "",
				Threshold:     "0.5",
			},
			wantErr: true,
		},
		{
			name: "empty threshold",
			params: Params{
				VotingPeriod:  7 * 24 * time.Hour,
				TimelockPeriod: 14 * 24 * time.Hour,
				MinDeposit:    "1000000",
				Quorum:        "0.4",
				Threshold:     "",
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

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "valid duration",
			input:   time.Hour,
			wantErr: false,
		},
		{
			name:    "invalid type - string",
			input:   "1h",
			wantErr: true,
		},
		{
			name:    "invalid type - int",
			input:   100,
			wantErr: true,
		},
		{
			name:    "zero duration",
			input:   time.Duration(0),
			wantErr: false, // validateDuration only checks type, not value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDuration(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateString(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "valid string",
			input:   "test",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: false, // validateString only checks type, not value
		},
		{
			name:    "invalid type - int",
			input:   100,
			wantErr: true,
		},
		{
			name:    "invalid type - duration",
			input:   time.Hour,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateString(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateDecimal(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "valid decimal string",
			input:   "0.5",
			wantErr: false,
		},
		{
			name:    "valid decimal string - zero",
			input:   "0.0",
			wantErr: false,
		},
		{
			name:    "valid decimal string - one",
			input:   "1.0",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid type - int",
			input:   100,
			wantErr: true,
		},
		{
			name:    "invalid type - duration",
			input:   time.Hour,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDecimal(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParamKeyTable(t *testing.T) {
	keyTable := ParamKeyTable()
	require.NotNil(t, keyTable)
}
