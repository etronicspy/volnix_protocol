package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewVerifiedAccount creates a new VerifiedAccount instance.
// blockTime must come from ctx.BlockTime() for determinism across nodes.
func NewVerifiedAccount(address string, role identv1.Role, identityHash string, blockTime ...time.Time) *identv1.VerifiedAccount {
	var now *timestamppb.Timestamp
	if len(blockTime) > 0 {
		now = timestamppb.New(blockTime[0])
	} else {
		now = timestamppb.Now()
	}
	return &identv1.VerifiedAccount{
		Address:      address,
		Role:         role,
		LastActive:   now,
		IdentityHash: identityHash,
		IsActive:     true, // New accounts are active by default
	}
}

// IsAccountActive checks if the account is active based on inactivity period.
// blockTime must come from ctx.BlockTime() for determinism across nodes.
func IsAccountActive(acc *identv1.VerifiedAccount, params Params, blockTime time.Time) bool {
	lastActive := acc.LastActive.AsTime()

	var inactivityPeriod time.Duration
	switch acc.Role {
	case identv1.Role_ROLE_SUPPLIER:
		inactivityPeriod = params.MoaSupplierWindow
	case identv1.Role_ROLE_VALIDATOR:
		inactivityPeriod = params.MoaValidatorWindow
	default:
		return false
	}

	return blockTime.Sub(lastActive) <= inactivityPeriod
}

// UpdateAccountActivity updates the last active timestamp.
// blockTime must come from ctx.BlockTime() for determinism across nodes.
func UpdateAccountActivity(acc *identv1.VerifiedAccount, blockTime ...time.Time) {
	var now *timestamppb.Timestamp
	if len(blockTime) > 0 {
		now = timestamppb.New(blockTime[0])
	} else {
		now = timestamppb.Now()
	}
	acc.LastActive = now
}

// ChangeAccountRole changes the role of the account.
// blockTime must come from ctx.BlockTime() for determinism across nodes.
func ChangeAccountRole(acc *identv1.VerifiedAccount, newRole identv1.Role, blockTime ...time.Time) {
	acc.Role = newRole
	UpdateAccountActivity(acc, blockTime...)
}

// ValidateAccount performs basic validation on the account
func ValidateAccount(acc *identv1.VerifiedAccount) error {
	if acc.Address == "" {
		return ErrEmptyAddress
	}

	if acc.IdentityHash == "" {
		return ErrEmptyIdentityHash
	}

	if acc.Role == identv1.Role_ROLE_UNSPECIFIED {
		return ErrInvalidRole
	}

	// Validate address format (bech32) - reject clearly invalid addresses
	if _, err := sdk.AccAddressFromBech32(acc.Address); err != nil {
		return ErrInvalidAddress
	}

	return nil
}
