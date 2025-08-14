package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

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
func (k Keeper) GetAllOrders(ctx sdk.Context) []*anteilv1.Order {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, anteiltypes.OrderKeyPrefix)

	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()
	var orders []*anteilv1.Order

	for ; iterator.Valid(); iterator.Next() {
		var order anteilv1.Order
		if err := k.cdc.Unmarshal(iterator.Value(), &order); err != nil {
			continue
		}
		orders = append(orders, &order)
	}

	return orders
}

// GetOrdersByOwner retrieves orders by owner
func (k Keeper) GetOrdersByOwner(ctx sdk.Context, owner string) []*anteilv1.Order {
	allOrders := k.GetAllOrders(ctx)
	var ownerOrders []*anteilv1.Order

	for _, order := range allOrders {
		if order.GetOwner() == owner {
			ownerOrders = append(ownerOrders, order)
		}
	}

	return ownerOrders
}

// GetOrdersByStatus retrieves orders by status
func (k Keeper) GetOrdersByStatus(ctx sdk.Context, status anteilv1.OrderStatus) []*anteilv1.Order {
	allOrders := k.GetAllOrders(ctx)
	var statusOrders []*anteilv1.Order

	for _, order := range allOrders {
		if order.GetStatus() == status {
			statusOrders = append(statusOrders, order)
		}
	}

	return statusOrders
}

// Trade Management Methods

// SetTrade stores a trade in the store
func (k Keeper) SetTrade(ctx sdk.Context, trade *anteilv1.Trade) error {
	if err := anteiltypes.IsTradeValid(trade); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	tradeKey := anteiltypes.GetTradeKey(trade.GetTradeId())

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
		return nil, fmt.Errorf("trade not found")
	}

	tradeBz := store.Get(tradeKey)
	var trade anteilv1.Trade
	if err := k.cdc.Unmarshal(tradeBz, &trade); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trade: %w", err)
	}

	return &trade, nil
}

// GetAllTrades retrieves all trades
func (k Keeper) GetAllTrades(ctx sdk.Context) []*anteilv1.Trade {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, anteiltypes.TradeKeyPrefix)

	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()
	var trades []*anteilv1.Trade

	for ; iterator.Valid(); iterator.Next() {
		var trade anteilv1.Trade
		if err := k.cdc.Unmarshal(iterator.Value(), &trade); err != nil {
			continue
		}
		trades = append(trades, &trade)
	}

	return trades
}

// UserPosition Management Methods

// SetUserPosition stores a user position in the store
func (k Keeper) SetUserPosition(ctx sdk.Context, position *anteilv1.UserPosition) error {
	if err := anteiltypes.IsUserPositionValid(position); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	positionKey := anteiltypes.GetUserPositionKey(position.GetOwner())

	// Store the position
	positionBz, err := k.cdc.Marshal(position)
	if err != nil {
		return fmt.Errorf("failed to marshal user position: %w", err)
	}

	store.Set(positionKey, positionBz)
	return nil
}

// GetUserPosition retrieves a user position by owner
func (k Keeper) GetUserPosition(ctx sdk.Context, owner string) (*anteilv1.UserPosition, error) {
	store := ctx.KVStore(k.storeKey)
	positionKey := anteiltypes.GetUserPositionKey(owner)

	if !store.Has(positionKey) {
		return nil, fmt.Errorf("user position not found")
	}

	positionBz := store.Get(positionKey)
	var position anteilv1.UserPosition
	if err := k.cdc.Unmarshal(positionBz, &position); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user position: %w", err)
	}

	return &position, nil
}

// GetAllUserPositions retrieves all user positions
func (k Keeper) GetAllUserPositions(ctx sdk.Context) []*anteilv1.UserPosition {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, anteiltypes.UserPositionKeyPrefix)

	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()
	var positions []*anteilv1.UserPosition

	for ; iterator.Valid(); iterator.Next() {
		var position anteilv1.UserPosition
		if err := k.cdc.Unmarshal(iterator.Value(), &position); err != nil {
			continue
		}
		positions = append(positions, &position)
	}

	return positions
}

// Auction Management Methods

// SetAuction stores an auction in the store
func (k Keeper) SetAuction(ctx sdk.Context, auction *anteilv1.Auction) error {
	if err := anteiltypes.IsAuctionValid(auction); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	auctionKey := anteiltypes.GetAuctionKey(auction.GetAuctionId())

	// Store the auction
	auctionBz, err := k.cdc.Marshal(auction)
	if err != nil {
		return fmt.Errorf("failed to marshal auction: %w", err)
	}

	store.Set(auctionKey, auctionBz)
	return nil
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

// GetAllAuctions retrieves all auctions
func (k Keeper) GetAllAuctions(ctx sdk.Context) []*anteilv1.Auction {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, anteiltypes.AuctionKeyPrefix)

	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()
	var auctions []*anteilv1.Auction

	for ; iterator.Valid(); iterator.Next() {
		var auction anteilv1.Auction
		if err := k.cdc.Unmarshal(iterator.Value(), &auction); err != nil {
			continue
		}
		auctions = append(auctions, &auction)
	}

	return auctions
}

// GetAuctionsByStatus retrieves auctions by status
func (k Keeper) GetAuctionsByStatus(ctx sdk.Context, status anteilv1.AuctionStatus) []*anteilv1.Auction {
	allAuctions := k.GetAllAuctions(ctx)
	var statusAuctions []*anteilv1.Auction

	for _, auction := range allAuctions {
		if auction.GetStatus() == status {
			statusAuctions = append(statusAuctions, auction)
		}
	}

	return statusAuctions
}

// OrderBook Management Methods

// GetOrderBook retrieves the current order book
func (k Keeper) GetOrderBook(ctx sdk.Context) *anteilv1.OrderBook {
	openOrders := k.GetOrdersByStatus(ctx, anteilv1.OrderStatus_ORDER_STATUS_OPEN)

	// Build order book from open orders
	orderBook := &anteilv1.OrderBook{
		BuyOrders:   []*anteilv1.OrderBookEntry{},
		SellOrders:  []*anteilv1.OrderBookEntry{},
		LastPrice:   "0",
		Volume_24H:  "0",
		TotalOrders: uint64(len(openOrders)),
	}

	// Aggregate buy and sell orders by price
	buyOrders := make(map[string]*anteilv1.OrderBookEntry)
	sellOrders := make(map[string]*anteilv1.OrderBookEntry)

	for _, order := range openOrders {
		if order.GetOrderSide() == anteilv1.OrderSide_ORDER_SIDE_BUY {
			if entry, exists := buyOrders[order.GetPrice()]; exists {
				// Aggregate at same price level
				// In real implementation, this would use decimal arithmetic
				entry.TotalAmount = order.GetAntAmount()
				entry.OrderCount++
			} else {
				buyOrders[order.GetPrice()] = &anteilv1.OrderBookEntry{
					Price:       order.GetPrice(),
					TotalAmount: order.GetAntAmount(),
					OrderCount:  1,
				}
			}
		} else if order.GetOrderSide() == anteilv1.OrderSide_ORDER_SIDE_SELL {
			if entry, exists := sellOrders[order.GetPrice()]; exists {
				// Aggregate at same price level
				entry.TotalAmount = order.GetAntAmount()
				entry.OrderCount++
			} else {
				sellOrders[order.GetPrice()] = &anteilv1.OrderBookEntry{
					Price:       order.GetPrice(),
					TotalAmount: order.GetAntAmount(),
					OrderCount:  1,
				}
			}
		}
	}

	// Convert maps to slices and sort
	// In real implementation, this would sort by price
	for _, entry := range buyOrders {
		orderBook.BuyOrders = append(orderBook.BuyOrders, entry)
	}
	for _, entry := range sellOrders {
		orderBook.SellOrders = append(orderBook.SellOrders, entry)
	}

	return orderBook
}
