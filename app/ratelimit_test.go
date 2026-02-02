package app

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
)

// rateLimitMockTx implements sdk.Tx for ratelimit tests (no signers on messages)
type rateLimitMockTx struct {
	msgs []sdk.Msg
}

func (m rateLimitMockTx) GetMsgs() []sdk.Msg { return m.msgs }
func (m rateLimitMockTx) GetMsgsV2() ([]proto.Message, error) {
	out := make([]proto.Message, 0, len(m.msgs))
	for _, msg := range m.msgs {
		if msg != nil {
			out = append(out, msg.(proto.Message))
		}
	}
	return out, nil
}
func (rateLimitMockTx) ValidateBasic() error { return nil }

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	require.True(t, cfg.Enabled)
	require.Equal(t, 1000.0, cfg.GlobalRate)
	require.Equal(t, 10.0, cfg.PerAddrRate)
	require.Equal(t, 20, cfg.BurstSize)
}

func TestNewRateLimiter_Disabled(t *testing.T) {
	cfg := RateLimitConfig{Enabled: false}
	rl := NewRateLimiter(cfg)
	require.Nil(t, rl)
}

func TestNewRateLimiter_Enabled(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	rl := NewRateLimiter(cfg)
	require.NotNil(t, rl)
}

func TestRateLimiter_Allow_NilLimiter(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("t"))
	tx := rateLimitMockTx{msgs: []sdk.Msg{&governancev1.MsgSubmitProposal{}}}
	err := (*RateLimiter)(nil).Allow(ctx, tx)
	require.NoError(t, err)
}

func TestRateLimiter_Allow_GlobalOnly(t *testing.T) {
	cfg := RateLimitConfig{GlobalRate: 1, PerAddrRate: 10, BurstSize: 1, Enabled: true}
	rl := NewRateLimiter(cfg)
	require.NotNil(t, rl)
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("t"))
	tx := rateLimitMockTx{msgs: []sdk.Msg{&governancev1.MsgSubmitProposal{}}}

	err := rl.Allow(ctx, tx)
	require.NoError(t, err)
	err = rl.Allow(ctx, tx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "global rate limit exceeded")
}

func TestRateLimiter_GetStats_Nil(t *testing.T) {
	stats := (*RateLimiter)(nil).GetStats()
	require.False(t, stats["enabled"].(bool))
}

func TestRateLimiter_GetStats_Enabled(t *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())
	require.NotNil(t, rl)
	stats := rl.GetStats()
	require.True(t, stats["enabled"].(bool))
	require.Equal(t, float64(1000), stats["global_rate"])
	require.Equal(t, float64(10), stats["per_addr_rate"])
	require.Equal(t, 20, stats["burst_size"])
}

func TestRateLimiter_Cleanup_Nil(t *testing.T) {
	(*RateLimiter)(nil).Cleanup(0)
}

func TestRateLimiter_Cleanup_Enabled(t *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())
	rl.Cleanup(0)
}
