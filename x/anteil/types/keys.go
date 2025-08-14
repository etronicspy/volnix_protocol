package types

const (
	// ModuleName defines the module name
	ModuleName = "anteil"

	// StoreKey is the KVStore key for the anteil module
	StoreKey = ModuleName

	// RouterKey is the msg router key for the anteil module (kept for compatibility)
	RouterKey = ModuleName

	// QuerierRoute is the querier route for the anteil module
	QuerierRoute = ModuleName
)

var (
	// OrderKeyPrefix defines the prefix for order keys
	OrderKeyPrefix = []byte{0x01}

	// TradeKeyPrefix defines the prefix for trade keys
	TradeKeyPrefix = []byte{0x02}

	// UserPositionKeyPrefix defines the prefix for user position keys
	UserPositionKeyPrefix = []byte{0x03}

	// AuctionKeyPrefix defines the prefix for auction keys
	AuctionKeyPrefix = []byte{0x04}
)

// GetOrderKey returns the key for an order
func GetOrderKey(orderID string) []byte {
	return append(OrderKeyPrefix, []byte(orderID)...)
}

// GetTradeKey returns the key for a trade
func GetTradeKey(tradeID string) []byte {
	return append(TradeKeyPrefix, []byte(tradeID)...)
}

// GetUserPositionKey returns the key for a user position
func GetUserPositionKey(owner string) []byte {
	return append(UserPositionKeyPrefix, []byte(owner)...)
}

// GetAuctionKey returns the key for an auction
func GetAuctionKey(auctionID string) []byte {
	return append(AuctionKeyPrefix, []byte(auctionID)...)
}
