package types

const (
	// EventTypeOrderPlaced defines the event type for order placement
	EventTypeOrderPlaced = "anteil.order_placed"
	
	// EventTypeOrderMatched defines the event type for order matching
	EventTypeOrderMatched = "anteil.order_matched"
	
	// EventTypeAuctionStarted defines the event type for auction start
	EventTypeAuctionStarted = "anteil.auction_started"
	
	// EventTypePositionUpdated defines the event type for user position update
	EventTypePositionUpdated = "anteil.position_updated"
	
	// Attribute keys
	AttributeKeyOrderId      = "order_id"
	AttributeKeyOwner        = "owner"
	AttributeKeyOrderType    = "order_type"
	AttributeKeyAmount       = "amount"
	AttributeKeyPrice        = "price"
	AttributeKeyAntBalance   = "ant_balance"
	AttributeKeyLockedAnt    = "locked_ant"
	AttributeKeyAvailableAnt = "available_ant"
	AttributeKeyBlockHeight   = "block_height"
	AttributeKeyAuctionId     = "auction_id"
)

