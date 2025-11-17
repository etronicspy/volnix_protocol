package types

import (
	"fmt"
	
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "consensus"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_consensus"
)

var (
	// ParamsKey defines the key for consensus module parameters
	ParamsKey = []byte("Params")

	// ValidatorKey defines the key for validator data
	ValidatorKey = "Validator"

	// BlockCreatorKey defines the key for block creator data
	BlockCreatorKey = "BlockCreator"

	// BurnProofKey defines the key for burn proof data
	BurnProofKey = "BurnProof"

	// ActivityScoreKey defines the key for activity score data
	ActivityScoreKey = "ActivityScore"

	// HalvingInfoKey defines the key for halving information
	HalvingInfoKey = []byte("HalvingInfo")
	
	// BlockTimeKeyPrefix defines the prefix for block time keys
	BlockTimeKeyPrefix = []byte{0x10}
	
	// AverageBlockTimeKey defines the key for average block time
	AverageBlockTimeKey = []byte("AverageBlockTime")

	// ConsensusStateKey defines the key for consensus state
	ConsensusStateKey = []byte("ConsensusState")

	// ValidatorWeightKey defines the key for validator weight
	ValidatorWeightKey = "ValidatorWeight"
	
	// BlindAuctionKey defines the key for blind auction data
	BlindAuctionKey = "BlindAuction"
	
	// BidHistoryKeyPrefix defines the prefix for bid history keys (anti-manipulation)
	BidHistoryKeyPrefix = []byte{0x11}
)

// Key prefixes
var (
	KeyValidatorPrefix       = []byte(ValidatorKey)
	KeyBlockCreatorPrefix    = []byte(BlockCreatorKey)
	KeyValidatorWeightPrefix = []byte(ValidatorWeightKey)
	KeyBlindAuctionPrefix    = []byte(BlindAuctionKey)
)

// KeyPrefix returns the key prefix for the consensus module
func KeyPrefix(key string) []byte {
	return []byte(key)
}

// GetValidatorKey returns the key for a validator
func GetValidatorKey(validator string) []byte {
	return append(KeyValidatorPrefix, []byte(validator)...)
}

// GetBlockCreatorKey returns the key for a block creator
func GetBlockCreatorKey(height uint64) []byte {
	return append(KeyBlockCreatorPrefix, []byte(fmt.Sprintf("%d", height))...)
}

// GetValidatorWeightKey returns the key for a validator weight
func GetValidatorWeightKey(validator string) []byte {
	return append(KeyValidatorWeightPrefix, []byte(validator)...)
}

// KeyHalvingInfo returns the key for halving info
func KeyHalvingInfo() []byte {
	return HalvingInfoKey
}

// GetBlockTimeKey returns the key for a block's time
func GetBlockTimeKey(height uint64) []byte {
	return append(BlockTimeKeyPrefix, sdk.Uint64ToBigEndian(height)...)
}

// KeyConsensusState returns the key for consensus state
func KeyConsensusState() []byte {
	return ConsensusStateKey
}

// GetBlindAuctionKey returns the key for a blind auction at a specific height
func GetBlindAuctionKey(height uint64) []byte {
	return append(KeyBlindAuctionPrefix, []byte(fmt.Sprintf("%d", height))...)
}

// GetBidHistoryKey returns the key for a validator's bid history
func GetBidHistoryKey(validator string) []byte {
	return append(BidHistoryKeyPrefix, []byte(validator)...)
}
