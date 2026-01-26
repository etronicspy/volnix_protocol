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

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
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
	
	// Set default params
	params := types.DefaultParams()
	params.MaxIdentitiesPerAddress = 10000
	k.SetParams(ctx, params)

	return k, ctx
}

// BenchmarkSetVerifiedAccount benchmarks account creation
func BenchmarkSetVerifiedAccount(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		account := &identv1.VerifiedAccount{
			Address:      fmt.Sprintf("cosmos1test%d", i),
			Role:         identv1.Role_ROLE_CITIZEN,
			IdentityHash: fmt.Sprintf("hash%d", i),
			IsActive:     true,
		}
		k.SetVerifiedAccount(ctx, account)
	}
}

// BenchmarkGetVerifiedAccount benchmarks account retrieval
func BenchmarkGetVerifiedAccount(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	// Setup: Create 1000 accounts
	for i := 0; i < 1000; i++ {
		account := &identv1.VerifiedAccount{
			Address:      fmt.Sprintf("cosmos1test%d", i),
			Role:         identv1.Role_ROLE_CITIZEN,
			IdentityHash: fmt.Sprintf("hash%d", i),
			IsActive:     true,
		}
		k.SetVerifiedAccount(ctx, account)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k.GetVerifiedAccount(ctx, fmt.Sprintf("cosmos1test%d", i%1000))
	}
}

// BenchmarkCheckDuplicateIdentityHash benchmarks duplicate hash check
func BenchmarkCheckDuplicateIdentityHash(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	// Setup: Create 1000 accounts with different hashes
	for i := 0; i < 1000; i++ {
		account := &identv1.VerifiedAccount{
			Address:      fmt.Sprintf("cosmos1test%d", i),
			Role:         identv1.Role_ROLE_CITIZEN,
			IdentityHash: fmt.Sprintf("hash%d", i),
			IsActive:     true,
		}
		k.SetVerifiedAccount(ctx, account)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Check for new hash (not duplicate)
		k.CheckDuplicateIdentityHash(ctx, fmt.Sprintf("newhash%d", i), fmt.Sprintf("cosmos1new%d", i))
	}
}

// BenchmarkGetAllVerifiedAccounts benchmarks getting all accounts
func BenchmarkGetAllVerifiedAccounts(b *testing.B) {
	// Setup: Create different numbers of accounts
	accountCounts := []int{10, 100, 1000, 5000}
	
	for _, count := range accountCounts {
		b.Run(fmt.Sprintf("accounts_%d", count), func(b *testing.B) {
			// Create fresh context for each sub-benchmark
			kSub, ctxSub := setupBenchmark(b)
			
			// Create accounts
			for i := 0; i < count; i++ {
				account := &identv1.VerifiedAccount{
					Address:      fmt.Sprintf("cosmos1test%d", i),
					Role:         identv1.Role_ROLE_CITIZEN,
					IdentityHash: fmt.Sprintf("hash%d", i),
					IsActive:     true,
				}
				kSub.SetVerifiedAccount(ctxSub, account)
			}
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				kSub.GetAllVerifiedAccounts(ctxSub)
			}
		})
	}
}

// BenchmarkChangeAccountRole benchmarks role changes
func BenchmarkChangeAccountRole(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	// Setup: Create 100 citizens
	for i := 0; i < 100; i++ {
		account := &identv1.VerifiedAccount{
			Address:      fmt.Sprintf("cosmos1test%d", i),
			Role:         identv1.Role_ROLE_CITIZEN,
			IdentityHash: fmt.Sprintf("hash%d", i),
			IsActive:     true,
		}
		k.SetVerifiedAccount(ctx, account)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Toggle between CITIZEN and VALIDATOR
		newRole := identv1.Role_ROLE_VALIDATOR
		if i%2 == 0 {
			newRole = identv1.Role_ROLE_CITIZEN
		}
		k.ChangeAccountRole(ctx, fmt.Sprintf("cosmos1test%d", i%100), newRole)
	}
}

// BenchmarkGetVerifiedAccountsByRole benchmarks filtered account retrieval
func BenchmarkGetVerifiedAccountsByRole(b *testing.B) {
	k, ctx := setupBenchmark(b)
	
	// Setup: Create 500 citizens, 300 validators, 200 guests
	roles := []struct {
		role  identv1.Role
		count int
	}{
		{identv1.Role_ROLE_CITIZEN, 500},
		{identv1.Role_ROLE_VALIDATOR, 300},
		{identv1.Role_ROLE_GUEST, 200},
	}
	
	idx := 0
	for _, r := range roles {
		for i := 0; i < r.count; i++ {
			account := &identv1.VerifiedAccount{
				Address:      fmt.Sprintf("cosmos1test%d", idx),
				Role:         r.role,
				IdentityHash: fmt.Sprintf("hash%d", idx),
				IsActive:     true,
			}
			k.SetVerifiedAccount(ctx, account)
			idx++
		}
	}
	
	b.Run("citizens", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			k.GetVerifiedAccountsByRole(ctx, identv1.Role_ROLE_CITIZEN)
		}
	})
	
	b.Run("validators", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			k.GetVerifiedAccountsByRole(ctx, identv1.Role_ROLE_VALIDATOR)
		}
	})
}
