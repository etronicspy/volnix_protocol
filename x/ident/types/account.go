package types

import (
	"time"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewVerifiedAccount creates a new VerifiedAccount instance
func NewVerifiedAccount(address string, role identv1.Role, identityHash string) *identv1.VerifiedAccount {
	now := timestamppb.Now()
	return &identv1.VerifiedAccount{
		Address:      address,
		Role:         role,
		LastActive:   now,
		IdentityHash: identityHash,
	}
}

// IsAccountActive checks if the account is active based on inactivity period
func IsAccountActive(acc *identv1.VerifiedAccount, params Params) bool {
	lastActive := acc.LastActive.AsTime()
	now := time.Now()

	var inactivityPeriod time.Duration
	switch acc.Role {
	case identv1.Role_ROLE_CITIZEN:
		inactivityPeriod = params.CitizenActivityPeriod
	case identv1.Role_ROLE_VALIDATOR:
		inactivityPeriod = params.ValidatorActivityPeriod
	default:
		return false
	}

	return now.Sub(lastActive) <= inactivityPeriod
}

// UpdateAccountActivity updates the last active timestamp
func UpdateAccountActivity(acc *identv1.VerifiedAccount) {
	acc.LastActive = timestamppb.Now()
}

// ChangeAccountRole changes the role of the account
func ChangeAccountRole(acc *identv1.VerifiedAccount, newRole identv1.Role) {
	acc.Role = newRole
	UpdateAccountActivity(acc)
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

	return nil
}
