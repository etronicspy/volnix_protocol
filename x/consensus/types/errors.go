package types

import (
	"cosmossdk.io/errors"
)

// consensus module sentinel errors
var (
	ErrInvalidBaseBlockTime         = errors.Register(ModuleName, 1, "invalid base block time")
	ErrInvalidHighActivityThreshold = errors.Register(ModuleName, 2, "invalid high activity threshold")
	ErrInvalidLowActivityThreshold  = errors.Register(ModuleName, 3, "invalid low activity threshold")
	ErrInvalidMinValidatorPower     = errors.Register(ModuleName, 4, "invalid minimum validator power")
	ErrInvalidMaxValidatorPower     = errors.Register(ModuleName, 5, "invalid maximum validator power")
	ErrInvalidActivityDecayRate     = errors.Register(ModuleName, 6, "invalid activity decay rate")
	ErrValidatorNotFound            = errors.Register(ModuleName, 7, "validator not found")
	ErrNoActiveValidators           = errors.Register(ModuleName, 8, "no active validators available")
	ErrZeroTotalPower               = errors.Register(ModuleName, 9, "total validator power is zero")
)
