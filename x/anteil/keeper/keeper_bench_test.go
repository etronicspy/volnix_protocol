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
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

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
			Owner:        fmt.Sprintf("cosmos1owner%d", i),
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
			Owner:        "cosmos1test",
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

// BenchmarkPlaceBid benchmarks bid placement with role validation
func BenchmarkPlaceBid(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	// Create mock ident keeper with 100 validators
	accounts := make([]*identv1.VerifiedAccount, 100)
	for i := 0; i < 100; i++ {
		accounts[i] = &identv1.VerifiedAccount{
			Address:      fmt.Sprintf("cosmos1validator%d", i),
			Role:         identv1.Role_ROLE_VALIDATOR,
			IsActive:     true,
			IdentityHash: fmt.Sprintf("hash%d", i),
		}
	}
	mockIdentKeeper := &MockIdentKeeper{accounts: accounts}
	k.SetIdentKeeper(mockIdentKeeper)
	
	// Create auction
	auction := &anteilv1.Auction{
		AuctionId:    "auction-bench",
		BlockHeight:  1000,
		ReservePrice: "10.0",
		AntAmount:    "1000000",
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
	}
	k.CreateAuction(ctx, auction)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validatorAddr := fmt.Sprintf("cosmos1validator%d", i%100)
		k.PlaceBid(ctx, "auction-bench", validatorAddr, "15.0")
	}
}

// BenchmarkDistributeAntToCitizens benchmarks ANT distribution
func BenchmarkDistributeAntToCitizens(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	// Create mock ident keeper with 1000 citizens
	accounts := make([]*identv1.VerifiedAccount, 1000)
	for i := 0; i < 1000; i++ {
		accounts[i] = &identv1.VerifiedAccount{
			Address:      fmt.Sprintf("cosmos1citizen%d", i),
			Role:         identv1.Role_ROLE_CITIZEN,
			IsActive:     true,
			IdentityHash: fmt.Sprintf("hash%d", i),
		}
	}
	mockIdentKeeper := &MockIdentKeeper{accounts: accounts}
	k.SetIdentKeeper(mockIdentKeeper)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k.DistributeAntToCitizens(ctx)
	}
}

// BenchmarkGetUserPosition benchmarks position retrieval
func BenchmarkGetUserPosition(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	// Setup: Create 1000 positions
	for i := 0; i < 1000; i++ {
		position := &anteilv1.UserPosition{
			Owner:      fmt.Sprintf("cosmos1user%d", i),
			AntBalance: "1000000",
		}
		k.SetUserPosition(ctx, position)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k.GetUserPosition(ctx, fmt.Sprintf("cosmos1user%d", i%1000))
	}
}

// Note: MockIdentKeeper is defined in keeper_test.go
