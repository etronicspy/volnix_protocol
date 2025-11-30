package keeper

import (
	"fmt"
	"strconv"
	"time"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

// IdentKeeperInterface defines the interface for interacting with ident module
// This allows anteil module to get verified citizens for ANT distribution
type IdentKeeperInterface interface {
	GetAllVerifiedAccounts(ctx sdk.Context) ([]*identv1.VerifiedAccount, error)
}

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		paramstore paramtypes.Subspace
		identKeeper IdentKeeperInterface // Optional: for getting verified citizens
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

// SetIdentKeeper sets the ident keeper interface for getting verified citizens
func (k *Keeper) SetIdentKeeper(identKeeper IdentKeeperInterface) {
	k.identKeeper = identKeeper
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
			// Log error instead of panicking - iterator close failures are non-critical
			// but should be logged for debugging
			ctx.Logger().Error("failed to close iterator", "error", err)
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
		TradeId:     fmt.Sprintf("trade_%s_%s", buyOrderID, sellOrderID),
		BuyOrderId:  buyOrderID,
		SellOrderId: sellOrderID,
		Buyer:       buyOrder.Owner,
		Seller:      sellOrder.Owner,
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

	// Update user positions
	if err := k.updateUserPositionForTrade(ctx, trade); err != nil {
		return err
	}

	return nil
}

// ExecuteTrade executes a trade between orders (public interface)
func (k Keeper) ExecuteTrade(ctx sdk.Context, buyOrderID, sellOrderID string) error {
	return k.executeTrade(ctx, buyOrderID, sellOrderID)
}

// updateUserPositionForTrade updates user positions after a trade
func (k Keeper) updateUserPositionForTrade(ctx sdk.Context, trade *anteilv1.Trade) error {
	// Update buyer position
	buyerPosition, err := k.GetUserPosition(ctx, trade.Buyer)
	if err != nil {
		// Create new position if not found
		buyerPosition = &anteilv1.UserPosition{
			Owner:        trade.Buyer,
			AntBalance:   "0",
			TotalTrades:  "0",
			TotalVolume:  "0",
			LastActivity: timestamppb.Now(),
		}
	}

	// Update buyer stats
	buyerTrades := anteiltypes.ParseUint64(buyerPosition.TotalTrades)
	buyerVolume := anteiltypes.ParseUint64(buyerPosition.TotalVolume)
	tradeAmount := anteiltypes.ParseUint64(trade.AntAmount)

	buyerPosition.TotalTrades = fmt.Sprintf("%d", buyerTrades+1)
	buyerPosition.TotalVolume = fmt.Sprintf("%d", buyerVolume+tradeAmount)
	buyerPosition.LastActivity = timestamppb.Now()

	if err := k.SetUserPosition(ctx, buyerPosition); err != nil {
		return err
	}

	// Update seller position
	sellerPosition, err := k.GetUserPosition(ctx, trade.Seller)
	if err != nil {
		// Create new position if not found
		sellerPosition = &anteilv1.UserPosition{
			Owner:        trade.Seller,
			AntBalance:   "0",
			TotalTrades:  "0",
			TotalVolume:  "0",
			LastActivity: timestamppb.Now(),
		}
	}

	// Update seller stats
	sellerTrades := anteiltypes.ParseUint64(sellerPosition.TotalTrades)
	sellerVolume := anteiltypes.ParseUint64(sellerPosition.TotalVolume)

	sellerPosition.TotalTrades = fmt.Sprintf("%d", sellerTrades+1)
	sellerPosition.TotalVolume = fmt.Sprintf("%d", sellerVolume+tradeAmount)
	sellerPosition.LastActivity = timestamppb.Now()

	if err := k.SetUserPosition(ctx, sellerPosition); err != nil {
		return err
	}

	return nil
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
			// Log error instead of panicking - iterator close failures are non-critical
			// but should be logged for debugging
			ctx.Logger().Error("failed to close iterator", "error", err)
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

// BeginBlocker processes auctions and trades, and distributes ANT to citizens
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	// Process active auctions
	if err := k.ProcessAuctions(ctx); err != nil {
		return err
	}

	// Check if it's time to distribute ANT to citizens
	params := k.GetParams(ctx)
	lastDistributionTime, err := k.GetLastDistributionTime(ctx)
	if err != nil {
		ctx.Logger().Error("Failed to get last distribution time", "error", err)
		// Continue anyway - will distribute on next block
	} else {
		currentTime := ctx.BlockTime()
		timeSinceLastDistribution := currentTime.Sub(lastDistributionTime)

		// Distribute if enough time has passed (or if first distribution)
		if lastDistributionTime.IsZero() || timeSinceLastDistribution >= params.CitizenAntDistributionPeriod {
			if err := k.DistributeAntToCitizens(ctx); err != nil {
				ctx.Logger().Error("Failed to distribute ANT to citizens", "error", err)
				// Don't fail the block if distribution fails
			} else {
				// Update last distribution time
				if err := k.SetLastDistributionTime(ctx, currentTime); err != nil {
					ctx.Logger().Error("Failed to set last distribution time", "error", err)
				}
			}
		}
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

// BurnAntFromUser burns all ANT tokens from a user's position
// According to whitepaper: "его права на ANT сгорают" when citizen is deactivated
func (k Keeper) BurnAntFromUser(ctx sdk.Context, user string) error {
	// Get user position
	position, err := k.GetUserPosition(ctx, user)
	if err != nil {
		// If position doesn't exist, nothing to burn
		ctx.Logger().Info("No ANT position found for user, nothing to burn", "user", user)
		return nil
	}

	// Get current balance
	currentBalance, err := strconv.ParseUint(position.AntBalance, 10, 64)
	if err != nil {
		currentBalance = 0
	}

	// If balance is already zero, nothing to burn
	if currentBalance == 0 {
		ctx.Logger().Info("User has zero ANT balance, nothing to burn", "user", user)
		return nil
	}

	// Set balance to zero (burn all ANT)
	position.AntBalance = "0"
	position.AvailableAnt = "0"
	position.LockedAnt = "0"
	position.LastActivity = timestamppb.Now()

	// Update position
	if err := k.SetUserPosition(ctx, position); err != nil {
		return fmt.Errorf("failed to update position after burning ANT: %w", err)
	}

	ctx.Logger().Info("ANT burned from user", "user", user, "amount", currentBalance)

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
	defer func() {
		if err := iterator.Close(); err != nil {
			// Log error instead of panicking - iterator close failures are non-critical
			// but should be logged for debugging
			ctx.Logger().Error("failed to close iterator", "error", err)
		}
	}()

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

// DistributeAntToCitizens distributes ANT tokens to verified citizens
// According to whitepaper: Citizens automatically receive ANT (10 ANT per day by default)
func (k Keeper) DistributeAntToCitizens(ctx sdk.Context) error {
	// Check if ident keeper is available
	if k.identKeeper == nil {
		ctx.Logger().Info("Ident keeper not set, skipping ANT distribution")
		return nil
	}

	// Get all verified accounts
	allAccounts, err := k.identKeeper.GetAllVerifiedAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get verified accounts: %w", err)
	}

	// Get parameters
	params := k.GetParams(ctx)
	rewardRate, err := strconv.ParseUint(params.CitizenAntRewardRate, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid citizen ANT reward rate: %w", err)
	}

	accumulationLimit, err := strconv.ParseUint(params.CitizenAntAccumulationLimit, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid citizen ANT accumulation limit: %w", err)
	}

	// Filter citizens and distribute ANT
	distributedCount := 0
	for _, account := range allAccounts {
		// Only process active citizens
		if account.Role != identv1.Role_ROLE_CITIZEN || !account.IsActive {
			continue
		}

		// Get or create user position
		position, err := k.GetUserPosition(ctx, account.Address)
		if err != nil {
			// Create new position if not found
			position = anteiltypes.NewUserPosition(account.Address, "0")
		}

		// Check accumulation limit
		currentBalance, err := strconv.ParseUint(position.AntBalance, 10, 64)
		if err != nil {
			currentBalance = 0
		}

		// Skip if already at limit
		if currentBalance >= accumulationLimit {
			ctx.Logger().Debug("Citizen at accumulation limit", "citizen", account.Address, "balance", currentBalance, "limit", accumulationLimit)
			continue
		}

		// Calculate new balance (respect limit)
		newBalance := currentBalance + rewardRate
		if newBalance > accumulationLimit {
			newBalance = accumulationLimit
		}

		// Update position
		position.AntBalance = fmt.Sprintf("%d", newBalance)
		position.AvailableAnt = position.AntBalance // Available = total (not locked)
		position.LastActivity = timestamppb.Now()

		if err := k.SetUserPosition(ctx, position); err != nil {
			ctx.Logger().Error("Failed to update citizen position", "citizen", account.Address, "error", err)
			continue // Continue with other citizens
		}

		distributedCount++

		// Emit event for ANT distribution
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"anteil.ant_distributed",
				sdk.NewAttribute("citizen", account.Address),
				sdk.NewAttribute("amount", fmt.Sprintf("%d", rewardRate)),
				sdk.NewAttribute("new_balance", fmt.Sprintf("%d", newBalance)),
				sdk.NewAttribute("limit", fmt.Sprintf("%d", accumulationLimit)),
			),
		)

		ctx.Logger().Info("ANT distributed to citizen", "citizen", account.Address, "amount", rewardRate, "new_balance", newBalance)
	}

	ctx.Logger().Info("ANT distribution completed", "citizens_distributed", distributedCount, "total_citizens", len(allAccounts))
	return nil
}

// GetLastDistributionTime returns the last time ANT was distributed to citizens
func (k Keeper) GetLastDistributionTime(ctx sdk.Context) (time.Time, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(anteiltypes.LastDistributionTimeKey)
	if bz == nil {
		// Return zero time if not set (first distribution)
		return time.Time{}, nil
	}

	var lastTime time.Time
	if err := lastTime.UnmarshalBinary(bz); err != nil {
		return time.Time{}, fmt.Errorf("failed to unmarshal last distribution time: %w", err)
	}

	return lastTime, nil
}

// SetLastDistributionTime sets the last time ANT was distributed to citizens
func (k Keeper) SetLastDistributionTime(ctx sdk.Context, t time.Time) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := t.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal distribution time: %w", err)
	}

	store.Set(anteiltypes.LastDistributionTimeKey, bz)
	return nil
}

// EndBlocker processes end-of-block operations
func (k Keeper) EndBlocker(ctx sdk.Context) error {
	// Create economic engine
	engine := NewEconomicEngine(&k)

	// Process order matching
	if err := engine.ProcessOrderMatching(ctx); err != nil {
		// Log error but continue
		ctx.Logger().Error("Failed to process order matching", "error", err)
	}

	// Process auctions
	if err := engine.ProcessAuctions(ctx); err != nil {
		// Log error but continue
		ctx.Logger().Error("Failed to process auctions", "error", err)
	}

	// Process market making
	if err := engine.ProcessMarketMaking(ctx); err != nil {
		// Log error but continue
		ctx.Logger().Error("Failed to process market making", "error", err)
	}

	return nil
}

// GetBidsByAuction returns all bids for a specific auction
func (k Keeper) GetBidsByAuction(ctx sdk.Context, auctionID string) ([]*anteilv1.Bid, error) {
	// For now, return empty slice - in real implementation would use proper indexing
	// This is a simplified version for demo purposes
	return []*anteilv1.Bid{}, nil
}
