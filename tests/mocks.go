package tests

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
)

// ============================================================================
// MockIdentKeeper - Mock for Identity keeper
// ============================================================================

type MockIdentKeeper struct {
	Accounts []*identv1.VerifiedAccount
	Err      error
}

func NewMockIdentKeeper() *MockIdentKeeper {
	return &MockIdentKeeper{
		Accounts: []*identv1.VerifiedAccount{},
	}
}

func NewMockIdentKeeperWithAccounts(accounts []*identv1.VerifiedAccount) *MockIdentKeeper {
	return &MockIdentKeeper{
		Accounts: accounts,
	}
}

func NewMockIdentKeeperWithRoles(citizens, validators, guests int) *MockIdentKeeper {
	accounts := []*identv1.VerifiedAccount{}
	
	// Add citizens
	for i := 0; i < citizens; i++ {
		acc := NewTestVerifiedAccountCustom(
			fmt.Sprintf("cosmos1citizen%d", i),
			identv1.Role_ROLE_CITIZEN,
			fmt.Sprintf("hash-citizen-%d", i),
		)
		accounts = append(accounts, acc)
	}
	
	// Add validators
	for i := 0; i < validators; i++ {
		acc := NewTestVerifiedAccountCustom(
			fmt.Sprintf("cosmos1validator%d", i),
			identv1.Role_ROLE_VALIDATOR,
			fmt.Sprintf("hash-validator-%d", i),
		)
		accounts = append(accounts, acc)
	}
	
	// Add guests
	for i := 0; i < guests; i++ {
		acc := NewTestVerifiedAccountCustom(
			fmt.Sprintf("cosmos1guest%d", i),
			identv1.Role_ROLE_GUEST,
			fmt.Sprintf("hash-guest-%d", i),
		)
		accounts = append(accounts, acc)
	}
	
	return &MockIdentKeeper{Accounts: accounts}
}

func (m *MockIdentKeeper) GetAllVerifiedAccounts(ctx sdk.Context) ([]*identv1.VerifiedAccount, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Accounts, nil
}

func (m *MockIdentKeeper) AddAccount(account *identv1.VerifiedAccount) {
	m.Accounts = append(m.Accounts, account)
}

func (m *MockIdentKeeper) SetError(err error) {
	m.Err = err
}

// ============================================================================
// MockAnteilKeeper - Mock for Anteil keeper
// ============================================================================

type MockAnteilKeeper struct {
	BurnedUsers []string
	BurnErr     error
	Positions   map[string]*anteilv1.UserPosition
}

func NewMockAnteilKeeper() *MockAnteilKeeper {
	return &MockAnteilKeeper{
		BurnedUsers: []string{},
		Positions:   make(map[string]*anteilv1.UserPosition),
	}
}

func (m *MockAnteilKeeper) BurnAntFromUser(ctx sdk.Context, user string) error {
	if m.BurnErr != nil {
		return m.BurnErr
	}
	m.BurnedUsers = append(m.BurnedUsers, user)
	return nil
}

func (m *MockAnteilKeeper) GetUserPosition(ctx sdk.Context, user string) (*anteilv1.UserPosition, error) {
	if pos, ok := m.Positions[user]; ok {
		return pos, nil
	}
	return nil, fmt.Errorf("position not found")
}

func (m *MockAnteilKeeper) SetUserPosition(user string, position *anteilv1.UserPosition) {
	m.Positions[user] = position
}

func (m *MockAnteilKeeper) SetBurnError(err error) {
	m.BurnErr = err
}

// ============================================================================
// MockLizenzKeeper - Mock for Lizenz keeper
// ============================================================================

type MockLizenzKeeper struct {
	ActivatedLizenz []interface{}
	MOACompliance   map[string]float64
	TotalLZN        string
	Errors          map[string]error
}

func NewMockLizenzKeeper() *MockLizenzKeeper {
	return &MockLizenzKeeper{
		ActivatedLizenz: []interface{}{},
		MOACompliance:   make(map[string]float64),
		Errors:          make(map[string]error),
		TotalLZN:        "1000000000",
	}
}

func (m *MockLizenzKeeper) GetAllActivatedLizenz(ctx sdk.Context) ([]interface{}, error) {
	if err, ok := m.Errors["GetAllActivatedLizenz"]; ok {
		return nil, err
	}
	return m.ActivatedLizenz, nil
}

func (m *MockLizenzKeeper) GetMOACompliance(ctx sdk.Context, validator string) (float64, error) {
	if err, ok := m.Errors["GetMOACompliance"]; ok {
		return 0, err
	}
	if compliance, ok := m.MOACompliance[validator]; ok {
		return compliance, nil
	}
	return 1.0, nil // Default: fully compliant
}

func (m *MockLizenzKeeper) GetTotalActivatedLZN(ctx sdk.Context) (string, error) {
	if err, ok := m.Errors["GetTotalActivatedLZN"]; ok {
		return "", err
	}
	return m.TotalLZN, nil
}

func (m *MockLizenzKeeper) AddActivatedLizenz(lizenz interface{}) {
	m.ActivatedLizenz = append(m.ActivatedLizenz, lizenz)
}

func (m *MockLizenzKeeper) SetMOACompliance(validator string, compliance float64) {
	m.MOACompliance[validator] = compliance
}

func (m *MockLizenzKeeper) SetError(method string, err error) {
	m.Errors[method] = err
}

// ============================================================================
// MockBankKeeper - Mock for Bank keeper
// ============================================================================

type MockBankKeeper struct {
	MintedCoins map[string]sdk.Coins
	SentCoins   map[string]sdk.Coins
	MintErrors  map[string]error
	SendErrors  map[string]error
}

func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{
		MintedCoins: make(map[string]sdk.Coins),
		SentCoins:   make(map[string]sdk.Coins),
		MintErrors:  make(map[string]error),
		SendErrors:  make(map[string]error),
	}
}

func (m *MockBankKeeper) MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	if err, ok := m.MintErrors[moduleName]; ok {
		return err
	}
	m.MintedCoins[moduleName] = m.MintedCoins[moduleName].Add(amt...)
	return nil
}

func (m *MockBankKeeper) SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	recipient := recipientAddr.String()
	if err, ok := m.SendErrors[recipient]; ok {
		return err
	}
	m.SentCoins[recipient] = m.SentCoins[recipient].Add(amt...)
	return nil
}

func (m *MockBankKeeper) GetMintedCoins(moduleName string) sdk.Coins {
	return m.MintedCoins[moduleName]
}

func (m *MockBankKeeper) GetSentCoins(address string) sdk.Coins {
	return m.SentCoins[address]
}

func (m *MockBankKeeper) SetMintError(moduleName string, err error) {
	m.MintErrors[moduleName] = err
}

func (m *MockBankKeeper) SetSendError(address string, err error) {
	m.SendErrors[address] = err
}
