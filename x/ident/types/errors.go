package types

import (
	"cosmossdk.io/errors"
)

var (
	// ErrEmptyAddress indicates that the address field is empty
	ErrEmptyAddress = errors.Register(ModuleName, 1, "address cannot be empty")

	// ErrEmptyIdentityHash indicates that the identity hash field is empty
	ErrEmptyIdentityHash = errors.Register(ModuleName, 2, "identity hash cannot be empty")

	// ErrInvalidRole indicates that the role is invalid or unspecified
	ErrInvalidRole = errors.Register(ModuleName, 3, "invalid role specified")

	// ErrAccountAlreadyExists indicates that the account field is empty
	ErrAccountAlreadyExists = errors.Register(ModuleName, 4, "account already exists")

	// ErrAccountNotFound indicates that the account was not found
	ErrAccountNotFound = errors.Register(ModuleName, 5, "account not found")

	// ErrInvalidIdentityHash indicates that the identity hash is invalid
	ErrInvalidIdentityHash = errors.Register(ModuleName, 6, "invalid identity hash")

	// ErrRoleChangeNotAllowed indicates that the role change is not allowed
	ErrRoleChangeNotAllowed = errors.Register(ModuleName, 7, "role change not allowed")

	// Migration errors
	ErrRoleMigrationNotFound  = errors.Register(ModuleName, 8, "role migration not found")
	ErrInvalidMigrationStatus = errors.Register(ModuleName, 9, "invalid migration status")
)
