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
	OrderKeyPrefix        = []byte{0x01}
	TradeKeyPrefix        = []byte{0x02}
	UserPositionKeyPrefix = []byte{0x03}
	EpochStateKey         = []byte{0x04}

	LastDistributionTimeKey = []byte{0x06}
)

func GetOrderKey(orderID string) []byte {
	return append(OrderKeyPrefix, []byte(orderID)...)
}

func GetTradeKey(tradeID string) []byte {
	return append(TradeKeyPrefix, []byte(tradeID)...)
}

func GetUserPositionKey(owner string) []byte {
	return append(UserPositionKeyPrefix, []byte(owner)...)
}

func GetOrderPrefix() []byte {
	return OrderKeyPrefix
}

func GetTradePrefix() []byte {
	return TradeKeyPrefix
}
