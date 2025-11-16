package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "governance"

	// StoreKey is the KVStore key for the governance module
	StoreKey = ModuleName

	// RouterKey is the msg router key for the governance module
	RouterKey = ModuleName

	// QuerierRoute is the querier route for the governance module
	QuerierRoute = ModuleName
)

var (
	// ProposalKeyPrefix defines the prefix for proposal keys
	ProposalKeyPrefix = []byte{0x01}

	// VoteKeyPrefix defines the prefix for vote keys
	VoteKeyPrefix = []byte{0x02}

	// ProposalIDKey defines the key for storing the next proposal ID
	ProposalIDKey = []byte{0x03}
)

// GetProposalKey returns the key for a proposal
func GetProposalKey(proposalID uint64) []byte {
	return append(ProposalKeyPrefix, sdk.Uint64ToBigEndian(proposalID)...)
}

// GetVoteKey returns the key for a vote
func GetVoteKey(proposalID uint64, voter string) []byte {
	return append(VoteKeyPrefix, append(sdk.Uint64ToBigEndian(proposalID), []byte(voter)...)...)
}

