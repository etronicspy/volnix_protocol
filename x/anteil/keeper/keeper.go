package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		paramstore paramtypes.Subspace
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ps paramtypes.Subspace,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(anteiltypes.ParamKeyTable())
	}

	return &Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		paramstore: ps,
	}
}

// GetParams returns the current parameters for the anteil module
func (k Keeper) GetParams(ctx sdk.Context) anteiltypes.Params {
	var params anteiltypes.Params
	k.paramstore.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the parameters for the anteil module
func (k Keeper) SetParams(ctx sdk.Context, params anteiltypes.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// Order Management Methods

// SetOrder stores an order in the store
func (k Keeper) SetOrder(ctx sdk.Context, order *anteilv1.Order) error {
	if err := anteiltypes.IsOrderValid(order); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	orderKey := anteiltypes.GetOrderKey(order.GetOrderId())

	// Check if order already exists
	if store.Has(orderKey) {
		return anteiltypes.ErrOrderAlreadyExists
	}

	// Store the order
	orderBz, err := k.cdc.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	store.Set(orderKey, orderBz)
	return nil
}

// CreateOrder creates a new order (alias for SetOrder)
func (k Keeper) CreateOrder(ctx sdk.Context, order *anteilv1.Order) error {
	return k.SetOrder(ctx, order)
}

// GetOrder retrieves an order by ID
func (k Keeper) GetOrder(ctx sdk.Context, orderID string) (*anteilv1.Order, error) {
	store := ctx.KVStore(k.storeKey)
	orderKey := anteiltypes.GetOrderKey(orderID)

	if !store.Has(orderKey) {
		return nil, anteiltypes.ErrOrderNotFound
	}

	orderBz := store.Get(orderKey)
	var order anteilv1.Order
	if err := k.cdc.Unmarshal(orderBz, &order); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order: %w", err)
	}

	return &order, nil
}

// UpdateOrder updates an existing order
func (k Keeper) UpdateOrder(ctx sdk.Context, order *anteilv1.Order) error {
	if err := anteiltypes.IsOrderValid(order); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	orderKey := anteiltypes.GetOrderKey(order.GetOrderId())

	// Check if order exists
	if !store.Has(orderKey) {
		return anteiltypes.ErrOrderNotFound
	}

	// Store the updated order
	orderBz, err := k.cdc.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	store.Set(orderKey, orderBz)
	return nil
}

// CancelOrder cancels an existing order
func (k Keeper) CancelOrder(ctx sdk.Context, orderID string) error {
	order, err := k.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// Update order status to cancelled
	order.Status = anteilv1.OrderStatus_ORDER_STATUS_CANCELLED
	return k.UpdateOrder(ctx, order)
}

// DeleteOrder removes an order from the store
func (k Keeper) DeleteOrder(ctx sdk.Context, orderID string) error {
	store := ctx.KVStore(k.storeKey)
	orderKey := anteiltypes.GetOrderKey(orderID)

	if !store.Has(orderKey) {
		return anteiltypes.ErrOrderNotFound
	}

	store.Delete(orderKey)
	return nil
}

// GetAllOrders retrieves all orders
func (k Keeper) GetAllOrders(ctx sdk.Context) ([]*anteilv1.Order, error) {
	store := ctx.KVStore(k.storeKey)
	orderStore := anteiltypes.NewOrderStore(store)

	var orders []*anteilv1.Order
	iterator := orderStore.Iterator(nil, nil)
	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()

	for ; iterator.Valid(); iterator.Next() {
		var order anteilv1.Order
		if err := k.cdc.Unmarshal(iterator.Value(), &order); err != nil {
			return nil, fmt.Errorf("failed to unmarshal order: %w", err)
		}
		orders = append(orders, &order)
	}

	return orders, nil
}

// Trade Management Methods

// executeTrade executes a trade between two orders
func (k Keeper) executeTrade(ctx sdk.Context, buyOrderID, sellOrderID string) error {
	buyOrder, err := k.GetOrder(ctx, buyOrderID)
	if err != nil {
		return err
	}

	sellOrder, err := k.GetOrder(ctx, sellOrderID)
	if err != nil {
		return err
	}

	// Validate trade compatibility
	if buyOrder.OrderSide != anteilv1.OrderSide_ORDER_SIDE_BUY {
		return anteiltypes.ErrInvalidOrderType
	}
	if sellOrder.OrderSide != anteilv1.OrderSide_ORDER_SIDE_SELL {
		return anteiltypes.ErrInvalidOrderType
	}

	// Execute the trade
	trade := &anteilv1.Trade{
		TradeId:     fmt.Sprintf("trade_%d", ctx.BlockHeight()),
		BuyOrderId:  buyOrderID,
		SellOrderId: sellOrderID,
		Price:       buyOrder.Price, // Use buy order price
		AntAmount:   buyOrder.AntAmount,
	}

	// Store the trade
	if err := k.SetTrade(ctx, trade); err != nil {
		return err
	}

	// Update order statuses
	buyOrder.Status = anteilv1.OrderStatus_ORDER_STATUS_FILLED
	sellOrder.Status = anteilv1.OrderStatus_ORDER_STATUS_FILLED

	if err := k.UpdateOrder(ctx, buyOrder); err != nil {
		return err
	}
	if err := k.UpdateOrder(ctx, sellOrder); err != nil {
		return err
	}

	return nil
}

// ExecuteTrade executes a trade between orders (public interface)
func (k Keeper) ExecuteTrade(ctx sdk.Context, buyOrderID, sellOrderID string) error {
	return k.executeTrade(ctx, buyOrderID, sellOrderID)
}

// SetTrade stores a trade in the store
func (k Keeper) SetTrade(ctx sdk.Context, trade *anteilv1.Trade) error {
	if err := anteiltypes.IsTradeValid(trade); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	tradeKey := anteiltypes.GetTradeKey(trade.TradeId)

	// Check if trade already exists
	if store.Has(tradeKey) {
		return anteiltypes.ErrTradeAlreadyExists
	}

	// Store the trade
	tradeBz, err := k.cdc.Marshal(trade)
	if err != nil {
		return fmt.Errorf("failed to marshal trade: %w", err)
	}

	store.Set(tradeKey, tradeBz)
	return nil
}

// GetTrade retrieves a trade by ID
func (k Keeper) GetTrade(ctx sdk.Context, tradeID string) (*anteilv1.Trade, error) {
	store := ctx.KVStore(k.storeKey)
	tradeKey := anteiltypes.GetTradeKey(tradeID)

	if !store.Has(tradeKey) {
		return nil, anteiltypes.ErrTradeNotFound
	}

	tradeBz := store.Get(tradeKey)
	var trade anteilv1.Trade
	if err := k.cdc.Unmarshal(tradeBz, &trade); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trade: %w", err)
	}

	return &trade, nil
}

// GetAllTrades retrieves all trades
func (k Keeper) GetAllTrades(ctx sdk.Context) ([]*anteilv1.Trade, error) {
	store := ctx.KVStore(k.storeKey)
	tradeStore := anteiltypes.NewTradeStore(store)

	var trades []*anteilv1.Trade
	iterator := tradeStore.Iterator(nil, nil)
	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()

	for ; iterator.Valid(); iterator.Next() {
		var trade anteilv1.Trade
		if err := k.cdc.Unmarshal(iterator.Value(), &trade); err != nil {
			return nil, fmt.Errorf("failed to unmarshal trade: %w", err)
		}
		trades = append(trades, &trade)
	}

	return trades, nil
}

// Auction Management Methods

// SetAuction stores an auction in the store
func (k Keeper) SetAuction(ctx sdk.Context, auction *anteilv1.Auction) error {
	if err := anteiltypes.IsAuctionValid(auction); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	auctionKey := anteiltypes.GetAuctionKey(auction.AuctionId)

	// Check if auction already exists
	if store.Has(auctionKey) {
		return anteiltypes.ErrAuctionAlreadyExists
	}

	// Store the auction
	auctionBz, err := k.cdc.Marshal(auction)
	if err != nil {
		return fmt.Errorf("failed to marshal auction: %w", err)
	}

	store.Set(auctionKey, auctionBz)
	return nil
}

// CreateAuction creates a new auction (alias for SetAuction)
func (k Keeper) CreateAuction(ctx sdk.Context, auction *anteilv1.Auction) error {
	return k.SetAuction(ctx, auction)
}

// GetAuction retrieves an auction by ID
func (k Keeper) GetAuction(ctx sdk.Context, auctionID string) (*anteilv1.Auction, error) {
	store := ctx.KVStore(k.storeKey)
	auctionKey := anteiltypes.GetAuctionKey(auctionID)

	if !store.Has(auctionKey) {
		return nil, anteiltypes.ErrAuctionNotFound
	}

	auctionBz := store.Get(auctionKey)
	var auction anteilv1.Auction
	if err := k.cdc.Unmarshal(auctionBz, &auction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auction: %w", err)
	}

	return &auction, nil
}

// UpdateAuction updates an existing auction
func (k Keeper) UpdateAuction(ctx sdk.Context, auction *anteilv1.Auction) error {
	if err := anteiltypes.IsAuctionValid(auction); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	auctionKey := anteiltypes.GetAuctionKey(auction.AuctionId)

	// Check if auction exists
	if !store.Has(auctionKey) {
		return anteiltypes.ErrAuctionNotFound
	}

	// Store the updated auction
	auctionBz, err := k.cdc.Marshal(auction)
	if err != nil {
		return fmt.Errorf("failed to marshal auction: %w", err)
	}

	store.Set(auctionKey, auctionBz)
	return nil
}

// GetAllAuctions retrieves all auctions
func (k Keeper) GetAllAuctions(ctx sdk.Context) ([]*anteilv1.Auction, error) {
	store := ctx.KVStore(k.storeKey)
	auctionStore := anteiltypes.NewAuctionStore(store)

	var auctions []*anteilv1.Auction
	iterator := auctionStore.Iterator(nil, nil)
	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()

	for ; iterator.Valid(); iterator.Next() {
		var auction anteilv1.Auction
		if err := k.cdc.Unmarshal(iterator.Value(), &auction); err != nil {
			return nil, fmt.Errorf("failed to unmarshal auction: %w", err)
		}
		auctions = append(auctions, &auction)
	}

	return auctions, nil
}

// ProcessAuctions processes active auctions
func (k Keeper) ProcessAuctions(ctx sdk.Context) error {
	auctions, err := k.GetAllAuctions(ctx)
	if err != nil {
		return err
	}

	for _, auction := range auctions {
		if auction.Status == anteilv1.AuctionStatus_AUCTION_STATUS_OPEN {
			// Check if auction has ended
			if ctx.BlockTime().After(auction.EndTime.AsTime()) {
				auction.Status = anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED
				if err := k.UpdateAuction(ctx, auction); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// BeginBlocker processes auctions and trades
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	// Process active auctions
	if err := k.ProcessAuctions(ctx); err != nil {
		return err
	}

	return nil
}

// PlaceBid places a bid on an auction
func (k Keeper) PlaceBid(ctx sdk.Context, auctionID string, bidder string, amount string) error {
	auction, err := k.GetAuction(ctx, auctionID)
	if err != nil {
		return err
	}

	// Check if auction is still open
	if auction.Status != anteilv1.AuctionStatus_AUCTION_STATUS_OPEN {
		return anteiltypes.ErrAuctionClosed
	}

	// Check if auction has ended
	if ctx.BlockTime().After(auction.EndTime.AsTime()) {
		return anteiltypes.ErrAuctionExpired
	}

	// Create bid
	bid := &anteilv1.Bid{
		BidId:       fmt.Sprintf("%s_%s_%d", auctionID, bidder, ctx.BlockHeight()),
		Bidder:      bidder,
		Amount:      amount,
		SubmittedAt: timestamppb.Now(),
	}

	// Store bid
	bidKey := anteiltypes.GetBidKey(auctionID, bid.BidId)
	store := ctx.KVStore(k.storeKey)
	bidBz, err := k.cdc.Marshal(bid)
	if err != nil {
		return err
	}
	store.Set(bidKey, bidBz)

	// Update auction with new bid
	if auction.WinningBid == "" {
		auction.WinningBid = bid.BidId
		if err := k.UpdateAuction(ctx, auction); err != nil {
			return err
		}
	} else {
		// Get current winning bid to compare amounts
		currentWinningBid, err := k.GetBid(ctx, auctionID, auction.WinningBid)
		if err == nil {
			// Compare amounts as strings (simplified comparison)
			if bid.Amount > currentWinningBid.Amount {
				auction.WinningBid = bid.BidId
				if err := k.UpdateAuction(ctx, auction); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// GetBid retrieves a bid by ID
func (k Keeper) GetBid(ctx sdk.Context, auctionID, bidID string) (*anteilv1.Bid, error) {
	store := ctx.KVStore(k.storeKey)
	bidKey := anteiltypes.GetBidKey(auctionID, bidID)

	if !store.Has(bidKey) {
		return nil, anteiltypes.ErrBidNotFound
	}

	bidBz := store.Get(bidKey)
	var bid anteilv1.Bid
	if err := k.cdc.Unmarshal(bidBz, &bid); err != nil {
		return nil, err
	}

	return &bid, nil
}

// SettleAuction settles an auction and distributes rewards
func (k Keeper) SettleAuction(ctx sdk.Context, auctionID string) error {
	auction, err := k.GetAuction(ctx, auctionID)
	if err != nil {
		return err
	}

	// Check if auction is closed
	if auction.Status != anteilv1.AuctionStatus_AUCTION_STATUS_CLOSED {
		return anteiltypes.ErrAuctionNotClosed
	}

	// Get winning bid
	if auction.WinningBid == "" {
		return anteiltypes.ErrNoWinningBid
	}

	_, err = k.GetBid(ctx, auctionID, auction.WinningBid)
	if err != nil {
		return err
	}

	// Process settlement logic here
	// For now, just mark as settled
	auction.Status = anteilv1.AuctionStatus_AUCTION_STATUS_SETTLED
	return k.UpdateAuction(ctx, auction)
}

// GetUserPosition retrieves user's position in the market
func (k Keeper) GetUserPosition(ctx sdk.Context, user string) (*anteilv1.UserPosition, error) {
	store := ctx.KVStore(k.storeKey)
	positionKey := anteiltypes.GetUserPositionKey(user)

	if !store.Has(positionKey) {
		return nil, anteiltypes.ErrPositionNotFound
	}

	positionBz := store.Get(positionKey)
	var position anteilv1.UserPosition
	if err := k.cdc.Unmarshal(positionBz, &position); err != nil {
		return nil, err
	}

	return &position, nil
}

// SetUserPosition sets user's position
func (k Keeper) SetUserPosition(ctx sdk.Context, position *anteilv1.UserPosition) error {
	store := ctx.KVStore(k.storeKey)
	positionKey := anteiltypes.GetUserPositionKey(position.Owner)

	positionBz, err := k.cdc.Marshal(position)
	if err != nil {
		return err
	}
	store.Set(positionKey, positionBz)

	return nil
}

// UpdateUserPosition updates user's position
func (k Keeper) UpdateUserPosition(ctx sdk.Context, user string, antBalance string, orderCount uint32) error {
	position := &anteilv1.UserPosition{
		Owner:        user,
		AntBalance:   antBalance,
		TotalTrades:  fmt.Sprintf("%d", orderCount),
		LastActivity: timestamppb.Now(),
	}

	return k.SetUserPosition(ctx, position)
}

// GetOrdersByOwner retrieves all orders for a specific owner
func (k Keeper) GetOrdersByOwner(ctx sdk.Context, owner string) ([]*anteilv1.Order, error) {
	store := ctx.KVStore(k.storeKey)
	prefix := anteiltypes.GetOrderPrefix()

	var orders []*anteilv1.Order
	iterator := store.Iterator(prefix, append(prefix, 0xFF))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var order anteilv1.Order
		if err := k.cdc.Unmarshal(iterator.Value(), &order); err != nil {
			continue
		}
		if order.Owner == owner {
			orders = append(orders, &order)
		}
	}

	return orders, nil
}

// EndBlocker processes end-of-block operations
func (k Keeper) EndBlocker(ctx sdk.Context) error {
	// Process any end-of-block logic here
	// For now, just return nil
	return nil
}
