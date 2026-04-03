package types

import (
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// getTimestamp returns timestamppb from blockTime if provided, otherwise timestamppb.Now()
func getTimestamp(blockTime ...time.Time) *timestamppb.Timestamp {
	if len(blockTime) > 0 {
		return timestamppb.New(blockTime[0])
	}
	return timestamppb.Now()
}

// NewOrder creates a new Order instance
func NewOrder(owner string, orderType anteilv1.OrderType, orderSide anteilv1.OrderSide, antAmount string, price string, identityHash string, blockTime ...time.Time) *anteilv1.Order {
	now := getTimestamp(blockTime...)
	expiresAt := timestamppb.New(now.AsTime().Add(24 * time.Hour))

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
func NewTrade(buyOrderID string, sellOrderID string, buyer string, seller string, antAmount string, price string, blockTime ...time.Time) *anteilv1.Trade {
	now := getTimestamp(blockTime...)

	return &anteilv1.Trade{
		TradeId:     generateTradeID(buyOrderID, sellOrderID, now.AsTime()),
		BuyOrderId:  buyOrderID,
		SellOrderId: sellOrderID,
		Buyer:       buyer,
		Seller:      seller,
		AntAmount:   antAmount,
		WrtAmount:   calculateWrtAmount(antAmount, price),
		Price:       price,
		ExecutedAt:  now,
		TradingFee:  "0",
	}
}

// NewUserPosition creates a new UserPosition instance
func NewUserPosition(owner string, antBalance string, blockTime ...time.Time) *anteilv1.UserPosition {
	now := getTimestamp(blockTime...)

	return &anteilv1.UserPosition{
		Owner:        owner,
		AntBalance:   antBalance,
		LockedAnt:    "0",
		AvailableAnt: antBalance,
		OpenOrderIds: []string{},
		TotalTrades:  "0",
		LastActivity: now,
	}
}

// NewEpochState creates a new EpochState instance
func NewEpochState(epochNumber uint64, start time.Time, end time.Time) *anteilv1.EpochState {
	return &anteilv1.EpochState{
		EpochNumber:           epochNumber,
		EpochStart:            timestamppb.New(start),
		EpochEnd:              timestamppb.New(end),
		TotalBurnedPrevEpoch:  "0",
		TotalBurnedCurrentEpoch: "0",
		EmissionCoefficient:   "1.0",
		TotalEmitted:          "0",
		ActiveSuppliers:       0,
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
	if order.AntAmount == "0" {
		return ErrZeroAntAmount
	}
	if order.Price == "" {
		return ErrEmptyPrice
	}
	if order.IdentityHash == "" {
		return ErrEmptyIdentityHash
	}
	if order.OrderType == anteilv1.OrderType_ORDER_TYPE_UNSPECIFIED {
		return ErrInvalidOrderType
	}
	if order.OrderSide == anteilv1.OrderSide_ORDER_SIDE_UNSPECIFIED {
		return ErrInvalidOrderSide
	}
	if price, err := strconv.ParseFloat(order.Price, 64); err != nil || price <= 0 {
		return ErrInvalidPrice
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

// NewOrderStore creates a new order store
func NewOrderStore(store storetypes.KVStore) storetypes.KVStore {
	return prefix.NewStore(store, OrderKeyPrefix)
}

// NewTradeStore creates a new trade store
func NewTradeStore(store storetypes.KVStore) storetypes.KVStore {
	return prefix.NewStore(store, TradeKeyPrefix)
}

// Helper functions

func generateOrderID(owner string, timestamp time.Time) string {
	return fmt.Sprintf("order_%s_%d", owner, timestamp.Unix())
}

func generateTradeID(buyOrderID, sellOrderID string, timestamp time.Time) string {
	return fmt.Sprintf("trade_%s_%s_%d", buyOrderID, sellOrderID, timestamp.Unix())
}

func calculateWrtAmount(antAmount, price string) string {
	antFloat, err := strconv.ParseFloat(antAmount, 64)
	if err != nil {
		return "0"
	}

	priceFloat, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return "0"
	}

	totalValue := antFloat * priceFloat
	return fmt.Sprintf("%.8f", totalValue)
}

// UpdateOrderStatus updates the status of an order
func UpdateOrderStatus(order *anteilv1.Order, status anteilv1.OrderStatus) {
	order.Status = status
}

// UpdateUserPosition updates user position based on trade
func UpdateUserPosition(position *anteilv1.UserPosition, trade *anteilv1.Trade, isBuyer bool, blockTime ...time.Time) {
	currentTrades, _ := strconv.ParseInt(position.TotalTrades, 10, 64)
	position.TotalTrades = fmt.Sprintf("%d", currentTrades+1)

	position.LastActivity = getTimestamp(blockTime...)
}

// ParseUint64 safely parses a string to uint64, returning 0 on error
func ParseUint64(s string) uint64 {
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}
