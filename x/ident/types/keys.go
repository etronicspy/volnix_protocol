package types

const (
	// ModuleName defines the module name
	ModuleName = "ident"

	// StoreKey is the KVStore key for the ident module
	StoreKey = ModuleName

	// RouterKey is the msg router key for the ident module (kept for compatibility)
	RouterKey = ModuleName

	// QuerierRoute is the querier route for the ident module
	QuerierRoute = ModuleName
)

var (
	// VerifiedAccountKeyPrefix defines the prefix for verified account keys
	VerifiedAccountKeyPrefix = []byte{0x01}

	// RoleMigrationKeyPrefix defines the prefix for role migration keys
	RoleMigrationKeyPrefix = []byte{0x02}

	// NullifierKeyPrefix defines the prefix for nullifier keys
	NullifierKeyPrefix = []byte{0x03}
	
	// ProviderKeyPrefix defines the prefix for verification provider keys
	ProviderKeyPrefix = []byte{0x04}
	
	// AccreditationKeyPrefix defines the prefix for accreditation keys
	AccreditationKeyPrefix = []byte{0x05}
	
	// VerificationRecordKeyPrefix defines the prefix for verification record keys
	VerificationRecordKeyPrefix = []byte{0x06}
	
	// ProofKeyPrefix defines the prefix for proof keys (anti-replay)
	ProofKeyPrefix = []byte{0x07}
)

// GetVerifiedAccountKey returns the key for a verified account
func GetVerifiedAccountKey(address string) []byte {
	return append(VerifiedAccountKeyPrefix, []byte(address)...)
}

// GetRoleMigrationKey returns the key for a role migration
func GetRoleMigrationKey(fromAddress, toAddress string) []byte {
	return append(RoleMigrationKeyPrefix, []byte(fromAddress+"_"+toAddress)...)
}

// GetNullifierKey returns the key for a nullifier
func GetNullifierKey(nullifier []byte) []byte {
	return append(NullifierKeyPrefix, nullifier...)
}

// GetProviderKey returns the key for a verification provider
func GetProviderKey(providerID string) []byte {
	return append(ProviderKeyPrefix, []byte(providerID)...)
}

// GetAccreditationKey returns the key for an accreditation
func GetAccreditationKey(accreditationHash string) []byte {
	return append(AccreditationKeyPrefix, []byte(accreditationHash)...)
}

// GetVerificationRecordKey returns the key for a verification record
func GetVerificationRecordKey(address string) []byte {
	return append(VerificationRecordKeyPrefix, []byte(address)...)
}

// GetProofKey returns the key for a proof (anti-replay)
func GetProofKey(proofHash []byte) []byte {
	return append(ProofKeyPrefix, proofHash...)
}
