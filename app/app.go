package app

import (
	"encoding/json"
	"fmt"
	"io"

	sdklog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	// custom modules
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	"github.com/volnix-protocol/volnix-protocol/x/consensus"
	consensustypes "github.com/volnix-protocol/volnix-protocol/x/consensus/types"
	"github.com/volnix-protocol/volnix-protocol/x/ident"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz"
	lizenztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"

	// keeper imports
	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	consensuskeeper "github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	lizenzkeeper "github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
)

// VolnixApp wires BaseApp with custom module keepers and services.
type VolnixApp struct {
	*baseapp.BaseApp

	appCodec codec.Codec

	// store keys
	keyParams    *storetypes.KVStoreKey
	tkeyParams   *storetypes.TransientStoreKey
	keyIdent     *storetypes.KVStoreKey
	keyLizenz    *storetypes.KVStoreKey
	keyAnteil    *storetypes.KVStoreKey
	keyConsensus *storetypes.KVStoreKey

	// keepers
	paramsKeeper paramskeeper.Keeper

	// custom module keepers
	identKeeper     *identkeeper.Keeper
	lizenzKeeper    *lizenzkeeper.Keeper
	anteilKeeper    *anteilkeeper.Keeper
	consensusKeeper *consensuskeeper.Keeper

	// module manager
	mm *module.Manager
}

func NewVolnixApp(logger sdklog.Logger, db cosmosdb.DB, traceStore io.Writer, encoding EncodingConfig) *VolnixApp {
	bapp := baseapp.NewBaseApp("volnix", logger, db, encoding.TxConfig.TxDecoder())
	bapp.SetVersion("0.1.0")
	// Provide interface registry so Msg/Query services can be registered safely
	bapp.SetInterfaceRegistry(encoding.InterfaceRegistry)

	// Store keys
	keyParams := storetypes.NewKVStoreKey(paramtypes.StoreKey)
	tkeyParams := storetypes.NewTransientStoreKey(paramtypes.TStoreKey)
	keyIdent := storetypes.NewKVStoreKey(identtypes.StoreKey)
	keyLizenz := storetypes.NewKVStoreKey(lizenztypes.StoreKey)
	keyAnteil := storetypes.NewKVStoreKey(anteiltypes.StoreKey)
	keyConsensus := storetypes.NewKVStoreKey(consensustypes.StoreKey)

	// Mount stores
	bapp.MountKVStores(map[string]*storetypes.KVStoreKey{
		paramtypes.StoreKey:     keyParams,
		identtypes.StoreKey:     keyIdent,
		lizenztypes.StoreKey:    keyLizenz,
		anteiltypes.StoreKey:    keyAnteil,
		consensustypes.StoreKey: keyConsensus,
	})
	bapp.MountTransientStores(map[string]*storetypes.TransientStoreKey{
		paramtypes.TStoreKey: tkeyParams,
	})

	// Params keeper and subspaces
	paramsKeeper := paramskeeper.NewKeeper(encoding.Codec, encoding.LegacyAmino, keyParams, tkeyParams)
	// Create subspaces for custom modules
	identSubspace := paramsKeeper.Subspace(identtypes.ModuleName)
	lizenzSubspace := paramsKeeper.Subspace(lizenztypes.ModuleName)
	anteilSubspace := paramsKeeper.Subspace(anteiltypes.ModuleName)
	consensusSubspace := paramsKeeper.Subspace(consensustypes.ModuleName)

	// Custom module keepers (constructors provided by each module's module.go)
	identKeeper := identkeeper.NewKeeper(encoding.Codec, keyIdent, identSubspace)
	lizenzKeeper := lizenzkeeper.NewKeeper(encoding.Codec, keyLizenz, lizenzSubspace)
	anteilKeeper := anteilkeeper.NewKeeper(encoding.Codec, keyAnteil, anteilSubspace)
	consensusKeeper := consensuskeeper.NewKeeper(encoding.Codec, keyConsensus, consensusSubspace)

	// Module manager (register Msg/Query services only at this stage)
	mm := module.NewManager(
		ident.NewAppModule(identKeeper),
		lizenz.NewAppModule(lizenzKeeper),
		anteil.NewAppModule(anteilKeeper),
		consensus.NewConsensusAppModule(encoding.Codec, *consensusKeeper),
	)

	// Create app instance
	app := &VolnixApp{
		BaseApp:         bapp,
		appCodec:        encoding.Codec,
		keyParams:       keyParams,
		tkeyParams:      tkeyParams,
		keyIdent:        keyIdent,
		keyLizenz:       keyLizenz,
		keyAnteil:       keyAnteil,
		keyConsensus:    keyConsensus,
		paramsKeeper:    paramsKeeper,
		identKeeper:     identKeeper,
		lizenzKeeper:    lizenzKeeper,
		anteilKeeper:    anteilKeeper,
		consensusKeeper: consensusKeeper,
		mm:              mm,
	}

	// Register interfaces first
	encoding.InterfaceRegistry.RegisterImplementations((*sdk.Msg)(nil),
		&anteilv1.MsgPlaceOrder{},
		&anteilv1.MsgCancelOrder{},
		&anteilv1.MsgUpdateOrder{},
		&anteilv1.MsgPlaceBid{},
		&anteilv1.MsgSettleAuction{},
	)
	encoding.InterfaceRegistry.RegisterImplementations((*sdk.Msg)(nil),
		&identv1.MsgVerifyIdentity{},
		&identv1.MsgMigrateRole{},
		&identv1.MsgChangeRole{},
	)
	encoding.InterfaceRegistry.RegisterImplementations((*sdk.Msg)(nil),
		&lizenzv1.MsgActivateLZN{},
		&lizenzv1.MsgDeactivateLZN{},
	)

	// Register Msg/Query services
	configurator := module.NewConfigurator(encoding.Codec, bapp.MsgServiceRouter(), bapp.GRPCQueryRouter())
	if err := mm.RegisterServices(configurator); err != nil {
		panic(err)
	}

	// Set BeginBlocker and EndBlocker for all modules
	bapp.SetBeginBlocker(func(ctx sdk.Context) (sdk.BeginBlock, error) {
		// Execute BeginBlocker for all modules
		if err := identKeeper.BeginBlocker(ctx); err != nil {
			return sdk.BeginBlock{}, fmt.Errorf("ident BeginBlocker failed: %w", err)
		}
		if err := anteilKeeper.BeginBlocker(ctx); err != nil {
			return sdk.BeginBlock{}, fmt.Errorf("anteil BeginBlocker failed: %w", err)
		}
		if err := consensusKeeper.BeginBlocker(ctx); err != nil {
			return sdk.BeginBlock{}, fmt.Errorf("consensus BeginBlocker failed: %w", err)
		}
		return sdk.BeginBlock{}, nil
	})

	bapp.SetEndBlocker(func(ctx sdk.Context) (sdk.EndBlock, error) {
		// Execute EndBlocker for all modules
		if err := identKeeper.EndBlocker(ctx); err != nil {
			return sdk.EndBlock{}, fmt.Errorf("ident EndBlocker failed: %w", err)
		}
		if err := anteilKeeper.EndBlocker(ctx); err != nil {
			return sdk.EndBlock{}, fmt.Errorf("anteil EndBlocker failed: %w", err)
		}
		if err := consensusKeeper.EndBlocker(ctx); err != nil {
			return sdk.EndBlock{}, fmt.Errorf("consensus EndBlocker failed: %w", err)
		}
		return sdk.EndBlock{}, nil
	})

	// InitGenesis handler (v0.53 InitChainer signature)
	bapp.SetInitChainer(func(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
		// If AppStateBytes is empty, BaseApp will have no-op; the CLI can pass default genesis explicitly
		// and we also support initializing from provided bytes.
		var genesisState map[string]json.RawMessage
		if len(req.AppStateBytes) > 0 {
			if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
				return nil, err
			}
		} else {
			// Create default genesis state
			genesisState = make(map[string]json.RawMessage)

			// Custom modules genesis
			genesisState[identtypes.ModuleName] = encoding.Codec.MustMarshalJSON(ident.DefaultGenesis())
			genesisState[lizenztypes.ModuleName] = encoding.Codec.MustMarshalJSON(lizenz.DefaultGenesis())
			genesisState[anteiltypes.ModuleName] = encoding.Codec.MustMarshalJSON(anteil.DefaultGenesis())
			genesisState[consensustypes.ModuleName] = encoding.Codec.MustMarshalJSON(consensus.DefaultGenesis())
		}
		_, err := mm.InitGenesis(ctx, encoding.Codec, genesisState)
		if err != nil {
			return nil, err
		}
		return &abci.ResponseInitChain{}, nil
	})

	return app
}
