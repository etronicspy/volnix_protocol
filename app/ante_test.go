package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
)

type mockTx struct {
	msgs []sdk.Msg
}

func (m mockTx) GetMsgs() []sdk.Msg { return m.msgs }
func (m mockTx) GetMsgsV2() ([]proto.Message, error) {
	out := make([]proto.Message, 0, len(m.msgs))
	for _, msg := range m.msgs {
		if msg == nil {
			continue
		}
		out = append(out, msg.(proto.Message))
	}
	return out, nil
}
func (mockTx) ValidateBasic() error { return nil }

// msgWithFailingValidateBasic implements sdk.Msg and ValidateBasic that returns error
type msgWithFailingValidateBasic struct {
	*governancev1.MsgSubmitProposal
}

func (msgWithFailingValidateBasic) ValidateBasic() error {
	return fmt.Errorf("validation failed")
}

func TestImprovedAnteHandler_NilTx(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("t"))
	_, err := ImprovedAnteHandler(ctx, nil, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil")
}

func TestImprovedAnteHandler_EmptyMsgs(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("t"))
	tx := mockTx{msgs: nil}
	_, err := ImprovedAnteHandler(ctx, tx, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one message")
}

func TestImprovedAnteHandler_ValidTx(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("t"))
	tx := mockTx{msgs: []sdk.Msg{&governancev1.MsgSubmitProposal{}}}
	_, err := ImprovedAnteHandler(ctx, tx, false)
	require.NoError(t, err)
}

func TestMinimalAnteHandler(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("t"))
	tx := mockTx{msgs: []sdk.Msg{&governancev1.MsgSubmitProposal{}}}
	_, err := MinimalAnteHandler(ctx, tx, false)
	require.NoError(t, err)
}

func TestImprovedAnteHandler_NilMessage(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("t"))
	tx := mockTx{msgs: []sdk.Msg{nil}}
	_, err := ImprovedAnteHandler(ctx, tx, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "message 0 cannot be nil")
}

func TestImprovedAnteHandler_ValidateBasicFails(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("t"))
	tx := mockTx{msgs: []sdk.Msg{&msgWithFailingValidateBasic{MsgSubmitProposal: &governancev1.MsgSubmitProposal{}}}}
	_, err := ImprovedAnteHandler(ctx, tx, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "validation failed")
}

// mockGasMeter implements storetypes.GasMeter with configurable limit and consumed (for testing gas exceeded path)
type mockGasMeter struct {
	limit   uint64
	consumed uint64
}

func (m *mockGasMeter) GasConsumed() uint64      { return m.consumed }
func (m *mockGasMeter) GasConsumedToLimit() uint64 {
	if m.consumed > m.limit {
		return m.limit
	}
	return m.consumed
}
func (m *mockGasMeter) GasRemaining() uint64 {
	if m.consumed >= m.limit {
		return 0
	}
	return m.limit - m.consumed
}
func (m *mockGasMeter) Limit() uint64           { return m.limit }
func (m *mockGasMeter) ConsumeGas(amount uint64, descriptor string) {}
func (m *mockGasMeter) RefundGas(amount uint64, descriptor string) {}
func (m *mockGasMeter) IsPastLimit() bool       { return m.consumed > m.limit }
func (m *mockGasMeter) IsOutOfGas() bool        { return m.consumed >= m.limit }
func (m *mockGasMeter) String() string         { return "mockGasMeter" }

func TestImprovedAnteHandler_GasExceeded(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("t"))
	gasMeter := &mockGasMeter{limit: 100, consumed: 150}
	ctx = ctx.WithGasMeter(gasMeter)
	tx := mockTx{msgs: []sdk.Msg{&governancev1.MsgSubmitProposal{}}}
	_, err := ImprovedAnteHandler(ctx, tx, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "gas limit exceeded")
}
