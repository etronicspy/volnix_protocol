package types

const (
	ModuleName   = "lizenz"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

var (
	// ActivatedLizenzKeyPrefix defines the prefix for activated LZN keys
	ActivatedLizenzKeyPrefix = []byte{0x01}
	
	// DeactivatingLizenzKeyPrefix defines the prefix for deactivating LZN keys
	DeactivatingLizenzKeyPrefix = []byte{0x02}
	
	// MOAStatusKeyPrefix defines the prefix for MOA status keys
	MOAStatusKeyPrefix = []byte{0x03}
)

// GetActivatedLizenzKey returns the key for an activated LZN
func GetActivatedLizenzKey(validator string) []byte {
	return append(ActivatedLizenzKeyPrefix, []byte(validator)...)
}

// GetDeactivatingLizenzKey returns the key for a deactivating LZN
func GetDeactivatingLizenzKey(validator string) []byte {
	return append(DeactivatingLizenzKeyPrefix, []byte(validator)...)
}

// GetMOAStatusKey returns the key for a MOA status
func GetMOAStatusKey(validator string) []byte {
	return append(MOAStatusKeyPrefix, []byte(validator)...)
}
