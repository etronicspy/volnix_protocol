package types

const (
	// §6.3 — anteil module events
	EventTypeOrderPlaced   = "anteil.order_placed"
	EventTypeOrderMatched  = "anteil.order_matched"
	EventTypeOrderCancelled = "anteil.order_cancelled"
	EventTypeTradeExecuted = "anteil.trade_executed"
	EventTypePositionUpdate = "anteil.position_updated"
	EventTypeEpochReset    = "anteil.epoch_reset"       // §5.5 epoch boundary
	EventTypeEpochEmission = "anteil.epoch_emission"     // §5.5 new emission
	EventTypeAntBurned     = "anteil.ant_burned"         // unified burn event

	// §6.3 mandatory attributes
	AttributeKeyAccount     = "account"
	AttributeKeyBlockHeight = "block_height"
	AttributeKeyTxHash      = "tx_hash"
	AttributeKeyOrderId     = "order_id"
	AttributeKeyOwner       = "owner"
	AttributeKeyOrderType   = "order_type"
	AttributeKeyOrderSide   = "order_side"
	AttributeKeyAmount      = "amount"
	AttributeKeyPrice       = "price"
	AttributeKeyAntBalance  = "ant_balance"
	AttributeKeyLockedAnt   = "locked_ant"
	AttributeKeyAvailableAnt = "available_ant"
	AttributeKeyBurnAmount  = "burn_amount"
	AttributeKeyBurnReason  = "burn_reason"
	AttributeKeyWrtAmount   = "wrt_amount"   // WRT settlement in trade

	// Epoch attributes §5.5
	AttributeKeyEpochNumber     = "epoch_number"
	AttributeKeyEpochCoefficient = "epoch_coefficient"
	AttributeKeyTotalEmitted    = "total_emitted"
	AttributeKeyActiveSuppliers = "active_suppliers"
	AttributeKeyBurnedPrevEpoch = "burned_prev_epoch"
)
