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

	// PerHeightBurnKey defines the key prefix for per-height burn data
	PerHeightBurnKey = "PerHeightBurn"

	// HeightBurnSummaryKey defines the key prefix for height burn summaries
	HeightBurnSummaryKey = "HeightBurnSummary"

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
	
	// ConsensusMappingKey defines the key prefix for consensus pubkey -> account mappings
	ConsensusMappingKey = "ConsensusMapping"

	// TargetBlockTimeKey defines the key for the dynamic target block time
	TargetBlockTimeKey = []byte("TargetBlockTime")

	// AntPurchaseKey defines the key prefix for per-validator ANT purchase totals
	AntPurchaseKey = "AntPurchase"

	// BlockCreatorKeyPrefix defines the key prefix for block creator records by height
	BlockCreatorKeyPrefix = "BlockCreator"

	// BlindAuctionKeyPrefix defines the key prefix for blind auction records by height
	BlindAuctionKeyPrefix = "BlindAuction"

	// BidHistoryKeyPrefix defines the key prefix for per-validator bid history
	BidHistoryKeyPrefix = "BidHistory"

	// EpochHeightKey defines the key for the current epoch start height
	EpochHeightKey = []byte("EpochHeight")

	// DefaultEpochLength is the number of blocks in one MOA epoch
	DefaultEpochLength = uint64(1000)
)

// Key prefixes
var (
	KeyValidatorPrefix        = []byte(ValidatorKey)
	KeyValidatorWeightPrefix  = []byte(ValidatorWeightKey)
	KeyPerHeightBurnPrefix    = []byte(PerHeightBurnKey)
	KeyHeightBurnSummaryPrefix = []byte(HeightBurnSummaryKey)
	KeyConsensusMappingPrefix = []byte(ConsensusMappingKey)
	AntPurchaseKeyPrefix      = []byte(AntPurchaseKey)
	KeyBlockCreatorPrefix     = []byte(BlockCreatorKeyPrefix)
	KeyBlindAuctionPrefix     = []byte(BlindAuctionKeyPrefix)
	KeyBidHistoryPrefix       = []byte(BidHistoryKeyPrefix)
)

// KeyPrefix returns the key prefix for the consensus module
func KeyPrefix(key string) []byte {
	return []byte(key)
}

// GetValidatorKey returns the key for a validator
func GetValidatorKey(validator string) []byte {
	return append(KeyValidatorPrefix, []byte(validator)...)
}

// GetValidatorWeightKey returns the key for a validator weight
func GetValidatorWeightKey(validator string) []byte {
	return append(KeyValidatorWeightPrefix, []byte(validator)...)
}

// GetPerHeightBurnKey returns the key for a validator's burn at a specific height
func GetPerHeightBurnKey(validator string, height uint64) []byte {
	prefix := append(KeyPerHeightBurnPrefix, []byte(validator)...)
	return append(prefix, sdk.Uint64ToBigEndian(height)...)
}

// GetHeightBurnSummaryKey returns the key for a height's burn summary
func GetHeightBurnSummaryKey(height uint64) []byte {
	return append(KeyHeightBurnSummaryPrefix, sdk.Uint64ToBigEndian(height)...)
}

// GetConsensusMappingKey returns the key for a consensus address mapping
func GetConsensusMappingKey(consensusAddr string) []byte {
	return append(KeyConsensusMappingPrefix, []byte(consensusAddr)...)
}

// GetAntPurchaseKey returns the key for a validator's ANT purchase total in the current epoch
func GetAntPurchaseKey(validator string) []byte {
	return append(AntPurchaseKeyPrefix, []byte(validator)...)
}

// KeyHalvingInfo returns the key for halving info
func KeyHalvingInfo() []byte {
	return HalvingInfoKey
}

// GetBlockTimeKey returns the key for a block's time
func GetBlockTimeKey(height uint64) []byte {
	return append(BlockTimeKeyPrefix, sdk.Uint64ToBigEndian(height)...)
}

// GetBlockCreatorKey returns the store key for the block creator at the given height.
func GetBlockCreatorKey(height uint64) []byte {
	return append(KeyBlockCreatorPrefix, []byte(fmt.Sprintf("%d", height))...)
}

// GetBlindAuctionKey returns the store key for the blind auction at the given height.
func GetBlindAuctionKey(height uint64) []byte {
	return append(KeyBlindAuctionPrefix, []byte(fmt.Sprintf("%d", height))...)
}

// GetBidHistoryKey returns the store key for a validator's bid history.
func GetBidHistoryKey(validator string) []byte {
	return append(KeyBidHistoryPrefix, []byte(validator)...)
}

// KeyConsensusState returns the key for consensus state
func KeyConsensusState() []byte {
	return ConsensusStateKey
}

