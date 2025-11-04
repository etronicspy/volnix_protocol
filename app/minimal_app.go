package app

import (
	"context"
	"encoding/json"

	sdklog "cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MinimalVolnixApp is a minimal version for CometBFT testing
type MinimalVolnixApp struct {
	*baseapp.BaseApp
	appCodec codec.Codec
}

// NewMinimalVolnixApp creates a minimal Volnix app for CometBFT testing
func NewMinimalVolnixApp(logger sdklog.Logger, db cosmosdb.DB, traceStore interface{}, encoding EncodingConfig) *MinimalVolnixApp {
	bapp := baseapp.NewBaseApp("volnix", logger, db, encoding.TxConfig.TxDecoder)
	bapp.SetVersion("0.1.0")
	bapp.SetInterfaceRegistry(encoding.InterfaceRegistry)
	bapp.SetTxEncoder(encoding.TxConfig.TxEncoder)

	app := &MinimalVolnixApp{
		BaseApp:  bapp,
		appCodec: encoding.Codec,
	}

	// Set minimal ABCI handlers
	bapp.SetBeginBlocker(func(ctx sdk.Context) (sdk.BeginBlock, error) {
		return sdk.BeginBlock{}, nil
	})

	bapp.SetEndBlocker(func(ctx sdk.Context) (sdk.EndBlock, error) {
		return sdk.EndBlock{}, nil
	})

	bapp.SetInitChainer(func(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
		return &abci.ResponseInitChain{}, nil
	})

	// Set minimal AnteHandler
	bapp.SetAnteHandler(func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})

	return app
}

// ExportAppStateAndValidators exports the state for genesis
func (app *MinimalVolnixApp) ExportAppStateAndValidators(
	forZeroHeight bool, jailAllowedAddrs []string,
) (map[string]json.RawMessage, error) {
	return make(map[string]json.RawMessage), nil
}

// GetBaseApp returns the base application
func (app *MinimalVolnixApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// ModuleAccountAddrs returns module account addresses
func (app *MinimalVolnixApp) ModuleAccountAddrs() map[string]bool {
	return make(map[string]bool)
}

// ABCI methods for CometBFT compatibility

// ApplySnapshotChunk implements the ABCI interface with context
func (app *MinimalVolnixApp) ApplySnapshotChunk(ctx context.Context, req *abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	return &abci.ResponseApplySnapshotChunk{
		Result: abci.ResponseApplySnapshotChunk_ACCEPT,
	}, nil
}

// LoadSnapshotChunk implements the ABCI interface with context
func (app *MinimalVolnixApp) LoadSnapshotChunk(ctx context.Context, req *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	return &abci.ResponseLoadSnapshotChunk{}, nil
}

// ListSnapshots implements the ABCI interface with context
func (app *MinimalVolnixApp) ListSnapshots(ctx context.Context, req *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	return &abci.ResponseListSnapshots{}, nil
}

// OfferSnapshot implements the ABCI interface with context
func (app *MinimalVolnixApp) OfferSnapshot(ctx context.Context, req *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	return &abci.ResponseOfferSnapshot{
		Result: abci.ResponseOfferSnapshot_REJECT,
	}, nil
}
