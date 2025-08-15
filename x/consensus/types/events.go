package types

const (
	// EventTypeBlockCreatorSelected defines the event type for block creator selection
	EventTypeBlockCreatorSelected = "block_creator_selected"
	
	// EventTypeValidatorPowerUpdated defines the event type for validator power updates
	EventTypeValidatorPowerUpdated = "validator_power_updated"
	
	// EventTypeBlockTimeAdjusted defines the event type for block time adjustments
	EventTypeBlockTimeAdjusted = "block_time_adjusted"
	
	// Attribute keys
	AttributeKeyBlockCreator = "block_creator"
	AttributeKeyBlockHeight  = "block_height"
	AttributeKeyValidator    = "validator"
	AttributeKeyPower        = "power"
	AttributeKeyBlockTime    = "block_time"
)
