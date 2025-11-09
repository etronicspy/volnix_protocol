package keeper

import (
	"fmt"
	"sort"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
)

// EconomicEngine handles advanced economic operations
type EconomicEngine struct {
	keeper *Keeper
}

// NewEconomicEngine creates a new economic engine
func NewEconomicEngine(keeper *Keeper) *EconomicEngine {
	return &EconomicEngine{
		keeper: keeper,
	}
}

// MatchingEngine handles order matching and execution
type MatchingEngine struct {
	buyOrders  []*anteilv1.Order
	sellOrders []*anteilv1.Order
}

// NewMatchingEngine creates a new matching engine
func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		buyOrders:  make([]*anteilv1.Order, 0),
		sellOrders: make([]*anteilv1.Order, 0),
	}
}

// ProcessOrderMatching processes order matching for the internal market
func (ee *EconomicEngine) ProcessOrderMatching(ctx sdk.Context) error {
	// Get all active orders
	orders, err := ee.keeper.GetAllOrders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get orders: %w", err)
	}

	// Create matching engine
	engine := NewMatchingEngine()

	// Separate buy and sell orders
	for _, order := range orders {
		if order.Status == anteilv1.OrderStatus_ORDER_STATUS_OPEN {
			switch order.OrderSide {
			case anteilv1.OrderSide_ORDER_SIDE_BUY:
				engine.buyOrders = append(engine.buyOrders, order)
			case anteilv1.OrderSide_ORDER_SIDE_SELL:
				engine.sellOrders = append(engine.sellOrders, order)
			}
		}
	}

	// Sort orders by price (buy orders descending, sell orders ascending)
	sort.Slice(engine.buyOrders, func(i, j int) bool {
		priceI, _ := strconv.ParseFloat(engine.buyOrders[i].Price, 64)
		priceJ, _ := strconv.ParseFloat(engine.buyOrders[j].Price, 64)
		return priceI > priceJ // Higher prices first for buy orders
	})

	sort.Slice(engine.sellOrders, func(i, j int) bool {
		priceI, _ := strconv.ParseFloat(engine.sellOrders[i].Price, 64)
		priceJ, _ := strconv.ParseFloat(engine.sellOrders[j].Price, 64)
		return priceI < priceJ // Lower prices first for sell orders
	})

	// Execute matching
	return ee.executeMatching(ctx, engine)
}

// executeMatching executes order matching
func (ee *EconomicEngine) executeMatching(ctx sdk.Context, engine *MatchingEngine) error {
	for len(engine.buyOrders) > 0 && len(engine.sellOrders) > 0 {
		buyOrder := engine.buyOrders[0]
		sellOrder := engine.sellOrders[0]

		buyPrice, _ := strconv.ParseFloat(buyOrder.Price, 64)
		sellPrice, _ := strconv.ParseFloat(sellOrder.Price, 64)

		// Check if orders can match
		if buyPrice >= sellPrice {
			// Execute trade
			if err := ee.executeTrade(ctx, buyOrder, sellOrder); err != nil {
				return fmt.Errorf("failed to execute trade: %w", err)
			}

			// Remove completed orders
			if buyOrder.Status == anteilv1.OrderStatus_ORDER_STATUS_FILLED {
				engine.buyOrders = engine.buyOrders[1:]
			}
			if sellOrder.Status == anteilv1.OrderStatus_ORDER_STATUS_FILLED {
				engine.sellOrders = engine.sellOrders[1:]
			}
		} else {
			// No more matches possible
			break
		}
	}

	return nil
}

// executeTrade executes a trade between two orders
func (ee *EconomicEngine) executeTrade(ctx sdk.Context, buyOrder, sellOrder *anteilv1.Order) error {
	// Determine trade price (use sell order price for simplicity)
	tradePrice, _ := strconv.ParseFloat(sellOrder.Price, 64)

	// Determine trade quantity (minimum of both orders)
	buyQty, _ := strconv.ParseFloat(buyOrder.AntAmount, 64)
	sellQty, _ := strconv.ParseFloat(sellOrder.AntAmount, 64)
	tradeQty := buyQty
	if sellQty < buyQty {
		tradeQty = sellQty
	}

	// Create trade record
	trade := &anteilv1.Trade{
		TradeId:     fmt.Sprintf("trade_%s_%s", buyOrder.OrderId, sellOrder.OrderId),
		BuyOrderId:  buyOrder.OrderId,
		SellOrderId: sellOrder.OrderId,
		Buyer:       buyOrder.Owner,
		Seller:      sellOrder.Owner,
		AntAmount:   fmt.Sprintf("%.6f", tradeQty),
		Price:       fmt.Sprintf("%.6f", tradePrice),
		TotalValue:  fmt.Sprintf("%.6f", tradeQty*tradePrice),
	}

	// Update order quantities
	newBuyQty := buyQty - tradeQty
	newSellQty := sellQty - tradeQty

	buyOrder.AntAmount = fmt.Sprintf("%.6f", newBuyQty)
	sellOrder.AntAmount = fmt.Sprintf("%.6f", newSellQty)

	// Update order status if fully filled
	if newBuyQty == 0 {
		buyOrder.Status = anteilv1.OrderStatus_ORDER_STATUS_FILLED
	}
	if newSellQty == 0 {
		sellOrder.Status = anteilv1.OrderStatus_ORDER_STATUS_FILLED
	}

	// Update orders in store
	if err := ee.keeper.UpdateOrder(ctx, buyOrder); err != nil {
		return fmt.Errorf("failed to update buy order: %w", err)
	}
	if err := ee.keeper.UpdateOrder(ctx, sellOrder); err != nil {
		return fmt.Errorf("failed to update sell order: %w", err)
	}

	// Update user positions
	if err := ee.updateUserPositions(ctx, trade, buyOrder.Owner, sellOrder.Owner); err != nil {
		return fmt.Errorf("failed to update user positions: %w", err)
	}

	// Store trade record
	if err := ee.keeper.SetTrade(ctx, trade); err != nil {
		return fmt.Errorf("failed to store trade: %w", err)
	}

	// Emit trade event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"trade_executed",
			sdk.NewAttribute("trade_id", trade.TradeId),
			sdk.NewAttribute("buy_order_id", buyOrder.OrderId),
			sdk.NewAttribute("sell_order_id", sellOrder.OrderId),
			sdk.NewAttribute("quantity", trade.AntAmount),
			sdk.NewAttribute("price", trade.Price),
			sdk.NewAttribute("buyer", buyOrder.Owner),
			sdk.NewAttribute("seller", sellOrder.Owner),
		),
	)

	return nil
}

// updateUserPositions updates user ANT positions after trade
func (ee *EconomicEngine) updateUserPositions(ctx sdk.Context, trade *anteilv1.Trade, buyer, seller string) error {
	tradeQty, _ := strconv.ParseFloat(trade.AntAmount, 64)

	// Update buyer position (increase ANT)
	if err := ee.keeper.UpdateUserPosition(ctx, buyer, fmt.Sprintf("%.6f", tradeQty), 1); err != nil {
		return fmt.Errorf("failed to update buyer position: %w", err)
	}

	// Update seller position (decrease ANT) - use 0 for decrease operation
	if err := ee.keeper.UpdateUserPosition(ctx, seller, fmt.Sprintf("%.6f", -tradeQty), 0); err != nil {
		return fmt.Errorf("failed to update seller position: %w", err)
	}

	return nil
}

// ProcessAuctions processes auction settlements
func (ee *EconomicEngine) ProcessAuctions(ctx sdk.Context) error {
	// Get all active auctions
	auctions, err := ee.keeper.GetAllAuctions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auctions: %w", err)
	}

	for _, auction := range auctions {
		if auction.Status == anteilv1.AuctionStatus_AUCTION_STATUS_OPEN {
			// Check if auction should be settled (simplified: settle all active auctions)
			if err := ee.settleAuction(ctx, auction); err != nil {
				return fmt.Errorf("failed to settle auction %s: %w", auction.AuctionId, err)
			}
		}
	}

	return nil
}

// settleAuction settles an auction
func (ee *EconomicEngine) settleAuction(ctx sdk.Context, auction *anteilv1.Auction) error {
	// Get all bids for this auction
	bids, err := ee.keeper.GetBidsByAuction(ctx, auction.AuctionId)
	if err != nil {
		return fmt.Errorf("failed to get bids: %w", err)
	}

	if len(bids) == 0 {
		// No bids, cancel auction
		auction.Status = anteilv1.AuctionStatus_AUCTION_STATUS_CANCELLED
		return ee.keeper.UpdateAuction(ctx, auction)
	}

	// Find highest bid
	var winningBid *anteilv1.Bid
	highestAmount := 0.0

	for _, bid := range bids {
		amount, _ := strconv.ParseFloat(bid.Amount, 64)
		if amount > highestAmount {
			highestAmount = amount
			winningBid = bid
		}
	}

	// Settle auction
	auction.Status = anteilv1.AuctionStatus_AUCTION_STATUS_SETTLED
	auction.WinningBid = winningBid.BidId // Use BidId instead of Amount

	// Update auction in store
	if err := ee.keeper.UpdateAuction(ctx, auction); err != nil {
		return fmt.Errorf("failed to update auction: %w", err)
	}

	// Update winner's position
	auctionQty, _ := strconv.ParseFloat(auction.AntAmount, 64)
	if err := ee.keeper.UpdateUserPosition(ctx, winningBid.Bidder, fmt.Sprintf("%.6f", auctionQty), 1); err != nil {
		return fmt.Errorf("failed to update winner position: %w", err)
	}

	// Emit auction settled event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"auction_settled",
			sdk.NewAttribute("auction_id", auction.AuctionId),
			sdk.NewAttribute("winner", winningBid.Bidder),
			sdk.NewAttribute("winning_bid", auction.WinningBid),
			sdk.NewAttribute("quantity", auction.AntAmount),
		),
	)

	return nil
}

// CalculateMarketMetrics calculates market metrics
func (ee *EconomicEngine) CalculateMarketMetrics(ctx sdk.Context) (*MarketMetrics, error) {
	// Get all orders
	orders, err := ee.keeper.GetAllOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	// Get all trades
	trades, err := ee.keeper.GetAllTrades(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	// Calculate metrics
	metrics := &MarketMetrics{
		TotalOrders:  len(orders),
		ActiveOrders: 0,
		TotalTrades:  len(trades),
		TotalVolume:  0.0,
		AveragePrice: 0.0,
		HighestPrice: 0.0,
		LowestPrice:  999999.0,
		PriceSpread:  0.0,
	}

	// Count active orders
	for _, order := range orders {
		if order.Status == anteilv1.OrderStatus_ORDER_STATUS_OPEN {
			metrics.ActiveOrders++
		}
	}

	// Calculate trade metrics
	totalValue := 0.0
	for _, trade := range trades {
		qty, _ := strconv.ParseFloat(trade.AntAmount, 64)
		price, _ := strconv.ParseFloat(trade.Price, 64)
		value := qty * price

		metrics.TotalVolume += qty
		totalValue += value

		if price > metrics.HighestPrice {
			metrics.HighestPrice = price
		}
		if price < metrics.LowestPrice {
			metrics.LowestPrice = price
		}
	}

	// Calculate average price
	if metrics.TotalVolume > 0 {
		metrics.AveragePrice = totalValue / metrics.TotalVolume
	}

	// Calculate price spread
	metrics.PriceSpread = metrics.HighestPrice - metrics.LowestPrice

	return metrics, nil
}

// MarketMetrics represents market statistics
type MarketMetrics struct {
	TotalOrders  int     `json:"total_orders"`
	ActiveOrders int     `json:"active_orders"`
	TotalTrades  int     `json:"total_trades"`
	TotalVolume  float64 `json:"total_volume"`
	AveragePrice float64 `json:"average_price"`
	HighestPrice float64 `json:"highest_price"`
	LowestPrice  float64 `json:"lowest_price"`
	PriceSpread  float64 `json:"price_spread"`
}

// ProcessMarketMaking handles automated market making
func (ee *EconomicEngine) ProcessMarketMaking(ctx sdk.Context) error {
	// Get market metrics
	metrics, err := ee.calculateCurrentMarketPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate market price: %w", err)
	}

	// Create market making orders if spread is too wide
	if metrics.PriceSpread > 0.1 { // 10% spread threshold
		if err := ee.createMarketMakingOrders(ctx, metrics.AveragePrice); err != nil {
			return fmt.Errorf("failed to create market making orders: %w", err)
		}
	}

	return nil
}

// calculateCurrentMarketPrice calculates current market price
func (ee *EconomicEngine) calculateCurrentMarketPrice(ctx sdk.Context) (*MarketMetrics, error) {
	return ee.CalculateMarketMetrics(ctx)
}

// createMarketMakingOrders creates market making orders
func (ee *EconomicEngine) createMarketMakingOrders(ctx sdk.Context, marketPrice float64) error {
	// Create buy order slightly below market price
	buyPrice := marketPrice * 0.99 // 1% below market
	buyOrder := &anteilv1.Order{
		OrderId:   fmt.Sprintf("mm_buy_%d", ctx.BlockTime().Unix()),
		Owner:     "market_maker_system",
		OrderType: anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide: anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount: "1000.0",
		Price:     fmt.Sprintf("%.6f", buyPrice),
		Status:    anteilv1.OrderStatus_ORDER_STATUS_OPEN,
	}

	// Create sell order slightly above market price
	sellPrice := marketPrice * 1.01 // 1% above market
	sellOrder := &anteilv1.Order{
		OrderId:   fmt.Sprintf("mm_sell_%d", ctx.BlockTime().Unix()),
		Owner:     "market_maker_system",
		OrderType: anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide: anteilv1.OrderSide_ORDER_SIDE_SELL,
		AntAmount: "1000.0",
		Price:     fmt.Sprintf("%.6f", sellPrice),
		Status:    anteilv1.OrderStatus_ORDER_STATUS_OPEN,
	}

	// Store orders
	if err := ee.keeper.CreateOrder(ctx, buyOrder); err != nil {
		return fmt.Errorf("failed to create market making buy order: %w", err)
	}

	if err := ee.keeper.CreateOrder(ctx, sellOrder); err != nil {
		return fmt.Errorf("failed to create market making sell order: %w", err)
	}

	// Emit market making event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"market_making_orders_created",
			sdk.NewAttribute("buy_price", fmt.Sprintf("%.6f", buyPrice)),
			sdk.NewAttribute("sell_price", fmt.Sprintf("%.6f", sellPrice)),
			sdk.NewAttribute("market_price", fmt.Sprintf("%.6f", marketPrice)),
		),
	)

	return nil
}
