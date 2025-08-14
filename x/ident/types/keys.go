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
)

// GetVerifiedAccountKey returns the key for a verified account
func GetVerifiedAccountKey(address string) []byte {
	return append(VerifiedAccountKeyPrefix, []byte(address)...)
}
