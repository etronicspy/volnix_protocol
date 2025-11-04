package app

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"
)

// ABCIWrapper wraps VolnixApp to provide context-aware ABCI methods
type ABCIWrapper struct {
	*MinimalVolnixApp
}

// NewABCIWrapper creates a new ABCI wrapper
func NewABCIWrapper(app *MinimalVolnixApp) *ABCIWrapper {
	return &ABCIWrapper{MinimalVolnixApp: app}
}

// CheckTx implements ABCI interface with context
func (w *ABCIWrapper) CheckTx(ctx context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	resp, err := w.MinimalVolnixApp.CheckTx(req)
	return resp, err
}

// FinalizeBlock implements ABCI interface with context
func (w *ABCIWrapper) FinalizeBlock(ctx context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	resp, err := w.MinimalVolnixApp.FinalizeBlock(req)
	return resp, err
}

// Commit implements ABCI interface with context
func (w *ABCIWrapper) Commit(ctx context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	resp, err := w.MinimalVolnixApp.Commit()
	return resp, err
}

// Query implements ABCI interface with context
func (w *ABCIWrapper) Query(ctx context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	resp, err := w.MinimalVolnixApp.Query(ctx, req)
	return resp, err
}

// Info implements ABCI interface with context
func (w *ABCIWrapper) Info(ctx context.Context, req *abci.RequestInfo) (*abci.ResponseInfo, error) {
	resp, err := w.MinimalVolnixApp.Info(req)
	return resp, err
}

// InitChain implements ABCI interface with context
func (w *ABCIWrapper) InitChain(ctx context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	return w.MinimalVolnixApp.InitChain(req)
}

// PrepareProposal implements ABCI interface with context
func (w *ABCIWrapper) PrepareProposal(ctx context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	resp, err := w.MinimalVolnixApp.PrepareProposal(req)
	return resp, err
}

// ProcessProposal implements ABCI interface with context
func (w *ABCIWrapper) ProcessProposal(ctx context.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	resp, err := w.MinimalVolnixApp.ProcessProposal(req)
	return resp, err
}

// ExtendVote implements ABCI interface with context
func (w *ABCIWrapper) ExtendVote(ctx context.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	resp, err := w.MinimalVolnixApp.ExtendVote(ctx, req)
	return resp, err
}

// VerifyVoteExtension implements ABCI interface with context
func (w *ABCIWrapper) VerifyVoteExtension(ctx context.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	resp, err := w.MinimalVolnixApp.VerifyVoteExtension(req)
	return resp, err
}