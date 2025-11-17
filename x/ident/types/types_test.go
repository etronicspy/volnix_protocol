package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()

	require.NotNil(t, params)
	require.Greater(t, params.MaxIdentitiesPerAddress, uint64(0))
	require.Greater(t, params.CitizenActivityPeriod, time.Duration(0))
	require.Greater(t, params.ValidatorActivityPeriod, time.Duration(0))
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

func TestNewVerifiedAccount(t *testing.T) {
	account := types.NewVerifiedAccount(
		"cosmos1test",
		identv1.Role_ROLE_CITIZEN,
		"hash123",
	)

	require.NotNil(t, account)
	require.Equal(t, "cosmos1test", account.Address)
	require.Equal(t, identv1.Role_ROLE_CITIZEN, account.Role)
	require.Equal(t, "hash123", account.IdentityHash)
	require.NotNil(t, account.LastActive)
}

func TestIsAccountActive(t *testing.T) {
	params := types.DefaultParams()

	tests := []struct {
		name     string
		account  *identv1.VerifiedAccount
		expected bool
	}{
		{
			name: "active citizen",
			account: &identv1.VerifiedAccount{
				Address:      "cosmos1test",
				Role:         identv1.Role_ROLE_CITIZEN,
				LastActive:   timestamppb.Now(),
				IdentityHash: "hash123",
			},
			expected: true,
		},
		{
			name: "inactive citizen",
			account: &identv1.VerifiedAccount{
				Address:      "cosmos1test",
				Role:         identv1.Role_ROLE_CITIZEN,
				LastActive:   timestamppb.New(time.Now().Add(-400 * 24 * time.Hour)),
				IdentityHash: "hash123",
			},
			expected: false,
		},
		{
			name: "active validator",
			account: &identv1.VerifiedAccount{
				Address:      "cosmos1validator",
				Role:         identv1.Role_ROLE_VALIDATOR,
				LastActive:   timestamppb.Now(),
				IdentityHash: "hash123",
			},
			expected: true,
		},
		{
			name: "inactive validator",
			account: &identv1.VerifiedAccount{
				Address:      "cosmos1validator",
				Role:         identv1.Role_ROLE_VALIDATOR,
				LastActive:   timestamppb.New(time.Now().Add(-200 * 24 * time.Hour)),
				IdentityHash: "hash123",
			},
			expected: false,
		},
		{
			name: "unspecified role",
			account: &identv1.VerifiedAccount{
				Address:      "cosmos1test",
				Role:         identv1.Role_ROLE_UNSPECIFIED,
				LastActive:   timestamppb.Now(),
				IdentityHash: "hash123",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := types.IsAccountActive(tt.account, params)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateAccountActivity(t *testing.T) {
	oldTime := time.Now().Add(-24 * time.Hour)
	account := &identv1.VerifiedAccount{
		Address:      "cosmos1test",
		Role:         identv1.Role_ROLE_CITIZEN,
		LastActive:   timestamppb.New(oldTime),
		IdentityHash: "hash123",
	}

	types.UpdateAccountActivity(account)

	require.True(t, account.LastActive.AsTime().After(oldTime))
}

func TestChangeAccountRole(t *testing.T) {
	oldTime := time.Now().Add(-24 * time.Hour)
	account := &identv1.VerifiedAccount{
		Address:      "cosmos1test",
		Role:         identv1.Role_ROLE_CITIZEN,
		LastActive:   timestamppb.New(oldTime),
		IdentityHash: "hash123",
	}

	types.ChangeAccountRole(account, identv1.Role_ROLE_VALIDATOR)

	require.Equal(t, identv1.Role_ROLE_VALIDATOR, account.Role)
	require.True(t, account.LastActive.AsTime().After(oldTime))
}

func TestValidateAccount(t *testing.T) {
	tests := []struct {
		name    string
		account *identv1.VerifiedAccount
		wantErr error
	}{
		{
			name: "valid account",
			account: &identv1.VerifiedAccount{
				Address:      "cosmos1test",
				Role:         identv1.Role_ROLE_CITIZEN,
				IdentityHash: "hash123",
			},
			wantErr: nil,
		},
		{
			name: "empty address",
			account: &identv1.VerifiedAccount{
				Address:      "",
				Role:         identv1.Role_ROLE_CITIZEN,
				IdentityHash: "hash123",
			},
			wantErr: types.ErrEmptyAddress,
		},
		{
			name: "empty identity hash",
			account: &identv1.VerifiedAccount{
				Address:      "cosmos1test",
				Role:         identv1.Role_ROLE_CITIZEN,
				IdentityHash: "",
			},
			wantErr: types.ErrEmptyIdentityHash,
		},
		{
			name: "unspecified role",
			account: &identv1.VerifiedAccount{
				Address:      "cosmos1test",
				Role:         identv1.Role_ROLE_UNSPECIFIED,
				IdentityHash: "hash123",
			},
			wantErr: types.ErrInvalidRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateAccount(tt.account)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetVerifiedAccountKey(t *testing.T) {
	address := "cosmos1test"
	key := types.GetVerifiedAccountKey(address)
	require.NotNil(t, key)
	require.Contains(t, string(key), address)
	
	// Test different addresses produce different keys
	key2 := types.GetVerifiedAccountKey("cosmos1test2")
	require.NotEqual(t, key, key2)
}

func TestGetRoleMigrationKey(t *testing.T) {
	fromAddress := "cosmos1from"
	toAddress := "cosmos1to"
	key := types.GetRoleMigrationKey(fromAddress, toAddress)
	require.NotNil(t, key)
	require.Contains(t, string(key), fromAddress)
	require.Contains(t, string(key), toAddress)
	
	// Test different migrations produce different keys
	key2 := types.GetRoleMigrationKey("cosmos1from2", "cosmos1to2")
	require.NotEqual(t, key, key2)
}

func TestGetNullifierKey(t *testing.T) {
	nullifier := []byte("nullifier123")
	key := types.GetNullifierKey(nullifier)
	require.NotNil(t, key)
	require.NotEmpty(t, key)
	
	// Test different nullifiers produce different keys
	key2 := types.GetNullifierKey([]byte("nullifier456"))
	require.NotEqual(t, key, key2)
}

func TestGetProviderKey(t *testing.T) {
	providerID := "provider123"
	key := types.GetProviderKey(providerID)
	require.NotNil(t, key)
	require.Contains(t, string(key), providerID)
	
	// Test different providers produce different keys
	key2 := types.GetProviderKey("provider456")
	require.NotEqual(t, key, key2)
}

func TestGetAccreditationKey(t *testing.T) {
	accreditationHash := "hash123"
	key := types.GetAccreditationKey(accreditationHash)
	require.NotNil(t, key)
	require.Contains(t, string(key), accreditationHash)
	
	// Test different hashes produce different keys
	key2 := types.GetAccreditationKey("hash456")
	require.NotEqual(t, key, key2)
}

func TestGetVerificationRecordKey(t *testing.T) {
	address := "cosmos1test"
	key := types.GetVerificationRecordKey(address)
	require.NotNil(t, key)
	require.Contains(t, string(key), address)
	
	// Test different addresses produce different keys
	key2 := types.GetVerificationRecordKey("cosmos1test2")
	require.NotEqual(t, key, key2)
}

func TestGetProofKey(t *testing.T) {
	proofHash := []byte("proof123")
	key := types.GetProofKey(proofHash)
	require.NotNil(t, key)
	require.NotEmpty(t, key)
	
	// Test different proofs produce different keys
	key2 := types.GetProofKey([]byte("proof456"))
	require.NotEqual(t, key, key2)
}
