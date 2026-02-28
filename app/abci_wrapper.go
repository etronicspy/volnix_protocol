package app

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
)

var latestHeightPattern = regexp.MustCompile(`latest height:\s*(\d+)`)

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
	if directResp, handled := w.handleDirectWalletQueries(req); handled {
		return directResp, nil
	}

	resp, err := w.VolnixApp.Query(ctx, req)
	if !shouldRetryMissingVersion(resp, err) {
		return resp, err
	}

	// CometBFT/CosmJS can occasionally query a just-committed height that is
	// not yet available in IAVL snapshots. Retry without fixed height first,
	// then one block lower as a final fallback.
	reqLatest := *req
	reqLatest.Height = 0
	retryResp, retryErr := w.VolnixApp.Query(ctx, &reqLatest)
	if retryErr == nil && !isMissingVersionQuery(retryResp) {
		return retryResp, nil
	}

	if prevHeight, ok := fallbackPrevHeight(req, resp, err); ok {
		reqPrev := *req
		reqPrev.Height = prevHeight
		prevResp, prevErr := w.VolnixApp.Query(ctx, &reqPrev)
		if prevErr == nil && !isMissingVersionQuery(prevResp) {
			return prevResp, nil
		}
	}

	return resp, err
}

func (w *ABCIWrapper) Info(ctx context.Context, req *abci.RequestInfo) (*abci.ResponseInfo, error) {
	return w.VolnixApp.Info(req)
}

func isMissingVersionQuery(resp *abci.ResponseQuery) bool {
	if resp == nil {
		return false
	}
	msg := strings.ToLower(resp.Log + " " + resp.Info)
	return strings.Contains(msg, "version does not exist") ||
		strings.Contains(msg, "failed to load state at height")
}

func isMissingVersionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "version does not exist") ||
		strings.Contains(msg, "failed to load state at height")
}

func shouldRetryMissingVersion(resp *abci.ResponseQuery, err error) bool {
	return isMissingVersionQuery(resp) || isMissingVersionError(err)
}

func fallbackPrevHeight(req *abci.RequestQuery, resp *abci.ResponseQuery, err error) (int64, bool) {
	if req != nil && req.Height > 1 {
		return req.Height - 1, true
	}

	msg := ""
	if resp != nil {
		msg = resp.Log + " " + resp.Info
	}
	if err != nil {
		msg = fmt.Sprintf("%s %s", msg, err.Error())
	}

	matches := latestHeightPattern.FindStringSubmatch(strings.ToLower(msg))
	if len(matches) < 2 {
		return 0, false
	}

	latestHeight, convErr := strconv.ParseInt(matches[1], 10, 64)
	if convErr != nil || latestHeight <= 1 {
		return 0, false
	}

	return latestHeight - 1, true
}

func (w *ABCIWrapper) handleDirectWalletQueries(req *abci.RequestQuery) (*abci.ResponseQuery, bool) {
	if req == nil {
		return nil, false
	}

	switch req.Path {
	case "/cosmos.bank.v1beta1.Query/AllBalances":
		var q banktypes.QueryAllBalancesRequest
		if err := proto.Unmarshal(req.Data, &q); err != nil {
			return queryErrorResponse(fmt.Errorf("failed to decode all balances request: %w", err)), true
		}
		addr, err := sdk.AccAddressFromBech32(q.Address)
		if err != nil {
			return queryErrorResponse(fmt.Errorf("invalid address: %w", err)), true
		}

		queryCtx := w.VolnixApp.NewContext(true)
		resp := &banktypes.QueryAllBalancesResponse{
			Balances: w.bankKeeper.GetAllBalances(queryCtx, addr),
		}
		return querySuccessResponse(resp, queryCtx.BlockHeight()), true

	case "/cosmos.bank.v1beta1.Query/Balance":
		var q banktypes.QueryBalanceRequest
		if err := proto.Unmarshal(req.Data, &q); err != nil {
			return queryErrorResponse(fmt.Errorf("failed to decode balance request: %w", err)), true
		}
		addr, err := sdk.AccAddressFromBech32(q.Address)
		if err != nil {
			return queryErrorResponse(fmt.Errorf("invalid address: %w", err)), true
		}

		queryCtx := w.VolnixApp.NewContext(true)
		balance := w.bankKeeper.GetBalance(queryCtx, addr, q.Denom)
		resp := &banktypes.QueryBalanceResponse{Balance: &balance}
		return querySuccessResponse(resp, queryCtx.BlockHeight()), true

	case "/cosmos.auth.v1beta1.Query/Account":
		var q authtypes.QueryAccountRequest
		if err := proto.Unmarshal(req.Data, &q); err != nil {
			return queryErrorResponse(fmt.Errorf("failed to decode account request: %w", err)), true
		}
		addr, err := sdk.AccAddressFromBech32(q.Address)
		if err != nil {
			return queryErrorResponse(fmt.Errorf("invalid address: %w", err)), true
		}

		queryCtx := w.VolnixApp.NewContext(true)
		account := w.authKeeper.GetAccount(queryCtx, addr)
		if account == nil {
			return querySuccessResponse(&authtypes.QueryAccountResponse{}, queryCtx.BlockHeight()), true
		}
		anyAccount, err := codectypes.NewAnyWithValue(account)
		if err != nil {
			return queryErrorResponse(fmt.Errorf("failed to encode account: %w", err)), true
		}

		resp := &authtypes.QueryAccountResponse{Account: anyAccount}
		return querySuccessResponse(resp, queryCtx.BlockHeight()), true
	}

	return nil, false
}

func querySuccessResponse(msg proto.Message, height int64) *abci.ResponseQuery {
	bz, err := proto.Marshal(msg)
	if err != nil {
		return queryErrorResponse(fmt.Errorf("failed to encode query response: %w", err))
	}
	if height <= 0 {
		height = 1
	}
	return &abci.ResponseQuery{
		Code:   0,
		Value:  bz,
		Height: height,
	}
}

func queryErrorResponse(err error) *abci.ResponseQuery {
	return &abci.ResponseQuery{
		Code: 1,
		Log:  err.Error(),
	}
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