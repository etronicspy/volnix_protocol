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
	ErrNoValidators                 = errors.Register(ModuleName, 10, "no validators available")
	ErrInvalidAntAmount             = errors.Register(ModuleName, 11, "invalid ANT amount")
	ErrEmptyValidatorAddress        = errors.Register(ModuleName, 12, "empty validator address")
	ErrUnauthorized                 = errors.Register(ModuleName, 13, "unauthorized")
	// Blind auction errors
	ErrInvalidCommitHash            = errors.Register(ModuleName, 14, "invalid commit hash")
	ErrInvalidBidAmount             = errors.Register(ModuleName, 15, "invalid bid amount")
	ErrAuctionNotInCommitPhase      = errors.Register(ModuleName, 16, "auction is not in commit phase")
	ErrAuctionNotInRevealPhase      = errors.Register(ModuleName, 17, "auction is not in reveal phase")
	ErrBidNotCommitted              = errors.Register(ModuleName, 18, "bid was not committed")
	ErrBidAlreadyRevealed           = errors.Register(ModuleName, 19, "bid already revealed")
	ErrCommitHashMismatch           = errors.Register(ModuleName, 20, "commit hash does not match reveal")
	ErrAuctionNotFound              = errors.Register(ModuleName, 21, "auction not found")
)
