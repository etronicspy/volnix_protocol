package types

import (
	"fmt"
	"time"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewOrder creates a new Order instance
func NewOrder(owner string, orderType anteilv1.OrderType, orderSide anteilv1.OrderSide, antAmount string, price string, identityHash string) *anteilv1.Order {
	now := timestamppb.Now()
	expiresAt := timestamppb.New(now.AsTime().Add(24 * time.Hour)) // Default 24h expiry

	return &anteilv1.Order{
		OrderId:      generateOrderID(owner, now.AsTime()),
		Owner:        owner,
		OrderType:    orderType,
		OrderSide:    orderSide,
		AntAmount:    antAmount,
		Price:        price,
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    now,
		ExpiresAt:    expiresAt,
		IdentityHash: identityHash,
	}
}

// NewTrade creates a new Trade instance
func NewTrade(buyOrderID string, sellOrderID string, buyer string, seller string, antAmount string, price string, identityHash string) *anteilv1.Trade {
	now := timestamppb.Now()

	return &anteilv1.Trade{
		TradeId:      generateTradeID(buyOrderID, sellOrderID, now.AsTime()),
		BuyOrderId:   buyOrderID,
		SellOrderId:  sellOrderID,
		Buyer:        buyer,
		Seller:       seller,
		AntAmount:    antAmount,
		Price:        price,
		TotalValue:   calculateTotalValue(antAmount, price),
		ExecutedAt:   now,
		IdentityHash: identityHash,
	}
}

// NewUserPosition creates a new UserPosition instance
func NewUserPosition(owner string, antBalance string) *anteilv1.UserPosition {
	now := timestamppb.Now()

	return &anteilv1.UserPosition{
		Owner:        owner,
		AntBalance:   antBalance,
		LockedAnt:    "0",
		AvailableAnt: antBalance,
		OpenOrderIds: []string{},
		TotalTrades:  "0",
		TotalVolume:  "0",
		LastActivity: now,
	}
}

// NewAuction creates a new Auction instance
func NewAuction(blockHeight uint64, antAmount string, reservePrice string) *anteilv1.Auction {
	now := timestamppb.Now()
	endTime := timestamppb.New(now.AsTime().Add(1 * time.Hour)) // Default 1h auction duration

	return &anteilv1.Auction{
		AuctionId:    generateAuctionID(blockHeight, now.AsTime()),
		BlockHeight:  blockHeight,
		StartTime:    now,
		EndTime:      endTime,
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		Bids:         []*anteilv1.Bid{},
		WinningBid:   "",
		ReservePrice: reservePrice,
		AntAmount:    antAmount,
	}
}

// NewBid creates a new Bid instance
func NewBid(bidder string, auctionID string, amount string, identityHash string) *anteilv1.Bid {
	now := timestamppb.Now()

	return &anteilv1.Bid{
		BidId:        generateBidID(bidder, auctionID, now.AsTime()),
		Bidder:       bidder,
		Amount:       amount,
		SubmittedAt:  now,
		IdentityHash: identityHash,
	}
}

// IsOrderValid checks if the order is valid
func IsOrderValid(order *anteilv1.Order) error {
	if order.Owner == "" {
		return ErrEmptyOwner
	}
	if order.AntAmount == "" {
		return ErrEmptyAntAmount
	}
	if order.Price == "" {
		return ErrEmptyPrice
	}
	if order.IdentityHash == "" {
		return ErrEmptyIdentityHash
	}
	return nil
}

// IsTradeValid checks if the trade is valid
func IsTradeValid(trade *anteilv1.Trade) error {
	if trade.BuyOrderId == "" {
		return ErrEmptyBuyOrderID
	}
	if trade.SellOrderId == "" {
		return ErrEmptySellOrderID
	}
	if trade.Buyer == "" {
		return ErrEmptyBuyer
	}
	if trade.Seller == "" {
		return ErrEmptySeller
	}
	if trade.AntAmount == "" {
		return ErrEmptyAntAmount
	}
	if trade.Price == "" {
		return ErrEmptyPrice
	}
	return nil
}

// IsUserPositionValid checks if the user position is valid
func IsUserPositionValid(position *anteilv1.UserPosition) error {
	if position.Owner == "" {
		return ErrEmptyOwner
	}
	if position.AntBalance == "" {
		return ErrEmptyAntBalance
	}
	return nil
}

// IsAuctionValid checks if the auction is valid
func IsAuctionValid(auction *anteilv1.Auction) error {
	if auction.AuctionId == "" {
		return ErrEmptyAuctionID
	}
	if auction.AntAmount == "" {
		return ErrEmptyAntAmount
	}
	if auction.ReservePrice == "" {
		return ErrEmptyReservePrice
	}
	return nil
}

// IsBidValid checks if the bid is valid
func IsBidValid(bid *anteilv1.Bid) error {
	if bid.Bidder == "" {
		return ErrEmptyBidder
	}
	if bid.Amount == "" {
		return ErrEmptyBidAmount
	}
	if bid.IdentityHash == "" {
		return ErrEmptyIdentityHash
	}
	return nil
}

// UpdateOrderStatus updates the order status
func UpdateOrderStatus(order *anteilv1.Order, newStatus anteilv1.OrderStatus) {
	order.Status = newStatus
}

// UpdateUserPosition updates the user position with new trade
func UpdateUserPosition(position *anteilv1.UserPosition, trade *anteilv1.Trade, isBuyer bool) {
	// Update total trades
	currentTrades := parseUint64(position.TotalTrades)
	position.TotalTrades = formatUint64(currentTrades + 1)

	// Update total volume
	currentVolume := parseUint64(position.TotalVolume)
	tradeVolume := parseUint64(trade.TotalValue)
	position.TotalVolume = formatUint64(currentVolume + tradeVolume)

	// Update last activity
	position.LastActivity = timestamppb.Now()
}

// Helper functions for ID generation
func generateOrderID(owner string, timestamp time.Time) string {
	return fmt.Sprintf("order_%s_%d", owner, timestamp.Unix())
}

func generateTradeID(buyOrderID string, sellOrderID string, timestamp time.Time) string {
	return fmt.Sprintf("trade_%s_%s_%d", buyOrderID, sellOrderID, timestamp.Unix())
}

func generateAuctionID(blockHeight uint64, timestamp time.Time) string {
	return fmt.Sprintf("auction_%d_%d", blockHeight, timestamp.Unix())
}

func generateBidID(bidder string, auctionID string, timestamp time.Time) string {
	return fmt.Sprintf("bid_%s_%s_%d", bidder, auctionID, timestamp.Unix())
}

// Helper functions for calculations
func calculateTotalValue(antAmount string, price string) string {
	// In real implementation, this would use decimal arithmetic
	// For now, return a placeholder
	return antAmount
}

// Helper functions for parsing and formatting
func parseUint64(s string) uint64 {
	// In real implementation, this would parse the string to uint64
	// For now, return 0
	return 0
}

func formatUint64(u uint64) string {
	return fmt.Sprintf("%d", u)
}
