package types

import (
	"cosmossdk.io/errors"
)

var (
	// ErrEmptyValidator indicates that the validator field is empty
	ErrEmptyValidator = errors.Register(ModuleName, 1, "validator cannot be empty")

	// ErrEmptyAmount indicates that the amount field is empty
	ErrEmptyAmount = errors.Register(ModuleName, 2, "amount cannot be empty")

	// ErrEmptyIdentityHash indicates that the identity hash field is empty
	ErrEmptyIdentityHash = errors.Register(ModuleName, 3, "identity hash cannot be empty")

	// ErrEmptyReason indicates that the reason field is empty
	ErrEmptyReason = errors.Register(ModuleName, 4, "reason cannot be empty")

	// ErrEmptyCurrentMOA indicates that the current MOA field is empty
	ErrEmptyCurrentMOA = errors.Register(ModuleName, 5, "current MOA cannot be empty")

	// ErrEmptyRequiredMOA indicates that the required MOA field is empty
	ErrEmptyRequiredMOA = errors.Register(ModuleName, 6, "required MOA cannot be empty")

	// ErrLizenzAlreadyExists indicates that the LZN already exists
	ErrLizenzAlreadyExists = errors.Register(ModuleName, 7, "LZN already exists")

	// ErrLizenzNotFound indicates that the LZN was not found
	ErrLizenzNotFound = errors.Register(ModuleName, 8, "LZN not found")

	// ErrInvalidAmount indicates that the amount is invalid
	ErrInvalidAmount = errors.Register(ModuleName, 9, "invalid amount")

	// ErrExceedsMaxActivated indicates that the amount exceeds maximum allowed
	ErrExceedsMaxActivated = errors.Register(ModuleName, 10, "amount exceeds maximum allowed")

	// ErrBelowMinAmount indicates that the amount is below minimum required
	ErrBelowMinAmount = errors.Register(ModuleName, 11, "amount below minimum required")

	// ErrExceedsMaxLznActivation indicates that validator exceeds 33% limit of total activated LZN
	ErrExceedsMaxLznActivation = errors.Register(ModuleName, 12, "validator exceeds maximum LZN activation limit (33% of total pool)")
	
	// ErrIdentityNotVerified indicates that validator has not verified their identity
	ErrIdentityNotVerified = errors.Register(ModuleName, 13, "validator must have verified identity to activate LZN")
	
	// ErrInvalidRoleForLizenz indicates that validator does not have VALIDATOR role
	ErrInvalidRoleForLizenz = errors.Register(ModuleName, 14, "only validators with ROLE_VALIDATOR can activate LZN")
)
