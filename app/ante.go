package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

const (
	maxMemoLength = 256
	maxMsgCount   = 16
)

// ImprovedAnteHandler provides enhanced transaction validation
// with timeout height, memo, and signature presence checks.
func ImprovedAnteHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	if tx == nil {
		return ctx, fmt.Errorf("transaction cannot be nil")
	}

	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return ctx, fmt.Errorf("transaction must contain at least one message")
	}
	if len(msgs) > maxMsgCount {
		return ctx, fmt.Errorf("transaction contains too many messages: %d (max %d)", len(msgs), maxMsgCount)
	}

	for i, msg := range msgs {
		if msg == nil {
			return ctx, fmt.Errorf("message %d cannot be nil", i)
		}

		if validator, ok := msg.(interface{ ValidateBasic() error }); ok {
			if err := validator.ValidateBasic(); err != nil {
				return ctx, fmt.Errorf("message %d validation failed: %w", i, err)
			}
		}
	}

	// Timeout height validation
	if txWithTimeout, ok := tx.(interface{ GetTimeoutHeight() uint64 }); ok {
		timeoutHeight := txWithTimeout.GetTimeoutHeight()
		if timeoutHeight > 0 {
			blockHeight := uint64(ctx.BlockHeight())
			if blockHeight > timeoutHeight {
				return ctx, fmt.Errorf(
					"transaction timeout: block height %d exceeds timeout height %d",
					blockHeight, timeoutHeight,
				)
			}
		}
	}

	// Memo length validation
	if txWithMemo, ok := tx.(interface{ GetMemo() string }); ok {
		memo := txWithMemo.GetMemo()
		if len(memo) > maxMemoLength {
			return ctx, fmt.Errorf(
				"memo too long: %d characters (max %d)",
				len(memo), maxMemoLength,
			)
		}
	}

	// Signature presence check
	if sigTx, ok := tx.(interface {
		GetSignaturesV2() ([]signing.SignatureV2, error)
	}); ok {
		sigs, err := sigTx.GetSignaturesV2()
		if err == nil && !simulate {
			if len(sigs) == 0 && ctx.IsCheckTx() {
				return ctx, fmt.Errorf("transaction must have at least one signature")
			}
		}
	}

	// Gas limit validation
	gasLimit := ctx.GasMeter().Limit()
	if gasLimit > 0 {
		gasConsumed := ctx.GasMeter().GasConsumed()
		if gasConsumed > gasLimit {
			return ctx, fmt.Errorf(
				"gas limit exceeded: limit=%d, consumed=%d",
				gasLimit, gasConsumed,
			)
		}
	}

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

// MinimalAnteHandler is kept for backward compatibility.
func MinimalAnteHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	return ImprovedAnteHandler(ctx, tx, simulate)
}
