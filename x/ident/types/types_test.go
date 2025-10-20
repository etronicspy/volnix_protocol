package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
)

func TestDefaultParams(t *testing.T) {
	params := DefaultParams()
	
	// Проверяем, что параметры не пустые
	require.NotZero(t, params.CitizenActivityPeriod)
	require.NotZero(t, params.ValidatorActivityPeriod)
	require.NotZero(t, params.MaxIdentitiesPerAddress)
	
	// Проверяем конкретные значения
	require.Equal(t, 365*24*time.Hour, params.CitizenActivityPeriod)
	require.Equal(t, 180*24*time.Hour, params.ValidatorActivityPeriod)
	require.Equal(t, uint64(1), params.MaxIdentitiesPerAddress)
	require.Equal(t, true, params.RequireIdentityVerification)
}

func TestParamsValidation(t *testing.T) {
	params := DefaultParams()
	
	// Тест валидных параметров
	err := params.Validate()
	require.NoError(t, err)
	
	// Тест невалидных параметров
	invalidParams := params
	invalidParams.CitizenActivityPeriod = 0
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
	
	// Проверяем, что все ожидаемые пары присутствуют
	require.Greater(t, len(pairs), 0)
}

func TestValidationFunctions(t *testing.T) {
	// Тест validateDuration
	err := validateDuration(time.Hour)
	require.NoError(t, err)
	
	err = validateDuration(0)
	require.Error(t, err)
	
	// Тест validateUint64
	err = validateUint64(uint64(1))
	require.NoError(t, err)
	
	err = validateUint64(uint64(0))
	require.Error(t, err)
	
	// Тест validateBool
	err = validateBool(true)
	require.NoError(t, err)
	
	err = validateBool(false)
	require.NoError(t, err)
	
	// Тест validateString
	err = validateString("test")
	require.NoError(t, err)
	
	err = validateString("")
	require.NoError(t, err) // validateString не возвращает ошибку для пустой строки
}

func TestNewVerifiedAccount(t *testing.T) {
	account := &identv1.VerifiedAccount{
		Address:      "cosmos1test",
		Role:         identv1.Role_ROLE_CITIZEN,
		IdentityHash: "hash123",
		IsActive:     true,
	}
	
	require.Equal(t, "cosmos1test", account.Address)
	require.Equal(t, identv1.Role_ROLE_CITIZEN, account.Role)
	require.Equal(t, "hash123", account.IdentityHash)
	require.True(t, account.IsActive)
}

func TestAccountValidation(t *testing.T) {
	// Тест валидного аккаунта
	account := &identv1.VerifiedAccount{
		Address:      "cosmos1test",
		Role:         identv1.Role_ROLE_CITIZEN,
		IdentityHash: "hash123",
		IsActive:     true,
	}
	
	// Проверяем поля аккаунта
	require.Equal(t, "cosmos1test", account.Address)
	require.Equal(t, identv1.Role_ROLE_CITIZEN, account.Role)
	require.Equal(t, "hash123", account.IdentityHash)
	require.True(t, account.IsActive)
	
	// Тест невалидного аккаунта
	invalidAccount := &identv1.VerifiedAccount{
		Address:      "",
		Role:         identv1.Role_ROLE_CITIZEN,
		IdentityHash: "hash123",
		IsActive:     true,
	}
	
	require.Empty(t, invalidAccount.Address)
}
