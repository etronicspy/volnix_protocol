package tests

import (
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	consensuskeeper "github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	consensustypes "github.com/volnix-protocol/volnix-protocol/x/consensus/types"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
	lizenzkeeper "github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
	lizenztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

// TestContext содержит все необходимые компоненты для тестирования
type TestContext struct {
	Cdc        codec.Codec
	Ctx        sdk.Context
	Cms        store.CommitMultiStore
	IdentKeeper     *identkeeper.Keeper
	LizenzKeeper    *lizenzkeeper.Keeper
	AnteilKeeper    *anteilkeeper.Keeper
	ConsensusKeeper *consensuskeeper.Keeper
	IdentStoreKey     storetypes.StoreKey
	LizenzStoreKey    storetypes.StoreKey
	AnteilStoreKey    storetypes.StoreKey
	ConsensusStoreKey storetypes.StoreKey
	IdentParamStore     paramtypes.Subspace
	LizenzParamStore    paramtypes.Subspace
	AnteilParamStore    paramtypes.Subspace
	ConsensusParamStore paramtypes.Subspace
}

// NewTestContext создает новый тестовый контекст со всеми необходимыми компонентами
// Это исправляет проблему "store does not exist" путем правильной инициализации всех stores
func NewTestContext(t require.TestingT) *TestContext {
	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create store keys
	identStoreKey := storetypes.NewKVStoreKey("test_ident")
	lizenzStoreKey := storetypes.NewKVStoreKey("test_lizenz")
	anteilStoreKey := storetypes.NewKVStoreKey("test_anteil")
	consensusStoreKey := storetypes.NewKVStoreKey("test_consensus")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")

	// Create test context with all store keys
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	
	// Mount all stores - это критически важно для исправления "store does not exist"
	cms.MountStoreWithDB(identStoreKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(lizenzStoreKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(anteilStoreKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(consensusStoreKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(tKey, storetypes.StoreTypeTransient, db)
	
	// Load latest version - обязательно перед созданием контекста
	err := cms.LoadLatestVersion()
	require.NoError(t, err, "failed to load latest version of commit multi store")

	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())

	// Create params keeper and subspaces
	paramsKeeper := paramskeeper.NewKeeper(cdc, codec.NewLegacyAmino(), identStoreKey, tKey)
	identParamStore := paramsKeeper.Subspace(identtypes.ModuleName)
	lizenzParamStore := paramsKeeper.Subspace(lizenztypes.ModuleName)
	anteilParamStore := paramsKeeper.Subspace(anteiltypes.ModuleName)
	consensusParamStore := paramsKeeper.Subspace(consensustypes.ModuleName)

	// Set key tables
	identParamStore = identParamStore.WithKeyTable(identtypes.ParamKeyTable())
	lizenzParamStore = lizenzParamStore.WithKeyTable(lizenztypes.ParamKeyTable())
	anteilParamStore = anteilParamStore.WithKeyTable(anteiltypes.ParamKeyTable())
	consensusParamStore = consensusParamStore.WithKeyTable(consensustypes.ParamKeyTable())

	// Create keepers
	identKeeper := identkeeper.NewKeeper(cdc, identStoreKey, identParamStore)
	lizenzKeeper := lizenzkeeper.NewKeeper(cdc, lizenzStoreKey, lizenzParamStore)
	anteilKeeper := anteilkeeper.NewKeeper(cdc, anteilStoreKey, anteilParamStore)
	consensusKeeper := consensuskeeper.NewKeeper(cdc, consensusStoreKey, consensusParamStore)

	// Set default params with increased limits for testing
	// Увеличиваем лимиты для исправления "Account limit exceeded"
	identParams := identtypes.DefaultParams()
	identParams.MaxIdentitiesPerAddress = 10000 // Значительно увеличиваем для тестов
	identKeeper.SetParams(ctx, identParams)

	lizenzKeeper.SetParams(ctx, lizenztypes.DefaultParams())
	anteilKeeper.SetParams(ctx, anteiltypes.DefaultParams())
	consensusKeeper.SetParams(ctx, *consensustypes.DefaultParams())

	return &TestContext{
		Cdc:                cdc,
		Ctx:                ctx,
		Cms:                cms,
		IdentKeeper:        identKeeper,
		LizenzKeeper:       lizenzKeeper,
		AnteilKeeper:       anteilKeeper,
		ConsensusKeeper:    consensusKeeper,
		IdentStoreKey:      identStoreKey,
		LizenzStoreKey:     lizenzStoreKey,
		AnteilStoreKey:     anteilStoreKey,
		ConsensusStoreKey:  consensusStoreKey,
		IdentParamStore:    identParamStore,
		LizenzParamStore:   lizenzParamStore,
		AnteilParamStore:   anteilParamStore,
		ConsensusParamStore: consensusParamStore,
	}
}

