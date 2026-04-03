package keeper_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

func init() {
	cfg := sdk.GetConfig()
	if cfg.GetBech32AccountAddrPrefix() != "cosmos" {
		cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
	}
}

// setupBenchmark creates a keeper for benchmarking
func setupBenchmark(b *testing.B) (*keeper.Keeper, sdk.Context) {
	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create store keys
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tKey := storetypes.NewTransientStoreKey("transient_test")

	// Create test context
	ctx := testutil.DefaultContext(storeKey, tKey)

	// Create params keeper and subspace
	paramsKeeper := paramskeeper.NewKeeper(cdc, codec.NewLegacyAmino(), storeKey, tKey)
	paramStore := paramsKeeper.Subspace(types.ModuleName)
	paramStore = paramStore.WithKeyTable(types.ParamKeyTable())

	// Create keeper
	k := keeper.NewKeeper(cdc, storeKey, paramStore)
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}

// BenchmarkCreateOrder benchmarks order creation
func BenchmarkCreateOrder(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		order := &anteilv1.Order{
			OrderId:      fmt.Sprintf("order-%d", i),
			Owner:        mustAddr(fmt.Sprintf("000000000000000000000000000000000000%04x", i%0x10000)),
			OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
			OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
			AntAmount:    "1000000",
			Price:        "1.5",
			Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
			IdentityHash: fmt.Sprintf("hash%d", i),
		}
		k.SetOrder(ctx, order)
	}
}

// BenchmarkGetOrder benchmarks order retrieval
func BenchmarkGetOrder(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	// Setup: Create 1000 orders
	for i := 0; i < 1000; i++ {
		order := &anteilv1.Order{
			OrderId:      fmt.Sprintf("order-%d", i),
			Owner:        addrTest,
			OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
			OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
			AntAmount:    "1000000",
			Price:        "1.5",
			Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
			IdentityHash: "hash",
		}
		k.SetOrder(ctx, order)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k.GetOrder(ctx, fmt.Sprintf("order-%d", i%1000))
	}
}

// BenchmarkGetUserPosition benchmarks position retrieval
func BenchmarkGetUserPosition(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	// Setup: Create 1000 positions (valid addresses)
	userAddrs := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		userAddrs[i] = mustAddr(fmt.Sprintf("0000000000000000000000000000000000%06x", i+0x2000))
		position := &anteilv1.UserPosition{Owner: userAddrs[i], AntBalance: "1000000"}
		k.SetUserPosition(ctx, position)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k.GetUserPosition(ctx, userAddrs[i%1000])
	}
}
