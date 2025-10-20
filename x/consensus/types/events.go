package types

const (
	// EventTypeBlockCreatorSelected defines the event type for block creator selection
	EventTypeBlockCreatorSelected = "block_creator_selected"
	
	// EventTypeValidatorPowerUpdated defines the event type for validator power updates
	EventTypeValidatorPowerUpdated = "validator_power_updated"
	
	// EventTypeBlockTimeAdjusted defines the event type for block time adjustments
	EventTypeBlockTimeAdjusted = "block_time_adjusted"
	
	// EventTypeHalving defines the event type for halving events
	EventTypeHalving = "halving"
	
	// EventTypeConsensusStateUpdated defines the event type for consensus state updates
	EventTypeConsensusStateUpdated = "consensus_state_updated"
	
	// Attribute keys
	AttributeKeyBlockCreator = "block_creator"
	AttributeKeyBlockHeight  = "block_height"
	AttributeKeyValidator    = "validator"
	AttributeKeyPower        = "power"
	AttributeKeyBlockTime    = "block_time"
	AttributeKeyHeight       = "height"
	AttributeKeyNextHalving  = "next_halving"
)
