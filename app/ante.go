package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

const (
	maxMemoLength = 256
	maxMsgCount   = 16
	antDenom      = "uant"
)

// ImprovedAnteHandler provides enhanced transaction validation
// with timeout height, memo, and signature presence checks.
// §4.1: rejects any bank MsgSend / MsgMultiSend that transfers ANT.
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

		// §4.1 — ANT direct transfer ban: reject MsgSend/MsgMultiSend containing uant
		if err := rejectANTTransfer(msg); err != nil {
			return ctx, fmt.Errorf("message %d rejected: %w", i, err)
		}

		if validator, ok := msg.(interface{ ValidateBasic() error }); ok {
			if err := validator.ValidateBasic(); err != nil {
				return ctx, fmt.Errorf("message %d validation failed: %w", i, err)
			}
		}
	}

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

	if txWithMemo, ok := tx.(interface{ GetMemo() string }); ok {
		memo := txWithMemo.GetMemo()
		if len(memo) > maxMemoLength {
			return ctx, fmt.Errorf(
				"memo too long: %d characters (max %d)",
				len(memo), maxMemoLength,
			)
		}
	}

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

// rejectANTTransfer blocks bank MsgSend and MsgMultiSend if they contain ANT (uant).
// Per §4.1: ANT changes owner only via internal market order execution and
// protocol service movements (emission, epoch reset, burn, migration).
func rejectANTTransfer(msg sdk.Msg) error {
	switch m := msg.(type) {
	case *banktypes.MsgSend:
		for _, coin := range m.Amount {
			if coin.Denom == antDenom {
				return fmt.Errorf("direct ANT transfers are prohibited (§4.1): use the internal market")
			}
		}
	case *banktypes.MsgMultiSend:
		for _, input := range m.Inputs {
			for _, coin := range input.Coins {
				if coin.Denom == antDenom {
					return fmt.Errorf("direct ANT transfers are prohibited (§4.1): use the internal market")
				}
			}
		}
	}
	return nil
}

// MinimalAnteHandler is kept for backward compatibility.
func MinimalAnteHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	return ImprovedAnteHandler(ctx, tx, simulate)
}
