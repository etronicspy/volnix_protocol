package types

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
)

// KeyPrefix returns the key prefix for the consensus module
func KeyPrefix(key string) []byte {
	return []byte(key)
}
