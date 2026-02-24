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

	// BidKeyPrefix defines the prefix for bid keys
	BidKeyPrefix = []byte{0x05}
	
	// LastDistributionTimeKey defines the key for storing last ANT distribution time
	LastDistributionTimeKey = []byte{0x06}

	// MarketMakerKeyPrefix defines the prefix for market maker keys
	MarketMakerKeyPrefix = []byte{0x07}

	// StakingPositionKeyPrefix defines the prefix for staking position keys
	StakingPositionKeyPrefix = []byte{0x08}

	// LiquidityPoolKeyPrefix defines the prefix for liquidity pool keys
	LiquidityPoolKeyPrefix = []byte{0x09}

	// LiquidityShareKeyPrefix defines the prefix for liquidity provider share keys
	LiquidityShareKeyPrefix = []byte{0x0A}
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

// GetBidKey returns the key for a bid
func GetBidKey(auctionID, bidID string) []byte {
	return append(BidKeyPrefix, []byte(auctionID+"_"+bidID)...)
}

// GetOrderPrefix returns the order prefix
func GetOrderPrefix() []byte {
	return OrderKeyPrefix
}

// GetTradePrefix returns the trade prefix
func GetTradePrefix() []byte {
	return TradeKeyPrefix
}

// GetAuctionPrefix returns the auction prefix
func GetAuctionPrefix() []byte {
	return AuctionKeyPrefix
}

// GetBidPrefix returns the bid prefix
func GetBidPrefix() []byte {
	return BidKeyPrefix
}

// GetMarketMakerKey returns the key for a market maker
func GetMarketMakerKey(makerID string) []byte {
	return append(MarketMakerKeyPrefix, []byte(makerID)...)
}

// GetStakingPositionKey returns the key for a staking position
func GetStakingPositionKey(owner string) []byte {
	return append(StakingPositionKeyPrefix, []byte(owner)...)
}

// GetLiquidityPoolKey returns the key for a liquidity pool
func GetLiquidityPoolKey(poolID string) []byte {
	return append(LiquidityPoolKeyPrefix, []byte(poolID)...)
}

// GetLiquidityShareKey returns the key for a liquidity provider share
func GetLiquidityShareKey(poolID, owner string) []byte {
	return append(LiquidityShareKeyPrefix, []byte(poolID+"_"+owner)...)
}
