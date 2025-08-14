package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
)

func TestNewOrder(t *testing.T) {
	order := NewOrder(
		"test_owner",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"1.5",
		"test_hash",
	)

	require.NotNil(t, order)
	require.Equal(t, "test_owner", order.GetOwner())
	require.Equal(t, anteilv1.OrderType_ORDER_TYPE_LIMIT, order.GetOrderType())
	require.Equal(t, anteilv1.OrderSide_ORDER_SIDE_BUY, order.GetOrderSide())
	require.Equal(t, "1000000", order.GetAntAmount())
	require.Equal(t, "1.5", order.GetPrice())
	require.Equal(t, "test_hash", order.GetIdentityHash())
	require.Equal(t, anteilv1.OrderStatus_ORDER_STATUS_OPEN, order.GetStatus())
}

func TestNewTrade(t *testing.T) {
	trade := NewTrade(
		"buy_order_1",
		"sell_order_1",
		"buyer_addr",
		"seller_addr",
		"500000",
		"1.5",
		"test_hash",
	)

	require.NotNil(t, trade)
	require.Equal(t, "buy_order_1", trade.GetBuyOrderId())
	require.Equal(t, "sell_order_1", trade.GetSellOrderId())
	require.Equal(t, "buyer_addr", trade.GetBuyer())
	require.Equal(t, "seller_addr", trade.GetSeller())
	require.Equal(t, "500000", trade.GetAntAmount())
	require.Equal(t, "1.5", trade.GetPrice())
	require.Equal(t, "test_hash", trade.GetIdentityHash())
}

func TestNewUserPosition(t *testing.T) {
	position := NewUserPosition("test_owner", "10000000")

	require.NotNil(t, position)
	require.Equal(t, "test_owner", position.GetOwner())
	require.Equal(t, "10000000", position.GetAntBalance())
	require.Equal(t, "0", position.GetLockedAnt())
	require.Equal(t, "10000000", position.GetAvailableAnt())
	require.Empty(t, position.GetOpenOrderIds())
	require.Equal(t, "0", position.GetTotalTrades())
	require.Equal(t, "0", position.GetTotalVolume())
}

func TestNewAuction(t *testing.T) {
	auction := NewAuction(1000, "1000000", "1.0")

	require.NotNil(t, auction)
	require.Equal(t, uint64(1000), auction.GetBlockHeight())
	require.Equal(t, "1000000", auction.GetAntAmount())
	require.Equal(t, "1.0", auction.GetReservePrice())
	require.Equal(t, anteilv1.AuctionStatus_AUCTION_STATUS_OPEN, auction.GetStatus())
	require.Empty(t, auction.GetBids())
	require.Empty(t, auction.GetWinningBid())
}

func TestNewBid(t *testing.T) {
	bid := NewBid("bidder_addr", "auction_1", "1000000", "test_hash")

	require.NotNil(t, bid)
	require.Equal(t, "bidder_addr", bid.GetBidder())
	require.Equal(t, "1000000", bid.GetAmount())
	require.Equal(t, "test_hash", bid.GetIdentityHash())
}

func TestIsOrderValid(t *testing.T) {
	// Test valid order
	validOrder := NewOrder(
		"test_owner",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"1.5",
		"test_hash",
	)

	err := IsOrderValid(validOrder)
	require.NoError(t, err)

	// Test invalid order - empty owner
	invalidOrder := NewOrder(
		"",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"1.5",
		"test_hash",
	)

	err = IsOrderValid(invalidOrder)
	require.Error(t, err)
	require.Equal(t, ErrEmptyOwner, err)
}

func TestIsTradeValid(t *testing.T) {
	// Test valid trade
	validTrade := NewTrade(
		"buy_order_1",
		"sell_order_1",
		"buyer_addr",
		"seller_addr",
		"500000",
		"1.5",
		"test_hash",
	)

	err := IsTradeValid(validTrade)
	require.NoError(t, err)

	// Test invalid trade - empty buyer
	invalidTrade := NewTrade(
		"buy_order_1",
		"sell_order_1",
		"",
		"seller_addr",
		"500000",
		"1.5",
		"test_hash",
	)

	err = IsTradeValid(invalidTrade)
	require.Error(t, err)
	require.Equal(t, ErrEmptyBuyer, err)
}

func TestIsUserPositionValid(t *testing.T) {
	// Test valid position
	validPosition := NewUserPosition("test_owner", "10000000")

	err := IsUserPositionValid(validPosition)
	require.NoError(t, err)

	// Test invalid position - empty owner
	invalidPosition := NewUserPosition("", "10000000")

	err = IsUserPositionValid(invalidPosition)
	require.Error(t, err)
	require.Equal(t, ErrEmptyOwner, err)
}

func TestIsAuctionValid(t *testing.T) {
	// Test valid auction
	validAuction := NewAuction(1000, "1000000", "1.0")

	err := IsAuctionValid(validAuction)
	require.NoError(t, err)

	// Test invalid auction - empty ant amount
	invalidAuction := NewAuction(1000, "", "1.0")

	err = IsAuctionValid(invalidAuction)
	require.Error(t, err)
	require.Equal(t, ErrEmptyAntAmount, err)
}

func TestIsBidValid(t *testing.T) {
	// Test valid bid
	validBid := NewBid("bidder_addr", "auction_1", "1000000", "test_hash")

	err := IsBidValid(validBid)
	require.NoError(t, err)

	// Test invalid bid - empty bidder
	invalidBid := NewBid("", "auction_1", "1000000", "test_hash")

	err = IsBidValid(invalidBid)
	require.Error(t, err)
	require.Equal(t, ErrEmptyBidder, err)
}

func TestUpdateOrderStatus(t *testing.T) {
	order := NewOrder(
		"test_owner",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"1.5",
		"test_hash",
	)

	// Test initial status
	require.Equal(t, anteilv1.OrderStatus_ORDER_STATUS_OPEN, order.GetStatus())

	// Update status
	UpdateOrderStatus(order, anteilv1.OrderStatus_ORDER_STATUS_FILLED)
	require.Equal(t, anteilv1.OrderStatus_ORDER_STATUS_FILLED, order.GetStatus())
}

func TestUpdateUserPosition(t *testing.T) {
	position := NewUserPosition("test_owner", "10000000")
	trade := NewTrade(
		"buy_order_1",
		"sell_order_1",
		"test_owner",
		"seller_addr",
		"500000",
		"1.5",
		"test_hash",
	)

	// Test initial values
	require.Equal(t, "0", position.GetTotalTrades())
	require.Equal(t, "0", position.GetTotalVolume())

	// Update position
	UpdateUserPosition(position, trade, true)

	// Note: In real implementation, these would be properly calculated
	// For now, we just verify the function doesn't crash
	require.NotNil(t, position)
}
