package types

import (
	"cosmossdk.io/errors"
)

var (
	// ErrEmptyOwner indicates that the owner field is empty
	ErrEmptyOwner = errors.Register(ModuleName, 1, "owner cannot be empty")

	// ErrEmptyAntAmount indicates that the ANT amount field is empty
	ErrEmptyAntAmount = errors.Register(ModuleName, 2, "ANT amount cannot be empty")

	// ErrEmptyPrice indicates that the price field is empty
	ErrEmptyPrice = errors.Register(ModuleName, 3, "price cannot be empty")

	// ErrEmptyIdentityHash indicates that the identity hash field is empty
	ErrEmptyIdentityHash = errors.Register(ModuleName, 4, "identity hash cannot be empty")

	// ErrEmptyBuyOrderID indicates that the buy order ID field is empty
	ErrEmptyBuyOrderID = errors.Register(ModuleName, 5, "buy order ID cannot be empty")

	// ErrEmptySellOrderID indicates that the sell order ID field is empty
	ErrEmptySellOrderID = errors.Register(ModuleName, 6, "sell order ID cannot be empty")

	// ErrEmptyBuyer indicates that the buyer field is empty
	ErrEmptyBuyer = errors.Register(ModuleName, 7, "buyer cannot be empty")

	// ErrEmptySeller indicates that the seller field is empty
	ErrEmptySeller = errors.Register(ModuleName, 8, "seller cannot be empty")

	// ErrEmptyAntBalance indicates that the ANT balance field is empty
	ErrEmptyAntBalance = errors.Register(ModuleName, 9, "ANT balance cannot be empty")

	// ErrEmptyAuctionID indicates that the auction ID field is empty
	ErrEmptyAuctionID = errors.Register(ModuleName, 10, "auction ID cannot be empty")

	// ErrEmptyReservePrice indicates that the reserve price field is empty
	ErrEmptyReservePrice = errors.Register(ModuleName, 11, "reserve price cannot be empty")

	// ErrEmptyBidder indicates that the bidder field is empty
	ErrEmptyBidder = errors.Register(ModuleName, 12, "bidder cannot be empty")

	// ErrEmptyBidAmount indicates that the bid amount field is empty
	ErrEmptyBidAmount = errors.Register(ModuleName, 13, "bid amount cannot be empty")

	// ErrOrderNotFound indicates that the order was not found
	ErrOrderNotFound = errors.Register(ModuleName, 14, "order not found")

	// ErrOrderAlreadyExists indicates that the order already exists
	ErrOrderAlreadyExists = errors.Register(ModuleName, 15, "order already exists")

	// ErrInvalidOrderType indicates that the order type is invalid
	ErrInvalidOrderType = errors.Register(ModuleName, 16, "invalid order type")

	// ErrInvalidOrderSide indicates that the order side is invalid
	ErrInvalidOrderSide = errors.Register(ModuleName, 17, "invalid order side")

	// ErrInvalidPrice indicates that the price is invalid
	ErrInvalidPrice = errors.Register(ModuleName, 18, "invalid price")

	// ErrInsufficientBalance indicates that the user has insufficient balance
	ErrInsufficientBalance = errors.Register(ModuleName, 19, "insufficient balance")

	// ErrOrderExpired indicates that the order has expired
	ErrOrderExpired = errors.Register(ModuleName, 20, "order has expired")

	// ErrAuctionNotFound indicates that the auction was not found
	ErrAuctionNotFound = errors.Register(ModuleName, 21, "auction not found")

	// ErrAuctionClosed indicates that the auction is closed
	ErrAuctionClosed = errors.Register(ModuleName, 22, "auction is closed")

	// ErrBidTooLow indicates that the bid is below the reserve price
	ErrBidTooLow = errors.Register(ModuleName, 23, "bid is below reserve price")

	// ErrMaxOrdersExceeded indicates that the user has exceeded maximum open orders
	ErrMaxOrdersExceeded = errors.Register(ModuleName, 24, "maximum open orders exceeded")

	// Trade errors
	ErrTradeNotFound      = errors.Register(ModuleName, 25, "trade not found")
	ErrTradeAlreadyExists = errors.Register(ModuleName, 26, "trade already exists")

	// Auction errors
	ErrAuctionAlreadyExists = errors.Register(ModuleName, 27, "auction already exists")
	ErrAuctionNotClosed     = errors.Register(ModuleName, 28, "auction is not closed")
	ErrAuctionExpired       = errors.Register(ModuleName, 29, "auction has expired")
	ErrNoWinningBid         = errors.Register(ModuleName, 30, "no winning bid found")

	// Position errors
	ErrPositionNotFound = errors.Register(ModuleName, 31, "position not found")

	// Bid errors
	ErrBidNotFound = errors.Register(ModuleName, 32, "bid not found")
)
