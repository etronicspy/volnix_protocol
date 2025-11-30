package types

const (
	// EventTypeLizenzActivated defines the event type for LZN license activation
	EventTypeLizenzActivated = "lizenz.lizenz_activated"
	
	// EventTypeLizenzDeactivated defines the event type for LZN license deactivation
	EventTypeLizenzDeactivated = "lizenz.lizenz_deactivated"
	
	// EventTypeMOAChecked defines the event type for MOA compliance check
	EventTypeMOAChecked = "lizenz.moa_checked"
	
	// EventTypeLZNLocked defines the event type for LZN token locking
	EventTypeLZNLocked = "lizenz.lzn_locked"
	
	// EventTypeLZNUnlocked defines the event type for LZN token unlocking
	EventTypeLZNUnlocked = "lizenz.lzn_unlocked"
	
	// EventTypeValidatorRegistered defines the event type for automatic validator registration
	EventTypeValidatorRegistered = "lizenz.validator_registered"
	
	// Attribute keys
	AttributeKeyValidator      = "validator"
	AttributeKeyAmount         = "amount"
	AttributeKeyActivationTime  = "activation_time"
	AttributeKeyDeactivationTime = "deactivation_time"
	AttributeKeyReason          = "reason"
	AttributeKeyMOAStatus       = "moa_status"
	AttributeKeyMOACompliance   = "moa_compliance"
	AttributeKeyLZNBalance      = "lzn_balance"
	AttributeKeyBlockHeight     = "block_height"
	AttributeKeyValidatorWeight  = "validator_weight"
)

