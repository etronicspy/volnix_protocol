package app

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"
)

// DemoWalletAddress is derived from the well-known test mnemonic
// "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
// with HD path m/44'/118'/0'/0/0 and bech32 prefix "volnix".
const DemoWalletAddress = "volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces"

// ABCIWrapper bridges CometBFT v0.38 context-aware ABCI interface
// to Cosmos SDK BaseApp methods. All state (balances, accounts, modules)
// is managed by the real VolnixApp through proper Cosmos SDK modules.
type ABCIWrapper struct {
	*VolnixApp
}

// NewABCIWrapper creates a new ABCI wrapper around the full Volnix app.
func NewABCIWrapper(app *VolnixApp) *ABCIWrapper {
	return &ABCIWrapper{VolnixApp: app}
}

func (w *ABCIWrapper) CheckTx(ctx context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	return w.VolnixApp.CheckTx(req)
}

func (w *ABCIWrapper) FinalizeBlock(ctx context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	return w.VolnixApp.FinalizeBlock(req)
}

func (w *ABCIWrapper) Commit(ctx context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	resp, err := w.VolnixApp.Commit()
	return resp, err
}

func (w *ABCIWrapper) Query(ctx context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	return w.VolnixApp.Query(ctx, req)
}

func (w *ABCIWrapper) Info(ctx context.Context, req *abci.RequestInfo) (*abci.ResponseInfo, error) {
	return w.VolnixApp.Info(req)
}

func (w *ABCIWrapper) InitChain(ctx context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	return w.VolnixApp.InitChain(req)
}

func (w *ABCIWrapper) PrepareProposal(ctx context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	return w.VolnixApp.PrepareProposal(req)
}

func (w *ABCIWrapper) ProcessProposal(ctx context.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	return w.VolnixApp.ProcessProposal(req)
}

func (w *ABCIWrapper) ExtendVote(ctx context.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	return w.VolnixApp.ExtendVote(ctx, req)
}

func (w *ABCIWrapper) VerifyVoteExtension(ctx context.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	return w.VolnixApp.VerifyVoteExtension(req)
}