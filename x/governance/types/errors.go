package types

import (
	"cosmossdk.io/errors"
)

var (
	// ErrEmptyProposer indicates that the proposer field is empty
	ErrEmptyProposer = errors.Register(ModuleName, 1, "proposer cannot be empty")

	// ErrEmptyTitle indicates that the title field is empty
	ErrEmptyTitle = errors.Register(ModuleName, 2, "title cannot be empty")

	// ErrEmptyDescription indicates that the description field is empty
	ErrEmptyDescription = errors.Register(ModuleName, 3, "description cannot be empty")

	// ErrProposalNotFound indicates that the proposal was not found
	ErrProposalNotFound = errors.Register(ModuleName, 4, "proposal not found")

	// ErrProposalAlreadyExists indicates that the proposal already exists
	ErrProposalAlreadyExists = errors.Register(ModuleName, 5, "proposal already exists")

	// ErrInvalidProposalType indicates that the proposal type is invalid
	ErrInvalidProposalType = errors.Register(ModuleName, 6, "invalid proposal type")

	// ErrInvalidVoteOption indicates that the vote option is invalid
	ErrInvalidVoteOption = errors.Register(ModuleName, 7, "invalid vote option")

	// ErrVoteNotFound indicates that the vote was not found
	ErrVoteNotFound = errors.Register(ModuleName, 8, "vote not found")

	// ErrVoteAlreadyExists indicates that the vote already exists
	ErrVoteAlreadyExists = errors.Register(ModuleName, 9, "vote already exists")

	// ErrInsufficientDeposit indicates that the deposit is insufficient
	ErrInsufficientDeposit = errors.Register(ModuleName, 10, "insufficient deposit")

	// ErrProposalNotInVotingPeriod indicates that the proposal is not in voting period
	ErrProposalNotInVotingPeriod = errors.Register(ModuleName, 11, "proposal is not in voting period")

	// ErrProposalNotPassed indicates that the proposal has not passed
	ErrProposalNotPassed = errors.Register(ModuleName, 12, "proposal has not passed")

	// ErrProposalNotExecutable indicates that the proposal is not executable (timelock not expired)
	ErrProposalNotExecutable = errors.Register(ModuleName, 13, "proposal is not executable (timelock not expired)")

	// ErrProposalAlreadyExecuted indicates that the proposal has already been executed
	ErrProposalAlreadyExecuted = errors.Register(ModuleName, 14, "proposal has already been executed")

	// ErrInvalidParameterChange indicates that the parameter change is invalid
	ErrInvalidParameterChange = errors.Register(ModuleName, 15, "invalid parameter change")

	// ErrConstitutionalParameter indicates that the parameter is constitutional and cannot be changed
	ErrConstitutionalParameter = errors.Register(ModuleName, 16, "parameter is constitutional and cannot be changed via governance")
)

