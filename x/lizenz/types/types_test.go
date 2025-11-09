package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

func TestNewActivatedLizenz(t *testing.T) {
	lizenz := types.NewActivatedLizenz("cosmos1validator", "1000000", "hash123")

	require.NotNil(t, lizenz)
	require.Equal(t, "cosmos1validator", lizenz.Validator)
	require.Equal(t, "1000000", lizenz.Amount)
	require.Equal(t, "hash123", lizenz.IdentityHash)
	require.NotNil(t, lizenz.ActivationTime)
	require.NotNil(t, lizenz.LastActivity)
}

func TestNewDeactivatingLizenz(t *testing.T) {
	lizenz := types.NewDeactivatingLizenz("cosmos1validator", "1000000", "inactivity")

	require.NotNil(t, lizenz)
	require.Equal(t, "cosmos1validator", lizenz.Validator)
	require.Equal(t, "1000000", lizenz.Amount)
	require.Equal(t, "inactivity", lizenz.Reason)
	require.NotNil(t, lizenz.DeactivationStart)
	require.NotNil(t, lizenz.DeactivationEnd)
}

func TestNewMOAStatus(t *testing.T) {
	status := types.NewMOAStatus("cosmos1validator", "1000000", "500000")

	require.NotNil(t, status)
	require.Equal(t, "cosmos1validator", status.Validator)
	require.Equal(t, "1000000", status.CurrentMoa)
	require.Equal(t, "500000", status.RequiredMoa)
	require.True(t, status.IsCompliant)
	require.NotNil(t, status.LastActivity)
}

func TestIsActivatedLizenzValid(t *testing.T) {
	tests := []struct {
		name    string
		lizenz  *lizenzv1.ActivatedLizenz
		wantErr error
	}{
		{
			name: "valid lizenz",
			lizenz: &lizenzv1.ActivatedLizenz{
				Validator:            "cosmos1validator",
				Amount:               "1000000",
				ActivationTime:       timestamppb.Now(),
				LastActivity:         timestamppb.Now(),
				IsEligibleForRewards: true,
				IdentityHash:         "hash123",
			},
			wantErr: nil,
		},
		{
			name: "empty validator",
			lizenz: &lizenzv1.ActivatedLizenz{
				Validator:            "",
				Amount:               "1000000",
				ActivationTime:       timestamppb.Now(),
				LastActivity:         timestamppb.Now(),
				IsEligibleForRewards: true,
				IdentityHash:         "hash123",
			},
			wantErr: types.ErrEmptyValidator,
		},
		{
			name: "empty amount",
			lizenz: &lizenzv1.ActivatedLizenz{
				Validator:            "cosmos1validator",
				Amount:               "",
				ActivationTime:       timestamppb.Now(),
				LastActivity:         timestamppb.Now(),
				IsEligibleForRewards: true,
				IdentityHash:         "hash123",
			},
			wantErr: types.ErrEmptyAmount,
		},
		{
			name: "empty identity hash",
			lizenz: &lizenzv1.ActivatedLizenz{
				Validator:            "cosmos1validator",
				Amount:               "1000000",
				ActivationTime:       timestamppb.Now(),
				LastActivity:         timestamppb.Now(),
				IsEligibleForRewards: true,
				IdentityHash:         "",
			},
			wantErr: types.ErrEmptyIdentityHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.IsActivatedLizenzValid(tt.lizenz)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsDeactivatingLizenzValid(t *testing.T) {
	tests := []struct {
		name    string
		lizenz  *lizenzv1.DeactivatingLizenz
		wantErr error
	}{
		{
			name: "valid deactivating lizenz",
			lizenz: &lizenzv1.DeactivatingLizenz{
				Validator:         "cosmos1validator",
				Amount:            "1000000",
				DeactivationStart: timestamppb.Now(),
				DeactivationEnd:   timestamppb.New(time.Now().Add(30 * 24 * time.Hour)),
				Reason:            "inactivity",
			},
			wantErr: nil,
		},
		{
			name: "empty validator",
			lizenz: &lizenzv1.DeactivatingLizenz{
				Validator:         "",
				Amount:            "1000000",
				DeactivationStart: timestamppb.Now(),
				DeactivationEnd:   timestamppb.New(time.Now().Add(30 * 24 * time.Hour)),
				Reason:            "inactivity",
			},
			wantErr: types.ErrEmptyValidator,
		},
		{
			name: "empty amount",
			lizenz: &lizenzv1.DeactivatingLizenz{
				Validator:         "cosmos1validator",
				Amount:            "",
				DeactivationStart: timestamppb.Now(),
				DeactivationEnd:   timestamppb.New(time.Now().Add(30 * 24 * time.Hour)),
				Reason:            "inactivity",
			},
			wantErr: types.ErrEmptyAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.IsDeactivatingLizenzValid(tt.lizenz)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsMOAStatusValid(t *testing.T) {
	tests := []struct {
		name    string
		status  *lizenzv1.MOAStatus
		wantErr error
	}{
		{
			name: "valid MOA status",
			status: &lizenzv1.MOAStatus{
				Validator:    "cosmos1validator",
				CurrentMoa:   "1000000",
				RequiredMoa:  "500000",
				LastActivity: timestamppb.Now(),
				IsCompliant:  true,
			},
			wantErr: nil,
		},
		{
			name: "empty validator",
			status: &lizenzv1.MOAStatus{
				Validator:    "",
				CurrentMoa:   "1000000",
				RequiredMoa:  "500000",
				LastActivity: timestamppb.Now(),
				IsCompliant:  true,
			},
			wantErr: types.ErrEmptyValidator,
		},
		{
			name: "empty current MOA",
			status: &lizenzv1.MOAStatus{
				Validator:    "cosmos1validator",
				CurrentMoa:   "",
				RequiredMoa:  "500000",
				LastActivity: timestamppb.Now(),
				IsCompliant:  true,
			},
			wantErr: types.ErrEmptyCurrentMOA,
		},
		{
			name: "empty required MOA",
			status: &lizenzv1.MOAStatus{
				Validator:    "cosmos1validator",
				CurrentMoa:   "1000000",
				RequiredMoa:  "",
				LastActivity: timestamppb.Now(),
				IsCompliant:  true,
			},
			wantErr: types.ErrEmptyRequiredMOA,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.IsMOAStatusValid(tt.status)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateActivatedLizenzActivity(t *testing.T) {
	oldTime := time.Now().Add(-24 * time.Hour)
	lizenz := &lizenzv1.ActivatedLizenz{
		Validator:            "cosmos1validator",
		Amount:               "1000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.New(oldTime),
		IsEligibleForRewards: true,
		IdentityHash:         "hash123",
	}

	types.UpdateActivatedLizenzActivity(lizenz)

	require.True(t, lizenz.LastActivity.AsTime().After(oldTime))
}

func TestUpdateMOAStatusActivity(t *testing.T) {
	oldTime := time.Now().Add(-24 * time.Hour)
	status := &lizenzv1.MOAStatus{
		Validator:    "cosmos1validator",
		CurrentMoa:   "1000000",
		RequiredMoa:  "500000",
		LastActivity: timestamppb.New(oldTime),
		IsCompliant:  true,
	}

	newMOA := "1500000"
	types.UpdateMOAStatusActivity(status, newMOA)

	require.True(t, status.LastActivity.AsTime().After(oldTime))
	require.Equal(t, newMOA, status.CurrentMoa)
}

func TestCalculateMOA(t *testing.T) {
	params := types.DefaultParams()

	moa, err := types.CalculateMOA("test_data", params)
	require.NoError(t, err)
	require.NotEmpty(t, moa)
}

func TestCalculateLizenzPrice(t *testing.T) {
	// Test basic license price calculation
	price, err := types.CalculateLizenzPrice("basic", 30)
	require.NoError(t, err)
	require.NotEmpty(t, price)

	// Test premium license
	premiumPrice, err := types.CalculateLizenzPrice("premium", 30)
	require.NoError(t, err)
	require.NotEmpty(t, premiumPrice)

	// Test enterprise license
	enterprisePrice, err := types.CalculateLizenzPrice("enterprise", 60)
	require.NoError(t, err)
	require.NotEmpty(t, enterprisePrice)
}
