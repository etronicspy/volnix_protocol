package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

func TestNewOrder(t *testing.T) {
	order := types.NewOrder(
		"cosmos1owner",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"1.5",
		"hash123",
	)

	require.NotNil(t, order)
	require.Equal(t, "cosmos1owner", order.Owner)
	require.Equal(t, anteilv1.OrderType_ORDER_TYPE_LIMIT, order.OrderType)
	require.Equal(t, anteilv1.OrderSide_ORDER_SIDE_BUY, order.OrderSide)
	require.Equal(t, "1000000", order.AntAmount)
	require.Equal(t, "1.5", order.Price)
	require.Equal(t, "hash123", order.IdentityHash)
	require.Equal(t, anteilv1.OrderStatus_ORDER_STATUS_OPEN, order.Status)
	require.NotEmpty(t, order.OrderId)
	require.NotNil(t, order.CreatedAt)
	require.NotNil(t, order.ExpiresAt)
}

func TestNewTrade(t *testing.T) {
	trade := types.NewTrade(
		"buy1",
		"sell1",
		"cosmos1buyer",
		"cosmos1seller",
		"1000000",
		"1.5",
		"hash123",
	)

	require.NotNil(t, trade)
	require.Equal(t, "buy1", trade.BuyOrderId)
	require.Equal(t, "sell1", trade.SellOrderId)
	require.Equal(t, "cosmos1buyer", trade.Buyer)
	require.Equal(t, "cosmos1seller", trade.Seller)
	require.Equal(t, "1000000", trade.AntAmount)
	require.Equal(t, "1.5", trade.Price)
	require.Equal(t, "hash123", trade.IdentityHash)
	require.NotEmpty(t, trade.TradeId)
	require.NotEmpty(t, trade.TotalValue)
	require.NotNil(t, trade.ExecutedAt)
}

func TestIsOrderValid(t *testing.T) {
	tests := []struct {
		name    string
		order   *anteilv1.Order
		wantErr error
	}{
		{
			name: "valid order",
			order: types.NewOrder(
				"cosmos1owner",
				anteilv1.OrderType_ORDER_TYPE_LIMIT,
				anteilv1.OrderSide_ORDER_SIDE_BUY,
				"1000000",
				"1.5",
				"hash123",
			),
			wantErr: nil,
		},
		{
			name: "empty owner",
			order: &anteilv1.Order{
				OrderId:      "order1",
				Owner:        "",
				OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
				OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
				AntAmount:    "1000000",
				Price:        "1.5",
				IdentityHash: "hash123",
			},
			wantErr: types.ErrEmptyOwner,
		},
		{
			name: "empty ant amount",
			order: &anteilv1.Order{
				OrderId:      "order1",
				Owner:        "cosmos1owner",
				OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
				OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
				AntAmount:    "",
				Price:        "1.5",
				IdentityHash: "hash123",
			},
			wantErr: types.ErrEmptyAntAmount,
		},
		{
			name: "empty price",
			order: &anteilv1.Order{
				OrderId:      "order1",
				Owner:        "cosmos1owner",
				OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
				OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
				AntAmount:    "1000000",
				Price:        "",
				IdentityHash: "hash123",
			},
			wantErr: types.ErrEmptyPrice,
		},
		{
			name: "empty identity hash",
			order: &anteilv1.Order{
				OrderId:      "order1",
				Owner:        "cosmos1owner",
				OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
				OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
				AntAmount:    "1000000",
				Price:        "1.5",
				IdentityHash: "",
			},
			wantErr: types.ErrEmptyIdentityHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.IsOrderValid(tt.order)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsTradeValid(t *testing.T) {
	tests := []struct {
		name    string
		trade   *anteilv1.Trade
		wantErr error
	}{
		{
			name: "valid trade",
			trade: types.NewTrade(
				"buy1",
				"sell1",
				"cosmos1buyer",
				"cosmos1seller",
				"1000000",
				"1.5",
				"hash123",
			),
			wantErr: nil,
		},
		{
			name: "empty buy order id",
			trade: &anteilv1.Trade{
				TradeId:     "trade1",
				BuyOrderId:  "",
				SellOrderId: "sell1",
				Buyer:       "cosmos1buyer",
				Seller:      "cosmos1seller",
				AntAmount:   "1000000",
				Price:       "1.5",
			},
			wantErr: types.ErrEmptyBuyOrderID,
		},
		{
			name: "empty sell order id",
			trade: &anteilv1.Trade{
				TradeId:     "trade1",
				BuyOrderId:  "buy1",
				SellOrderId: "",
				Buyer:       "cosmos1buyer",
				Seller:      "cosmos1seller",
				AntAmount:   "1000000",
				Price:       "1.5",
			},
			wantErr: types.ErrEmptySellOrderID,
		},
		{
			name: "empty buyer",
			trade: &anteilv1.Trade{
				TradeId:     "trade1",
				BuyOrderId:  "buy1",
				SellOrderId: "sell1",
				Buyer:       "",
				Seller:      "cosmos1seller",
				AntAmount:   "1000000",
				Price:       "1.5",
			},
			wantErr: types.ErrEmptyBuyer,
		},
		{
			name: "empty seller",
			trade: &anteilv1.Trade{
				TradeId:     "trade1",
				BuyOrderId:  "buy1",
				SellOrderId: "sell1",
				Buyer:       "cosmos1buyer",
				Seller:      "",
				AntAmount:   "1000000",
				Price:       "1.5",
			},
			wantErr: types.ErrEmptySeller,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.IsTradeValid(tt.trade)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsAuctionValid(t *testing.T) {
	tests := []struct {
		name    string
		auction *anteilv1.Auction
		wantErr error
	}{
		{
			name: "valid auction",
			auction: &anteilv1.Auction{
				AuctionId:    "auction1",
				ReservePrice: "1000000",
				AntAmount:    "1000000",
			},
			wantErr: nil,
		},
		{
			name: "empty auction id",
			auction: &anteilv1.Auction{
				AuctionId:    "",
				ReservePrice: "1000000",
				AntAmount:    "1000000",
			},
			wantErr: types.ErrEmptyAuctionID,
		},
		{
			name: "empty reserve price",
			auction: &anteilv1.Auction{
				AuctionId:    "auction1",
				ReservePrice: "",
				AntAmount:    "1000000",
			},
			wantErr: types.ErrEmptyReservePrice,
		},
		{
			name: "empty ant amount",
			auction: &anteilv1.Auction{
				AuctionId:    "auction1",
				ReservePrice: "1000000",
				AntAmount:    "",
			},
			wantErr: types.ErrEmptyAntAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.IsAuctionValid(tt.auction)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsBidValid(t *testing.T) {
	tests := []struct {
		name    string
		bid     *anteilv1.Bid
		wantErr error
	}{
		{
			name: "valid bid",
			bid: &anteilv1.Bid{
				BidId:  "bid1",
				Bidder: "cosmos1bidder",
				Amount: "1500000",
			},
			wantErr: nil,
		},
		{
			name: "empty bidder",
			bid: &anteilv1.Bid{
				BidId:  "bid1",
				Bidder: "",
				Amount: "1500000",
			},
			wantErr: types.ErrEmptyBidder,
		},
		{
			name: "empty amount",
			bid: &anteilv1.Bid{
				BidId:  "bid1",
				Bidder: "cosmos1bidder",
				Amount: "",
			},
			wantErr: types.ErrEmptyBidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.IsBidValid(tt.bid)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
