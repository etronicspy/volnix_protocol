package types

const (
	// §6.3 — lizenz module events
	EventTypeLizenzActivated   = "lizenz.lizenz_activated"
	EventTypeLizenzDeactivated = "lizenz.lizenz_deactivated"
	EventTypeLZNLocked         = "lizenz.lzn_locked"
	EventTypeLZNUnlocked       = "lizenz.lzn_unlocked"
	EventTypeMOAChecked        = "lizenz.moa_checked"
	EventTypeValidatorRegistered = "lizenz.validator_registered"

	// §6.3 mandatory attributes
	AttributeKeyValidator        = "validator"
	AttributeKeyAccount          = "account"
	AttributeKeyBlockHeight      = "block_height"
	AttributeKeyTxHash           = "tx_hash"
	AttributeKeyAmount           = "amount"
	AttributeKeyActivationTime   = "activation_time"
	AttributeKeyDeactivationTime = "deactivation_time"
	AttributeKeyFreezeUntil      = "freeze_until"  // §4.1 freeze period
	AttributeKeyReason           = "reason"
	AttributeKeyMOAStatus        = "moa_status"
	AttributeKeyMOACompliance    = "moa_compliance"
	AttributeKeyLZNBalance       = "lzn_balance"
	AttributeKeyValidatorWeight  = "validator_weight"
	AttributeKeyPower            = "power"

	// §6.3 / Appendix A — consensus_change_reason values (emitted by lizenz on activation/deactivation)
	AttributeKeyConsensusChangeReason = "consensus_change_reason"
	ConsensusChangeReasonLZNActivated   = "lzn_activated"
	ConsensusChangeReasonLZNDeactivated = "lzn_deactivated"
)
