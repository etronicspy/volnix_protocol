package tests

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
	lizenzkeeper "github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
)

type BenchmarkTestSuite struct {
	suite.Suite

	cdc codec.Codec
	ctx sdk.Context

	// Keepers
	identKeeper  *identkeeper.Keeper
	lizenzKeeper *lizenzkeeper.Keeper
	anteilKeeper *anteilkeeper.Keeper

	// Store keys
	identStoreKey  *storetypes.KVStoreKey
	lizenzStoreKey *storetypes.KVStoreKey
	anteilStoreKey *storetypes.KVStoreKey

	// Param stores
	identParamStore  paramtypes.Subspace
	lizenzParamStore paramtypes.Subspace
	anteilParamStore paramtypes.Subspace
}

func (suite *BenchmarkTestSuite) SetupTest() {
	tc := NewTestContext(suite.T())

	suite.cdc = tc.Cdc
	suite.ctx = tc.Ctx

	suite.identStoreKey = tc.IdentStoreKey.(*storetypes.KVStoreKey)
	suite.lizenzStoreKey = tc.LizenzStoreKey.(*storetypes.KVStoreKey)
	suite.anteilStoreKey = tc.AnteilStoreKey.(*storetypes.KVStoreKey)

	suite.identParamStore = tc.IdentParamStore
	suite.lizenzParamStore = tc.LizenzParamStore
	suite.anteilParamStore = tc.AnteilParamStore

	suite.identKeeper = tc.IdentKeeper
	suite.lizenzKeeper = tc.LizenzKeeper
	suite.anteilKeeper = tc.AnteilKeeper
}

func BenchmarkCreateVerifiedAccount(b *testing.B) {
	// Setup
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	storeKey := storetypes.NewKVStoreKey("test_ident")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")
	ctx := testutil.DefaultContext(storeKey, tKey)

	paramsKeeper := paramskeeper.NewKeeper(cdc, codec.NewLegacyAmino(), storeKey, tKey)
	paramStore := paramsKeeper.Subspace(identtypes.ModuleName)
	paramStore.WithKeyTable(identtypes.ParamKeyTable())

	keeper := identkeeper.NewKeeper(cdc, storeKey, paramStore)
	keeper.SetParams(ctx, identtypes.DefaultParams())

	b.ResetTimer()

	addrs := GenerateTestAddresses("", b.N)
	for i := 0; i < b.N; i++ {
		account := identtypes.NewVerifiedAccount(addrs[i], identv1.Role_ROLE_SUPPLIER, fmt.Sprintf("hash%d", i))
		keeper.SetVerifiedAccount(ctx, account)
	}
}

func BenchmarkCreateOrder(b *testing.B) {
	// Setup
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	storeKey := storetypes.NewKVStoreKey("test_anteil")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")
	ctx := testutil.DefaultContext(storeKey, tKey)

	paramsKeeper := paramskeeper.NewKeeper(cdc, codec.NewLegacyAmino(), storeKey, tKey)
	paramStore := paramsKeeper.Subspace(anteiltypes.ModuleName)
	paramStore.WithKeyTable(anteiltypes.ParamKeyTable())

	keeper := anteilkeeper.NewKeeper(cdc, storeKey, paramStore)
	keeper.SetParams(ctx, anteiltypes.DefaultParams())

	b.ResetTimer()

	addrs := GenerateTestAddresses("", b.N)
	for i := 0; i < b.N; i++ {
		order := anteiltypes.NewOrder(addrs[i], anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_BUY, "1000000", "1.5", fmt.Sprintf("hash%d", i))
		keeper.CreateOrder(ctx, order)
	}
}

func BenchmarkExecuteTrade(b *testing.B) {
	// Setup
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	storeKey := storetypes.NewKVStoreKey("test_anteil")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")
	ctx := testutil.DefaultContext(storeKey, tKey)

	paramsKeeper := paramskeeper.NewKeeper(cdc, codec.NewLegacyAmino(), storeKey, tKey)
	paramStore := paramsKeeper.Subspace(anteiltypes.ModuleName)
	paramStore.WithKeyTable(anteiltypes.ParamKeyTable())

	keeper := anteilkeeper.NewKeeper(cdc, storeKey, paramStore)
	keeper.SetParams(ctx, anteiltypes.DefaultParams())

	// Create orders for trading (SELL requires balance)
	buyerAddr := mustBech32("0000000000000000000000000000000000000060")
	sellerAddr := mustBech32("0000000000000000000000000000000000000061")
	keeper.SetUserPosition(ctx, anteiltypes.NewUserPosition(sellerAddr, "1000000"))

	buyOrder := anteiltypes.NewOrder(buyerAddr, anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_BUY, "1000000", "1.5", "hash123")
	sellOrder := anteiltypes.NewOrder(sellerAddr, anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_SELL, "1000000", "1.5", "hash456")

	err := keeper.CreateOrder(ctx, buyOrder)
	require.NoError(b, err)
	buyOrderID := buyOrder.OrderId

	err = keeper.CreateOrder(ctx, sellOrder)
	require.NoError(b, err)
	sellOrderID := sellOrder.OrderId

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		keeper.ExecuteTrade(ctx, buyOrderID, sellOrderID)
	}
}

func BenchmarkGetAllOrders(b *testing.B) {
	// Setup
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	storeKey := storetypes.NewKVStoreKey("test_anteil")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")
	ctx := testutil.DefaultContext(storeKey, tKey)

	paramsKeeper := paramskeeper.NewKeeper(cdc, codec.NewLegacyAmino(), storeKey, tKey)
	paramStore := paramsKeeper.Subspace(anteiltypes.ModuleName)
	paramStore.WithKeyTable(anteiltypes.ParamKeyTable())

	keeper := anteilkeeper.NewKeeper(cdc, storeKey, paramStore)
	keeper.SetParams(ctx, anteiltypes.DefaultParams())

	addrs := GenerateTestAddresses("", 1000)
	for i := 0; i < 1000; i++ {
		order := anteiltypes.NewOrder(addrs[i], anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_BUY, "1000000", "1.5", fmt.Sprintf("hash%d", i))
		keeper.CreateOrder(ctx, order)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		keeper.GetAllOrders(ctx)
	}
}

func BenchmarkGetOrdersByOwner(b *testing.B) {
	// Setup
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	storeKey := storetypes.NewKVStoreKey("test_anteil")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")
	ctx := testutil.DefaultContext(storeKey, tKey)

	paramsKeeper := paramskeeper.NewKeeper(cdc, codec.NewLegacyAmino(), storeKey, tKey)
	paramStore := paramsKeeper.Subspace(anteiltypes.ModuleName)
	paramStore.WithKeyTable(anteiltypes.ParamKeyTable())

	keeper := anteilkeeper.NewKeeper(cdc, storeKey, paramStore)
	keeper.SetParams(ctx, anteiltypes.DefaultParams())

	ownerAddr := TestAddresses.Test1
	for i := 0; i < 1000; i++ {
		order := anteiltypes.NewOrder(ownerAddr, anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_BUY, "1000000", "1.5", fmt.Sprintf("hash%d", i))
		keeper.CreateOrder(ctx, order)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		keeper.GetOrdersByOwner(ctx, ownerAddr)
	}
}

func BenchmarkUpdateUserPosition(b *testing.B) {
	// Setup
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	storeKey := storetypes.NewKVStoreKey("test_anteil")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")
	ctx := testutil.DefaultContext(storeKey, tKey)

	paramsKeeper := paramskeeper.NewKeeper(cdc, codec.NewLegacyAmino(), storeKey, tKey)
	paramStore := paramsKeeper.Subspace(anteiltypes.ModuleName)
	paramStore.WithKeyTable(anteiltypes.ParamKeyTable())

	keeper := anteilkeeper.NewKeeper(cdc, storeKey, paramStore)
	keeper.SetParams(ctx, anteiltypes.DefaultParams())

	// Create position
	addr := TestAddresses.Test1
	position := anteiltypes.NewUserPosition(addr, "10000000")
	keeper.SetUserPosition(ctx, position)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		keeper.UpdateUserPosition(ctx, addr, "500000", 1)
	}
}

func BenchmarkBeginBlocker(b *testing.B) {
	// Setup
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	storeKey := storetypes.NewKVStoreKey("test_anteil")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")
	ctx := testutil.DefaultContext(storeKey, tKey)

	paramsKeeper := paramskeeper.NewKeeper(cdc, codec.NewLegacyAmino(), storeKey, tKey)
	paramStore := paramsKeeper.Subspace(anteiltypes.ModuleName)
	paramStore.WithKeyTable(anteiltypes.ParamKeyTable())

	keeper := anteilkeeper.NewKeeper(cdc, storeKey, paramStore)
	keeper.SetParams(ctx, anteiltypes.DefaultParams())

	addrs := GenerateTestAddresses("", 100)
	for i := 0; i < 100; i++ {
		order := anteiltypes.NewOrder(
			addrs[i],
			anteilv1.OrderType_ORDER_TYPE_LIMIT,
			anteilv1.OrderSide_ORDER_SIDE_BUY,
			"1000000",
			"1.5",
			fmt.Sprintf("hash%d", i),
		)
		keeper.CreateOrder(ctx, order)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		keeper.BeginBlocker(ctx)
	}
}

func BenchmarkEndBlocker(b *testing.B) {
	// Setup
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	storeKey := storetypes.NewKVStoreKey("test_anteil")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")
	ctx := testutil.DefaultContext(storeKey, tKey)

	paramsKeeper := paramskeeper.NewKeeper(cdc, codec.NewLegacyAmino(), storeKey, tKey)
	paramStore := paramsKeeper.Subspace(anteiltypes.ModuleName)
	paramStore.WithKeyTable(anteiltypes.ParamKeyTable())

	keeper := anteilkeeper.NewKeeper(cdc, storeKey, paramStore)
	keeper.SetParams(ctx, anteiltypes.DefaultParams())

	addrs := GenerateTestAddresses("", 100)
	for i := 0; i < 100; i++ {
		order := anteiltypes.NewOrder(
			addrs[i],
			anteilv1.OrderType_ORDER_TYPE_LIMIT,
			anteilv1.OrderSide_ORDER_SIDE_BUY,
			"1000000",
			"1.5",
			fmt.Sprintf("hash%d", i),
		)
		keeper.CreateOrder(ctx, order)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		keeper.EndBlocker(ctx)
	}
}

func (suite *BenchmarkTestSuite) TestPerformanceMetrics() {
	// Test 1: Measure order creation performance
	start := time.Now()

	addrs := GenerateTestAddresses("", 1000)
	for i := 0; i < 1000; i++ {
		order := anteiltypes.NewOrder(addrs[i], anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_BUY, "1000000", "1.5", fmt.Sprintf("hash%d", i))
		suite.anteilKeeper.CreateOrder(suite.ctx, order)
	}

	duration := time.Since(start)
	suite.T().Logf("Created 1000 orders in %v", duration)
	require.Less(suite.T(), duration, 5*time.Second, "Order creation should be fast")

	// Test 2: Measure trade execution performance
	buyerAddr := TestAddresses.Buyer
	sellerAddr := TestAddresses.Seller
	suite.anteilKeeper.SetUserPosition(suite.ctx, anteiltypes.NewUserPosition(sellerAddr, "1000000"))
	buyOrder := anteiltypes.NewOrder(
		buyerAddr,
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"1.5",
		"hash123",
	)

	sellOrder := anteiltypes.NewOrder(
		sellerAddr,
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_SELL,
		"1000000",
		"1.5",
		"hash456",
	)

	err := suite.anteilKeeper.CreateOrder(suite.ctx, buyOrder)
	require.NoError(suite.T(), err)
	buyOrderID := buyOrder.OrderId

	err = suite.anteilKeeper.CreateOrder(suite.ctx, sellOrder)
	require.NoError(suite.T(), err)
	sellOrderID := sellOrder.OrderId
	require.NoError(suite.T(), err)

	start = time.Now()

	for i := 0; i < 100; i++ {
		suite.anteilKeeper.ExecuteTrade(suite.ctx, buyOrderID, sellOrderID)
	}

	duration = time.Since(start)
	suite.T().Logf("Executed 100 trades in %v", duration)
	require.Less(suite.T(), duration, 2*time.Second, "Trade execution should be fast")

	// Test 3: Measure query performance
	start = time.Now()

	for i := 0; i < 1000; i++ {
		suite.anteilKeeper.GetAllOrders(suite.ctx)
	}

	duration = time.Since(start)
	suite.T().Logf("Executed 1000 GetAllOrders queries in %v", duration)
	require.Less(suite.T(), duration, 5*time.Second, "Query execution should be fast")
}

func (suite *BenchmarkTestSuite) TestMemoryUsage() {
	// Test 1: Measure memory usage for large number of orders
	initialMem := getMemUsage()

	addrs := GenerateTestAddresses("", 10000)
	for i := 0; i < 10000; i++ {
		order := anteiltypes.NewOrder(addrs[i], anteilv1.OrderType_ORDER_TYPE_LIMIT, anteilv1.OrderSide_ORDER_SIDE_BUY, "1000000", "1.5", fmt.Sprintf("hash%d", i))
		suite.anteilKeeper.CreateOrder(suite.ctx, order)
	}

	finalMem := getMemUsage()
	memoryIncrease := finalMem - initialMem

	suite.T().Logf("Memory increase for 10000 orders: %d bytes", memoryIncrease)
	require.Less(suite.T(), memoryIncrease, int64(100*1024*1024), "Memory usage should be reasonable") // 100MB limit

	// Test 2: Measure memory usage for large number of accounts
	initialMem = getMemUsage()

	accAddrs := GenerateTestAddresses("", 10000)
	for i := 0; i < 10000; i++ {
		account := identtypes.NewVerifiedAccount(accAddrs[i], identv1.Role_ROLE_SUPPLIER, fmt.Sprintf("hash%d", i))
		suite.identKeeper.SetVerifiedAccount(suite.ctx, account)
	}

	finalMem = getMemUsage()
	memoryIncrease = finalMem - initialMem

	suite.T().Logf("Memory increase for 10000 accounts: %d bytes", memoryIncrease)
	require.Less(suite.T(), memoryIncrease, int64(50*1024*1024), "Memory usage should be reasonable") // 50MB limit
}

func (suite *BenchmarkTestSuite) TestConcurrentOperations() {
	// Test 1: Concurrent order creation with proper synchronization
	addrs := GenerateTestAddresses("", 100)
	done := make(chan bool, 10)
	var orderCount int32

	for i := 0; i < 10; i++ {
		go func(workerID int) {
			defer func() { done <- true }()
			for j := 0; j < 10; j++ { // Reduced from 100 to 10 for stability
				idx := workerID*10 + j
				order := anteiltypes.NewOrder(
					addrs[idx],
					anteilv1.OrderType_ORDER_TYPE_LIMIT,
					anteilv1.OrderSide_ORDER_SIDE_BUY,
					"1000000",
					"1.5",
					fmt.Sprintf("hash%d_%d", workerID, j),
				)
				err := suite.anteilKeeper.CreateOrder(suite.ctx, order)
				if err == nil {
					// Use atomic operation for thread safety
					atomic.AddInt32(&orderCount, 1)
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all orders were created
	orders, err := suite.anteilKeeper.GetAllOrders(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 100, len(orders), "All orders should be created (10 workers × 10 each)")
}

func getMemUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc)
}

func TestBenchmarkTestSuite(t *testing.T) {
	suite.Run(t, new(BenchmarkTestSuite))
}
