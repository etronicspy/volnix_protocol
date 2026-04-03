package keeper

import (
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
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
type IdentKeeperInterface interface {
	GetAllVerifiedAccounts(ctx sdk.Context) ([]*identv1.VerifiedAccount, error)
	GetVerifiedAccount(ctx sdk.Context, address string) (*identv1.VerifiedAccount, error)
}

// BankKeeperInterface defines bank operations needed for order escrow §4.1
type BankKeeperInterface interface {
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type (
	Keeper struct {
		cdc         codec.BinaryCodec
		storeKey    storetypes.StoreKey
		paramstore  paramtypes.Subspace
		identKeeper IdentKeeperInterface
		bankKeeper  BankKeeperInterface
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

// SetBankKeeper sets the bank keeper for escrow operations §4.1
func (k *Keeper) SetBankKeeper(bk BankKeeperInterface) {
	k.bankKeeper = bk
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

	// Validate account exists and has valid role (CITIZEN or VALIDATOR) for order creation
	if k.identKeeper != nil {
		account, err := k.identKeeper.GetVerifiedAccount(ctx, order.Owner)
		if err != nil || account == nil {
			return anteiltypes.ErrAccountNotFound
		}
		if !account.IsActive {
			return anteiltypes.ErrAccountNotFound
		}
		switch account.Role {
		case identv1.Role_ROLE_SUPPLIER, identv1.Role_ROLE_VALIDATOR:
			// Allowed
		default:
			return anteiltypes.ErrInvalidRoleForOrder
		}
	}

	// §4.1 / §5.2 — Escrow: lock collateral when placing an order
	if order.OrderSide == anteilv1.OrderSide_ORDER_SIDE_SELL {
		// Sell side: lock ANT from user position
		pos, err := k.GetUserPosition(ctx, order.Owner)
		if err != nil || pos == nil {
			return anteiltypes.ErrInsufficientBalance
		}
		available := anteiltypes.ParseUint64(pos.AvailableAnt)
		sellAmount := anteiltypes.ParseUint64(order.AntAmount)
		if sellAmount > available {
			return anteiltypes.ErrInsufficientBalance
		}
		// Move ANT from available to locked (escrow)
		pos.AvailableAnt = fmt.Sprintf("%d", available-sellAmount)
		pos.LockedAnt = fmt.Sprintf("%d", anteiltypes.ParseUint64(pos.LockedAnt)+sellAmount)
		pos.OpenOrderIds = append(pos.OpenOrderIds, order.OrderId)
		order.EscrowedAmount = order.AntAmount
		if err := k.SetUserPosition(ctx, pos); err != nil {
			return fmt.Errorf("failed to update position for escrow: %w", err)
		}
	} else if order.OrderSide == anteilv1.OrderSide_ORDER_SIDE_BUY {
		// Buy side: lock WRT via bank module (send to module escrow account)
		if k.bankKeeper != nil {
			buyerAddr, addrErr := sdk.AccAddressFromBech32(order.Owner)
			if addrErr != nil {
				return fmt.Errorf("invalid buyer address: %w", addrErr)
			}
			price := anteiltypes.ParseUint64(order.Price)
			amount := anteiltypes.ParseUint64(order.AntAmount)
			wrtAmount := price * amount / 1_000_000 // price in micro WRT per micro ANT
			if wrtAmount == 0 {
				wrtAmount = 1
			}
			escrowCoins := sdk.NewCoins(sdk.NewInt64Coin("uwrt", int64(wrtAmount)))
			if sendErr := k.bankKeeper.SendCoinsFromAccountToModule(ctx, buyerAddr, anteiltypes.ModuleName, escrowCoins); sendErr != nil {
				return fmt.Errorf("insufficient WRT for buy order escrow: %w", sendErr)
			}
			order.EscrowedAmount = fmt.Sprintf("%d", wrtAmount)
		}
	}

	store := ctx.KVStore(k.storeKey)
	orderKey := anteiltypes.GetOrderKey(order.GetOrderId())

	if store.Has(orderKey) {
		return anteiltypes.ErrOrderAlreadyExists
	}

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

	// §4.1 — Release escrowed collateral on cancel
	if err := k.releaseEscrow(ctx, order); err != nil {
		return fmt.Errorf("failed to release escrow: %w", err)
	}

	order.Status = anteilv1.OrderStatus_ORDER_STATUS_CANCELLED
	return k.UpdateOrder(ctx, order)
}

// releaseEscrow returns escrowed funds to the order owner on cancellation
func (k Keeper) releaseEscrow(ctx sdk.Context, order *anteilv1.Order) error {
	escrowedAmt := anteiltypes.ParseUint64(order.EscrowedAmount)
	if escrowedAmt == 0 {
		return nil
	}
	if order.OrderSide == anteilv1.OrderSide_ORDER_SIDE_SELL {
		pos, err := k.GetUserPosition(ctx, order.Owner)
		if err != nil || pos == nil {
			return nil
		}
		locked := anteiltypes.ParseUint64(pos.LockedAnt)
		if escrowedAmt > locked {
			escrowedAmt = locked
		}
		pos.LockedAnt = fmt.Sprintf("%d", locked-escrowedAmt)
		pos.AvailableAnt = fmt.Sprintf("%d", anteiltypes.ParseUint64(pos.AvailableAnt)+escrowedAmt)
		// Remove order ID from open orders
		newOpenIds := make([]string, 0, len(pos.OpenOrderIds))
		for _, id := range pos.OpenOrderIds {
			if id != order.OrderId {
				newOpenIds = append(newOpenIds, id)
			}
		}
		pos.OpenOrderIds = newOpenIds
		return k.SetUserPosition(ctx, pos)
	} else if order.OrderSide == anteilv1.OrderSide_ORDER_SIDE_BUY && k.bankKeeper != nil {
		ownerAddr, err := sdk.AccAddressFromBech32(order.Owner)
		if err != nil {
			return nil
		}
		refundCoins := sdk.NewCoins(sdk.NewInt64Coin("uwrt", int64(escrowedAmt)))
		return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, anteiltypes.ModuleName, ownerAddr, refundCoins)
	}
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

	// Price matching: buy price must be >= sell price for trade to execute
	buyPrice, errBuy := math.LegacyNewDecFromStr(buyOrder.Price)
	sellPrice, errSell := math.LegacyNewDecFromStr(sellOrder.Price)
	if errBuy != nil || errSell != nil {
		return anteiltypes.ErrInvalidPrice
	}
	if buyPrice.LT(sellPrice) {
		return anteiltypes.ErrPriceMismatch
	}

	// Execute the trade
	antAmt := anteiltypes.ParseUint64(buyOrder.AntAmount)
	wrtAmount := fmt.Sprintf("%d", antAmt*anteiltypes.ParseUint64(buyOrder.Price))

	trade := &anteilv1.Trade{
		TradeId:     fmt.Sprintf("trade_%s_%s", buyOrderID, sellOrderID),
		BuyOrderId:  buyOrderID,
		SellOrderId: sellOrderID,
		Buyer:       buyOrder.Owner,
		Seller:      sellOrder.Owner,
		Price:       buyOrder.Price,
		AntAmount:   buyOrder.AntAmount,
		WrtAmount:   wrtAmount,
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
		buyerPosition = &anteilv1.UserPosition{
			Owner:        trade.Buyer,
			AntBalance:   "0",
			TotalTrades:  "0",
			LastActivity: timestamppb.New(ctx.BlockTime()),
		}
	}

	buyerTrades := anteiltypes.ParseUint64(buyerPosition.TotalTrades)
	buyerPosition.TotalTrades = fmt.Sprintf("%d", buyerTrades+1)
	buyerPosition.LastActivity = timestamppb.New(ctx.BlockTime())

	if err := k.SetUserPosition(ctx, buyerPosition); err != nil {
		return err
	}

	// Update seller position
	sellerPosition, err := k.GetUserPosition(ctx, trade.Seller)
	if err != nil {
		sellerPosition = &anteilv1.UserPosition{
			Owner:        trade.Seller,
			AntBalance:   "0",
			TotalTrades:  "0",
			LastActivity: timestamppb.New(ctx.BlockTime()),
		}
	}

	sellerTrades := anteiltypes.ParseUint64(sellerPosition.TotalTrades)
	sellerPosition.TotalTrades = fmt.Sprintf("%d", sellerTrades+1)
	sellerPosition.LastActivity = timestamppb.New(ctx.BlockTime())

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


// BeginBlocker distributes ANT to citizens on schedule
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	lastDistributionTime, err := k.GetLastDistributionTime(ctx)
	if err != nil {
		ctx.Logger().Error("Failed to get last distribution time", "error", err)
	} else {
		currentTime := ctx.BlockTime()
		period := params.EpochLength
		if period == 0 {
			period = params.CitizenAntDistributionPeriod
		}

		if lastDistributionTime.IsZero() || currentTime.Sub(lastDistributionTime) >= period {
			// §5.5 — Run epoch emission: burn supplier ANT, compute coefficient, re-emit
			if err := k.ProcessEpochEmission(ctx); err != nil {
				ctx.Logger().Error("Failed to process epoch emission", "error", err)
			} else {
				if err := k.SetLastDistributionTime(ctx, currentTime); err != nil {
					ctx.Logger().Error("Failed to set last distribution time", "error", err)
				}
			}
		}
	}

	return nil
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
	position.LastActivity = timestamppb.New(ctx.BlockTime())

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
		LastActivity: timestamppb.New(ctx.BlockTime()),
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

// ProcessEpochEmission implements §5.5: burn all supplier ANT, compute coefficient, re-emit
func (k Keeper) ProcessEpochEmission(ctx sdk.Context) error {
	if k.identKeeper == nil {
		return nil
	}

	allAccounts, err := k.identKeeper.GetAllVerifiedAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get verified accounts: %w", err)
	}

	params := k.GetParams(ctx)
	store := ctx.KVStore(k.storeKey)
	now := timestamppb.New(ctx.BlockTime())

	// Step 1: Collect all active suppliers and their current ANT balances
	type supplierData struct {
		address string
		balance uint64
	}
	var suppliers []supplierData
	totalSupplierANT := uint64(0)

	for _, account := range allAccounts {
		if account.Role != identv1.Role_ROLE_SUPPLIER || !account.IsActive {
			continue
		}
		positionKey := anteiltypes.GetUserPositionKey(account.Address)
		positionBz := store.Get(positionKey)
		balance := uint64(0)
		if positionBz != nil {
			var position anteilv1.UserPosition
			if err := k.cdc.Unmarshal(positionBz, &position); err == nil {
				balance, _ = strconv.ParseUint(position.AntBalance, 10, 64)
			}
		}
		suppliers = append(suppliers, supplierData{address: account.Address, balance: balance})
		totalSupplierANT += balance
	}

	if len(suppliers) == 0 {
		return nil
	}

	// Step 2: Burn all supplier ANT §5.5
	for _, s := range suppliers {
		positionKey := anteiltypes.GetUserPositionKey(s.address)
		position := anteiltypes.NewUserPosition(s.address, "0", ctx.BlockTime())
		position.AntBalance = "0"
		position.AvailableAnt = "0"
		position.LockedAnt = "0"
		position.LastActivity = now
		positionBz, err := k.cdc.Marshal(position)
		if err != nil {
			continue
		}
		store.Set(positionKey, positionBz)
	}

	// Step 3: Compute epoch coefficient §5.5
	// coefficient = totalBurnedByValidators / totalSupplierANT (from previous epoch)
	// Clamped to [epoch_coefficient_min, epoch_coefficient_max]
	epochState := k.GetEpochState(ctx)
	burnedPrevEpoch, _ := strconv.ParseUint(epochState.TotalBurnedCurrentEpoch, 10, 64)

	coefficient := float64(1.0)
	if totalSupplierANT > 0 && burnedPrevEpoch > 0 {
		coefficient = float64(burnedPrevEpoch) / float64(totalSupplierANT)
	}

	coeffMin := 0.75
	coeffMax := 1.50
	if params.EpochCoefficientMin != "" {
		if parsed, err := strconv.ParseFloat(params.EpochCoefficientMin, 64); err == nil {
			coeffMin = parsed
		}
	}
	if params.EpochCoefficientMax != "" {
		if parsed, err := strconv.ParseFloat(params.EpochCoefficientMax, 64); err == nil {
			coeffMax = parsed
		}
	}
	if coefficient < coeffMin {
		coefficient = coeffMin
	}
	if coefficient > coeffMax {
		coefficient = coeffMax
	}

	// Step 4: Re-emit ANT to suppliers §5.5
	// new emission = totalSupplierANT * coefficient, split equally among active suppliers
	totalEmission := uint64(float64(totalSupplierANT) * coefficient)
	perSupplierEmission := totalEmission / uint64(len(suppliers))

	accumulationLimit := uint64(0)
	if params.SupplierEpochAntLimit != "" {
		accumulationLimit, _ = strconv.ParseUint(params.SupplierEpochAntLimit, 10, 64)
	}

	distributedCount := 0
	for _, s := range suppliers {
		emission := perSupplierEmission
		if accumulationLimit > 0 && emission > accumulationLimit {
			emission = accumulationLimit
		}

		positionKey := anteiltypes.GetUserPositionKey(s.address)
		balanceStr := strconv.FormatUint(emission, 10)
		position := anteiltypes.NewUserPosition(s.address, balanceStr, ctx.BlockTime())
		position.AntBalance = balanceStr
		position.AvailableAnt = balanceStr
		position.LastActivity = now

		positionBz, err := k.cdc.Marshal(position)
		if err != nil {
			continue
		}
		store.Set(positionKey, positionBz)
		distributedCount++
	}

	// Step 5: Update epoch state
	epochNumber := epochState.EpochNumber + 1
	newEpochState := &anteilv1.EpochState{
		EpochNumber:             epochNumber,
		EpochStart:              now,
		TotalBurnedPrevEpoch:    fmt.Sprintf("%d", burnedPrevEpoch),
		TotalBurnedCurrentEpoch: "0",
		EmissionCoefficient:     fmt.Sprintf("%.6f", coefficient),
		TotalEmitted:            fmt.Sprintf("%d", totalEmission),
		ActiveSuppliers:         uint64(len(suppliers)),
	}
	k.SetEpochState(ctx, newEpochState)

	// Emit epoch events §6.3
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			anteiltypes.EventTypeEpochReset,
			sdk.NewAttribute(anteiltypes.AttributeKeyEpochNumber, fmt.Sprintf("%d", epochNumber)),
			sdk.NewAttribute(anteiltypes.AttributeKeyBurnAmount, fmt.Sprintf("%d", totalSupplierANT)),
			sdk.NewAttribute(anteiltypes.AttributeKeyBurnReason, "epoch_supplier_reset"),
			sdk.NewAttribute(anteiltypes.AttributeKeyActiveSuppliers, fmt.Sprintf("%d", len(suppliers))),
		),
	)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			anteiltypes.EventTypeEpochEmission,
			sdk.NewAttribute(anteiltypes.AttributeKeyEpochNumber, fmt.Sprintf("%d", epochNumber)),
			sdk.NewAttribute(anteiltypes.AttributeKeyEpochCoefficient, fmt.Sprintf("%.6f", coefficient)),
			sdk.NewAttribute(anteiltypes.AttributeKeyTotalEmitted, fmt.Sprintf("%d", totalEmission)),
			sdk.NewAttribute(anteiltypes.AttributeKeyBurnedPrevEpoch, fmt.Sprintf("%d", burnedPrevEpoch)),
			sdk.NewAttribute(anteiltypes.AttributeKeyActiveSuppliers, fmt.Sprintf("%d", len(suppliers))),
		),
	)

	ctx.Logger().Info("Epoch emission completed",
		"epoch", epochNumber,
		"coefficient", coefficient,
		"burned", totalSupplierANT,
		"emitted", totalEmission,
		"suppliers", len(suppliers),
	)
	return nil
}

// GetEpochState retrieves the current epoch state
func (k Keeper) GetEpochState(ctx sdk.Context) *anteilv1.EpochState {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(anteiltypes.EpochStateKey)
	if bz == nil {
		return &anteilv1.EpochState{}
	}
	var state anteilv1.EpochState
	if err := k.cdc.Unmarshal(bz, &state); err != nil {
		return &anteilv1.EpochState{}
	}
	return &state
}

// SetEpochState stores the epoch state
func (k Keeper) SetEpochState(ctx sdk.Context, state *anteilv1.EpochState) {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(state)
	if err != nil {
		return
	}
	store.Set(anteiltypes.EpochStateKey, bz)
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
	engine := NewEconomicEngine(&k)

	if err := engine.ProcessOrderMatching(ctx); err != nil {
		ctx.Logger().Error("Failed to process order matching", "error", err)
	}

	if err := engine.ProcessMarketMaking(ctx); err != nil {
		ctx.Logger().Error("Failed to process market making", "error", err)
	}

	return nil
}

