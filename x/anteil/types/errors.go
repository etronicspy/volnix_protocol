package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrEmptyOwner        = errors.Register(ModuleName, 1, "owner cannot be empty")
	ErrEmptyAntAmount    = errors.Register(ModuleName, 2, "ANT amount cannot be empty")
	ErrZeroAntAmount     = errors.Register(ModuleName, 41, "ANT amount must be greater than zero")
	ErrEmptyPrice        = errors.Register(ModuleName, 3, "price cannot be empty")
	ErrEmptyIdentityHash = errors.Register(ModuleName, 4, "identity hash cannot be empty")
	ErrEmptyBuyOrderID   = errors.Register(ModuleName, 5, "buy order ID cannot be empty")
	ErrEmptySellOrderID  = errors.Register(ModuleName, 6, "sell order ID cannot be empty")
	ErrEmptyBuyer        = errors.Register(ModuleName, 7, "buyer cannot be empty")
	ErrEmptySeller       = errors.Register(ModuleName, 8, "seller cannot be empty")
	ErrEmptyAntBalance   = errors.Register(ModuleName, 9, "ANT balance cannot be empty")

	ErrOrderNotFound      = errors.Register(ModuleName, 14, "order not found")
	ErrOrderAlreadyExists = errors.Register(ModuleName, 15, "order already exists")
	ErrInvalidOrderType   = errors.Register(ModuleName, 16, "invalid order type")
	ErrInvalidOrderSide   = errors.Register(ModuleName, 17, "invalid order side")
	ErrInvalidPrice       = errors.Register(ModuleName, 18, "invalid price")

	ErrInsufficientBalance = errors.Register(ModuleName, 19, "insufficient balance")
	ErrOrderExpired        = errors.Register(ModuleName, 20, "order has expired")
	ErrMaxOrdersExceeded   = errors.Register(ModuleName, 24, "maximum open orders exceeded")

	ErrTradeNotFound      = errors.Register(ModuleName, 25, "trade not found")
	ErrTradeAlreadyExists = errors.Register(ModuleName, 26, "trade already exists")

	ErrPositionNotFound = errors.Register(ModuleName, 31, "position not found")

	ErrOrderNotOpen = errors.Register(ModuleName, 40, "order is not open for updates")

	ErrAccountNotFound     = errors.Register(ModuleName, 42, "account not found or not verified")
	ErrInvalidRoleForOrder = errors.Register(ModuleName, 43, "only citizens and validators can create orders")
	ErrPriceMismatch       = errors.Register(ModuleName, 44, "buy price must be >= sell price for trade execution")
)
