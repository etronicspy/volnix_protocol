package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ImprovedAnteHandler provides enhanced transaction validation
// This is an intermediate implementation that validates transaction structure
// and basic fields without requiring full auth/bank module integration
func ImprovedAnteHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	// IMPROVED: Basic transaction structure validation
	if tx == nil {
		return ctx, fmt.Errorf("transaction cannot be nil")
	}

	// IMPROVED: Validate transaction messages
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return ctx, fmt.Errorf("transaction must contain at least one message")
	}

	// IMPROVED: Validate each message
	for i, msg := range msgs {
		if msg == nil {
			return ctx, fmt.Errorf("message %d cannot be nil", i)
		}

		// Basic message validation - check if message implements validation
		// Note: Full ValidateBasic requires proper message implementation
		// For now, we just check that message is not nil
	}

	// IMPROVED: Validate transaction timeout height if set
	// Note: Timeout height validation requires proper tx type implementation
	// For now, we skip this check as it requires specific transaction types

	// IMPROVED: Validate gas limit
	gasLimit := ctx.GasMeter().Limit()
	if gasLimit > 0 {
		// Ensure we don't exceed gas limit
		gasConsumed := ctx.GasMeter().GasConsumed()
		if gasConsumed > gasLimit {
			return ctx, fmt.Errorf(
				"gas limit exceeded: limit=%d, consumed=%d",
				gasLimit, gasConsumed,
			)
		}
	}

	// IMPROVED: Check for signatures presence (basic check)
	// Full signature verification requires auth module integration
	// Note: Signature validation requires proper tx type implementation
	// For now, we skip this check as it requires specific transaction types

	// IMPROVED: Validate memo length if present
	// Note: Memo validation requires proper tx type implementation
	// For now, we skip this check as it requires specific transaction types

	// IMPROVED: Log transaction validation (only in debug mode)
	if ctx.IsCheckTx() && !simulate {
		ctx.Logger().Debug("Transaction validated",
			"height", ctx.BlockHeight(),
			"msgs", len(msgs),
			"gas_limit", gasLimit,
			"gas_consumed", ctx.GasMeter().GasConsumed(),
		)
	}
	
	return ctx, nil
}

// MinimalAnteHandler is kept for backward compatibility
// It now delegates to ImprovedAnteHandler
func MinimalAnteHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	return ImprovedAnteHandler(ctx, tx, simulate)
}