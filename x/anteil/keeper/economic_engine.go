package keeper

import (
	"fmt"
	"sort"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
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

// mustParseDec parses a string to math.LegacyDec, returning zero on failure.
func mustParseDec(s string) math.LegacyDec {
	d, err := math.LegacyNewDecFromStr(s)
	if err != nil {
		return math.LegacyZeroDec()
	}
	return d
}

// ProcessOrderMatching processes order matching for the internal market
func (ee *EconomicEngine) ProcessOrderMatching(ctx sdk.Context) error {
	orders, err := ee.keeper.GetAllOrders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get orders: %w", err)
	}

	engine := NewMatchingEngine()

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

	sort.Slice(engine.buyOrders, func(i, j int) bool {
		pi := mustParseDec(engine.buyOrders[i].Price)
		pj := mustParseDec(engine.buyOrders[j].Price)
		return pi.GT(pj)
	})

	sort.Slice(engine.sellOrders, func(i, j int) bool {
		pi := mustParseDec(engine.sellOrders[i].Price)
		pj := mustParseDec(engine.sellOrders[j].Price)
		return pi.LT(pj)
	})

	return ee.executeMatching(ctx, engine)
}

// executeMatching executes order matching
func (ee *EconomicEngine) executeMatching(ctx sdk.Context, engine *MatchingEngine) error {
	for len(engine.buyOrders) > 0 && len(engine.sellOrders) > 0 {
		buyOrder := engine.buyOrders[0]
		sellOrder := engine.sellOrders[0]

		buyPrice := mustParseDec(buyOrder.Price)
		sellPrice := mustParseDec(sellOrder.Price)

		if buyPrice.GTE(sellPrice) {
			if err := ee.executeTrade(ctx, buyOrder, sellOrder); err != nil {
				return fmt.Errorf("failed to execute trade: %w", err)
			}

			if buyOrder.Status == anteilv1.OrderStatus_ORDER_STATUS_FILLED {
				engine.buyOrders = engine.buyOrders[1:]
			}
			if sellOrder.Status == anteilv1.OrderStatus_ORDER_STATUS_FILLED {
				engine.sellOrders = engine.sellOrders[1:]
			}
		} else {
			break
		}
	}

	return nil
}

// executeTrade executes a trade between two orders using deterministic decimal math
func (ee *EconomicEngine) executeTrade(ctx sdk.Context, buyOrder, sellOrder *anteilv1.Order) error {
	tradePrice := mustParseDec(sellOrder.Price)
	buyQty := mustParseDec(buyOrder.AntAmount)
	sellQty := mustParseDec(sellOrder.AntAmount)

	tradeQty := buyQty
	if sellQty.LT(buyQty) {
		tradeQty = sellQty
	}

	totalValue := tradeQty.Mul(tradePrice)

	trade := &anteilv1.Trade{
		TradeId:     fmt.Sprintf("trade_%s_%s", buyOrder.OrderId, sellOrder.OrderId),
		BuyOrderId:  buyOrder.OrderId,
		SellOrderId: sellOrder.OrderId,
		Buyer:       buyOrder.Owner,
		Seller:      sellOrder.Owner,
		AntAmount:   tradeQty.String(),
		Price:       tradePrice.String(),
		TotalValue:  totalValue.String(),
	}

	newBuyQty := buyQty.Sub(tradeQty)
	newSellQty := sellQty.Sub(tradeQty)

	buyOrder.AntAmount = newBuyQty.String()
	sellOrder.AntAmount = newSellQty.String()

	if newBuyQty.IsZero() {
		buyOrder.Status = anteilv1.OrderStatus_ORDER_STATUS_FILLED
	}
	if newSellQty.IsZero() {
		sellOrder.Status = anteilv1.OrderStatus_ORDER_STATUS_FILLED
	}

	if err := ee.keeper.UpdateOrder(ctx, buyOrder); err != nil {
		return fmt.Errorf("failed to update buy order: %w", err)
	}
	if err := ee.keeper.UpdateOrder(ctx, sellOrder); err != nil {
		return fmt.Errorf("failed to update sell order: %w", err)
	}

	if err := ee.updateUserPositions(ctx, trade, buyOrder.Owner, sellOrder.Owner); err != nil {
		return fmt.Errorf("failed to update user positions: %w", err)
	}

	if err := ee.keeper.SetTrade(ctx, trade); err != nil {
		return fmt.Errorf("failed to store trade: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeTradeExecuted,
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
	tradeQty := mustParseDec(trade.AntAmount)

	if err := ee.keeper.UpdateUserPosition(ctx, buyer, tradeQty.String(), 1); err != nil {
		return fmt.Errorf("failed to update buyer position: %w", err)
	}

	negQty := tradeQty.Neg()
	if err := ee.keeper.UpdateUserPosition(ctx, seller, negQty.String(), 0); err != nil {
		return fmt.Errorf("failed to update seller position: %w", err)
	}

	return nil
}

// ProcessAuctions processes auction settlements
func (ee *EconomicEngine) ProcessAuctions(ctx sdk.Context) error {
	auctions, err := ee.keeper.GetAllAuctions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auctions: %w", err)
	}

	for _, auction := range auctions {
		if auction.Status == anteilv1.AuctionStatus_AUCTION_STATUS_OPEN {
			if err := ee.settleAuction(ctx, auction); err != nil {
				return fmt.Errorf("failed to settle auction %s: %w", auction.AuctionId, err)
			}
		}
	}

	return nil
}

// settleAuction settles an auction using deterministic decimal comparison
func (ee *EconomicEngine) settleAuction(ctx sdk.Context, auction *anteilv1.Auction) error {
	bids, err := ee.keeper.GetBidsByAuction(ctx, auction.AuctionId)
	if err != nil {
		return fmt.Errorf("failed to get bids: %w", err)
	}

	if len(bids) == 0 {
		auction.Status = anteilv1.AuctionStatus_AUCTION_STATUS_CANCELLED
		return ee.keeper.UpdateAuction(ctx, auction)
	}

	var winningBid *anteilv1.Bid
	highestAmount := math.LegacyZeroDec()

	for _, bid := range bids {
		amount := mustParseDec(bid.Amount)
		if amount.GT(highestAmount) {
			highestAmount = amount
			winningBid = bid
		}
	}

	auction.Status = anteilv1.AuctionStatus_AUCTION_STATUS_SETTLED
	auction.WinningBid = winningBid.BidId

	if err := ee.keeper.UpdateAuction(ctx, auction); err != nil {
		return fmt.Errorf("failed to update auction: %w", err)
	}

	auctionQty := mustParseDec(auction.AntAmount)
	if err := ee.keeper.UpdateUserPosition(ctx, winningBid.Bidder, auctionQty.String(), 1); err != nil {
		return fmt.Errorf("failed to update winner position: %w", err)
	}

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

// MarketMetrics represents market statistics using deterministic types
type MarketMetrics struct {
	TotalOrders  int              `json:"total_orders"`
	ActiveOrders int              `json:"active_orders"`
	TotalTrades  int              `json:"total_trades"`
	TotalVolume  math.LegacyDec   `json:"total_volume"`
	AveragePrice math.LegacyDec   `json:"average_price"`
	HighestPrice math.LegacyDec   `json:"highest_price"`
	LowestPrice  math.LegacyDec   `json:"lowest_price"`
	PriceSpread  math.LegacyDec   `json:"price_spread"`
}

// CalculateMarketMetrics calculates market metrics with deterministic math
func (ee *EconomicEngine) CalculateMarketMetrics(ctx sdk.Context) (*MarketMetrics, error) {
	orders, err := ee.keeper.GetAllOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	trades, err := ee.keeper.GetAllTrades(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	maxDecSentinel, _ := math.LegacyNewDecFromStr("999999999999")
	metrics := &MarketMetrics{
		TotalOrders:  len(orders),
		ActiveOrders: 0,
		TotalTrades:  len(trades),
		TotalVolume:  math.LegacyZeroDec(),
		AveragePrice: math.LegacyZeroDec(),
		HighestPrice: math.LegacyZeroDec(),
		LowestPrice:  maxDecSentinel,
		PriceSpread:  math.LegacyZeroDec(),
	}

	for _, order := range orders {
		if order.Status == anteilv1.OrderStatus_ORDER_STATUS_OPEN {
			metrics.ActiveOrders++
		}
	}

	totalValue := math.LegacyZeroDec()
	for _, trade := range trades {
		qty := mustParseDec(trade.AntAmount)
		price := mustParseDec(trade.Price)
		value := qty.Mul(price)

		metrics.TotalVolume = metrics.TotalVolume.Add(qty)
		totalValue = totalValue.Add(value)

		if price.GT(metrics.HighestPrice) {
			metrics.HighestPrice = price
		}
		if price.LT(metrics.LowestPrice) {
			metrics.LowestPrice = price
		}
	}

	if metrics.TotalVolume.GT(math.LegacyZeroDec()) {
		metrics.AveragePrice = totalValue.Quo(metrics.TotalVolume)
	}

	if metrics.HighestPrice.GT(metrics.LowestPrice) {
		metrics.PriceSpread = metrics.HighestPrice.Sub(metrics.LowestPrice)
	} else {
		metrics.PriceSpread = math.LegacyZeroDec()
	}

	return metrics, nil
}

// ProcessMarketMaking handles automated market making via module account
func (ee *EconomicEngine) ProcessMarketMaking(ctx sdk.Context) error {
	metrics, err := ee.CalculateMarketMetrics(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate market price: %w", err)
	}

	if metrics.TotalTrades == 0 {
		return nil
	}

	spreadThreshold, _ := math.LegacyNewDecFromStr("0.1")
	if metrics.PriceSpread.GT(spreadThreshold) {
		if err := ee.createMarketMakingOrders(ctx, metrics.AveragePrice); err != nil {
			return fmt.Errorf("failed to create market making orders: %w", err)
		}
	}

	return nil
}

// createMarketMakingOrders creates market making orders using deterministic math
func (ee *EconomicEngine) createMarketMakingOrders(ctx sdk.Context, marketPrice math.LegacyDec) error {
	params := ee.keeper.GetParams(ctx)

	buyDiscount := mustParseDec(params.MarketMakingBuyDiscount)
	if buyDiscount.IsZero() || buyDiscount.IsNegative() {
		buyDiscount, _ = math.LegacyNewDecFromStr("0.99")
	}

	sellPremium := mustParseDec(params.MarketMakingSellPremium)
	if sellPremium.IsZero() || sellPremium.IsNegative() {
		sellPremium, _ = math.LegacyNewDecFromStr("1.01")
	}

	orderSize := params.MarketMakingOrderSize
	if orderSize == "" {
		orderSize = "1000.0"
	}

	buyPrice := marketPrice.Mul(buyDiscount)
	sellPrice := marketPrice.Mul(sellPremium)

	// Use anteil module account for market making (governance can fund via proposals)
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName).String()

	buyOrder := &anteilv1.Order{
		OrderId:   fmt.Sprintf("mm_buy_%d", ctx.BlockTime().Unix()),
		Owner:     moduleAddr,
		OrderType: anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide: anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount: orderSize,
		Price:     buyPrice.String(),
		Status:    anteilv1.OrderStatus_ORDER_STATUS_OPEN,
	}

	sellOrder := &anteilv1.Order{
		OrderId:   fmt.Sprintf("mm_sell_%d", ctx.BlockTime().Unix()),
		Owner:     moduleAddr,
		OrderType: anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide: anteilv1.OrderSide_ORDER_SIDE_SELL,
		AntAmount: orderSize,
		Price:     sellPrice.String(),
		Status:    anteilv1.OrderStatus_ORDER_STATUS_OPEN,
	}

	if err := ee.keeper.CreateOrder(ctx, buyOrder); err != nil {
		return fmt.Errorf("failed to create market making buy order: %w", err)
	}

	if err := ee.keeper.CreateOrder(ctx, sellOrder); err != nil {
		return fmt.Errorf("failed to create market making sell order: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"market_making_orders_created",
			sdk.NewAttribute("buy_price", buyPrice.String()),
			sdk.NewAttribute("sell_price", sellPrice.String()),
			sdk.NewAttribute("market_price", marketPrice.String()),
		),
	)

	return nil
}
