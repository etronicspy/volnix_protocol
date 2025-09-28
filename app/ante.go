package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// MinimalAnteHandler performs a minimal pre-execution validation of the transaction.
// It checks that the transaction exposes signatures and at least one signature is present.
// Cryptographic signature verification and fee/sequence checks are intentionally omitted
// until x/auth and related modules are wired in.
func MinimalAnteHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	if tx == nil {
		return ctx, fmt.Errorf("tx is nil")
	}

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return ctx, fmt.Errorf("tx is not SigVerifiableTx")
	}
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return ctx, fmt.Errorf("failed to fetch signatures: %w", err)
	}
	if len(sigs) == 0 {
		return ctx, fmt.Errorf("no signatures present in tx")
	}

	return ctx, nil
}
