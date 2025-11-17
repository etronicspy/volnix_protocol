package app

import (
	"context"
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
	"github.com/volnix-protocol/volnix-protocol/x/anteil"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	"github.com/volnix-protocol/volnix-protocol/x/consensus"
	consensustypes "github.com/volnix-protocol/volnix-protocol/x/consensus/types"
	"github.com/volnix-protocol/volnix-protocol/x/governance"
	governancetypes "github.com/volnix-protocol/volnix-protocol/x/governance/types"
	"github.com/volnix-protocol/volnix-protocol/x/ident"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz"
	lizenztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"

	// keeper imports
	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	consensuskeeper "github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	governancekeeper "github.com/volnix-protocol/volnix-protocol/x/governance/keeper"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	lizenzkeeper "github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
	
	// proto imports for adapters
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
)

// Application name
const Name = "volnix"

// LizenzKeeperAdapter adapts lizenz keeper to consensus interface
// Converts []*lizenzv1.ActivatedLizenz to []interface{} for interface compatibility
type LizenzKeeperAdapter struct {
	keeper *lizenzkeeper.Keeper
}

// GetAllActivatedLizenz returns all activated LZN as []interface{}
func (a *LizenzKeeperAdapter) GetAllActivatedLizenz(ctx sdk.Context) ([]interface{}, error) {
	lizenzs, err := a.keeper.GetAllActivatedLizenz(ctx)
	if err != nil {
		return nil, err
	}
	
	// Convert []*lizenzv1.ActivatedLizenz to []interface{}
	result := make([]interface{}, len(lizenzs))
	for i, lizenz := range lizenzs {
		result[i] = lizenz
	}
	return result, nil
}

// GetTotalActivatedLizenz returns total activated LZN
func (a *LizenzKeeperAdapter) GetTotalActivatedLizenz(ctx sdk.Context) (string, error) {
	return a.keeper.GetTotalActivatedLizenz(ctx)
}

// GetMOACompliance returns MOA compliance ratio
func (a *LizenzKeeperAdapter) GetMOACompliance(ctx sdk.Context, validator string) (float64, error) {
	return a.keeper.GetMOACompliance(ctx, validator)
}

// UpdateRewardStats updates reward statistics
func (a *LizenzKeeperAdapter) UpdateRewardStats(ctx sdk.Context, validator string, rewardAmount uint64, blockHeight uint64, moaCompliance float64, penaltyMultiplier float64, baseReward uint64) error {
	return a.keeper.UpdateRewardStats(ctx, validator, rewardAmount, blockHeight, moaCompliance, penaltyMultiplier, baseReward)
}

// AnteilKeeperAdapter adapts anteil keeper to consensus interface
// Converts *anteilv1.UserPosition to interface{} for interface compatibility
type AnteilKeeperAdapter struct {
	keeper *anteilkeeper.Keeper
}

// GetUserPosition returns user position as interface{}
func (a *AnteilKeeperAdapter) GetUserPosition(ctx sdk.Context, user string) (interface{}, error) {
	return a.keeper.GetUserPosition(ctx, user)
}

// SetUserPosition sets user position
func (a *AnteilKeeperAdapter) SetUserPosition(ctx sdk.Context, position interface{}) error {
	// Type assert to *anteilv1.UserPosition
	userPos, ok := position.(*anteilv1.UserPosition)
	if !ok {
		return fmt.Errorf("invalid position type: expected *anteilv1.UserPosition")
	}
	return a.keeper.SetUserPosition(ctx, userPos)
}

// UpdateUserPosition updates user position
func (a *AnteilKeeperAdapter) UpdateUserPosition(ctx sdk.Context, user string, antBalance string, orderCount uint32) error {
	return a.keeper.UpdateUserPosition(ctx, user, antBalance, orderCount)
}

// VolnixApp wires BaseApp with custom module keepers and services.
type VolnixApp struct {
	*baseapp.BaseApp

	appCodec codec.Codec

	// store keys
	keyParams       *storetypes.KVStoreKey
	tkeyParams      *storetypes.TransientStoreKey
	keyIdent        *storetypes.KVStoreKey
	keyLizenz       *storetypes.KVStoreKey
	keyAnteil       *storetypes.KVStoreKey
	keyConsensus    *storetypes.KVStoreKey
	keyGovernance   *storetypes.KVStoreKey

	// keepers
	paramsKeeper paramskeeper.Keeper

	// custom module keepers
	identKeeper      *identkeeper.Keeper
	lizenzKeeper     *lizenzkeeper.Keeper
	anteilKeeper     *anteilkeeper.Keeper
	consensusKeeper  *consensuskeeper.Keeper
	governanceKeeper *governancekeeper.Keeper

	// module manager
	mm *module.Manager
}

func NewVolnixApp(logger sdklog.Logger, db cosmosdb.DB, traceStore io.Writer, encoding EncodingConfig) *VolnixApp {
	bapp := baseapp.NewBaseApp("volnix", logger, db, encoding.TxConfig.TxDecoder)
	bapp.SetVersion("0.1.0")
	// Provide interface registry so Msg/Query services can be registered safely
	bapp.SetInterfaceRegistry(encoding.InterfaceRegistry)
	// Minimal Tx encoder to match TxConfig
	bapp.SetTxEncoder(encoding.TxConfig.TxEncoder)

	// Store keys
	keyParams := storetypes.NewKVStoreKey(paramtypes.StoreKey)
	tkeyParams := storetypes.NewTransientStoreKey(paramtypes.TStoreKey)
	keyIdent := storetypes.NewKVStoreKey(identtypes.StoreKey)
	keyLizenz := storetypes.NewKVStoreKey(lizenztypes.StoreKey)
	keyAnteil := storetypes.NewKVStoreKey(anteiltypes.StoreKey)
	keyConsensus := storetypes.NewKVStoreKey(consensustypes.StoreKey)
	keyGovernance := storetypes.NewKVStoreKey(governancetypes.StoreKey)

	// Mount stores
	bapp.MountKVStores(map[string]*storetypes.KVStoreKey{
		paramtypes.StoreKey:      keyParams,
		identtypes.StoreKey:      keyIdent,
		lizenztypes.StoreKey:     keyLizenz,
		anteiltypes.StoreKey:     keyAnteil,
		consensustypes.StoreKey:  keyConsensus,
		governancetypes.StoreKey: keyGovernance,
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
	governanceSubspace := paramsKeeper.Subspace(governancetypes.ModuleName)

	// Custom module keepers (constructors provided by each module's module.go)
	identKeeper := identkeeper.NewKeeper(encoding.Codec, keyIdent, identSubspace)
	lizenzKeeper := lizenzkeeper.NewKeeper(encoding.Codec, keyLizenz, lizenzSubspace)
	anteilKeeper := anteilkeeper.NewKeeper(encoding.Codec, keyAnteil, anteilSubspace)
	consensusKeeper := consensuskeeper.NewKeeper(encoding.Codec, keyConsensus, consensusSubspace)
	governanceKeeper := governancekeeper.NewKeeper(encoding.Codec, keyGovernance, governanceSubspace)

	// Set up governance keeper dependencies for parameter updates
	// Governance can update parameters in other modules
	governanceKeeper.SetLizenzKeeper(lizenzKeeper)
	governanceKeeper.SetAnteilKeeper(anteilKeeper)
	governanceKeeper.SetConsensusKeeper(consensusKeeper)
	// TODO: Set bank keeper when bank module is available for WRT balance queries

	// Set up consensus keeper dependencies
	// Consensus needs lizenz keeper for reward distribution and anteil keeper for ANT balances
	// Create adapter wrappers to convert types for interface compatibility
	lizenzAdapter := &LizenzKeeperAdapter{keeper: lizenzKeeper}
	anteilAdapter := &AnteilKeeperAdapter{keeper: anteilKeeper}
	consensusKeeper.SetLizenzKeeper(lizenzAdapter)
	consensusKeeper.SetAnteilKeeper(anteilAdapter)
	// TODO: Set bank keeper when bank module is available for sending WRT rewards

	// Interface registration temporarily disabled for CometBFT integration

	// Module manager (register Msg/Query services only at this stage)
	mm := module.NewManager(
		ident.NewAppModule(identKeeper),
		lizenz.NewAppModule(lizenzKeeper),
		anteil.NewAppModule(anteilKeeper),
		consensus.NewConsensusAppModule(encoding.Codec, *consensusKeeper),
		governance.NewAppModule(governanceKeeper),
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
		keyGovernance:   keyGovernance,
		paramsKeeper:    paramsKeeper,
		identKeeper:     identKeeper,
		lizenzKeeper:    lizenzKeeper,
		anteilKeeper:    anteilKeeper,
		consensusKeeper: consensusKeeper,
		governanceKeeper: governanceKeeper,
		mm:              mm,
	}

	// Register interfaces first
	basicManager := module.NewBasicManager(
		ident.AppModuleBasic{},
		lizenz.AppModuleBasic{},
		anteil.AppModuleBasic{},
		consensus.ConsensusAppModuleBasic{},
		governance.AppModuleBasic{},
	)
	basicManager.RegisterInterfaces(encoding.InterfaceRegistry)

	// Register Msg/Query services
	configurator := module.NewConfigurator(encoding.Codec, bapp.MsgServiceRouter(), bapp.GRPCQueryRouter())
	if err := mm.RegisterServices(configurator); err != nil {
		panic(err)
	}

	// Minimal AnteHandler: check for signatures presence only
	bapp.SetAnteHandler(MinimalAnteHandler)

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
			// Governance genesis uses JSON marshaling (not proto)
			govGenState := governance.DefaultGenesis()
			govGenBz, err := json.Marshal(govGenState)
			if err != nil {
				panic(fmt.Errorf("failed to marshal governance genesis: %w", err))
			}
			genesisState[governancetypes.ModuleName] = govGenBz
		}
		_, err := mm.InitGenesis(ctx, encoding.Codec, genesisState)
		if err != nil {
			return nil, err
		}
		return &abci.ResponseInitChain{}, nil
	})

	return app
}

// ExportAppStateAndValidators exports the state of the application for a genesis file.
func (app *VolnixApp) ExportAppStateAndValidators(
	forZeroHeight bool, jailAllowedAddrs []string,
) (map[string]json.RawMessage, error) {
	// Create the context
	ctx := app.NewContext(true)

	// Export genesis state
	genesisState, err := app.mm.ExportGenesis(ctx, app.appCodec)
	if err != nil {
		return nil, err
	}

	return genesisState, nil
}

// GetBaseApp returns the base application of type *baseapp.BaseApp.
func (app *VolnixApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *VolnixApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	// For now, return empty map since we don't have standard modules yet
	return modAccAddrs
}

// GetModuleManager returns the app module manager.
func (app *VolnixApp) GetModuleManager() *module.Manager {
	return app.mm
}

// ModuleManager returns the app module manager (alias for compatibility).
func (app *VolnixApp) ModuleManager() *module.Manager {
	return app.mm
}

// GetConsensusKeeper returns the consensus keeper.
func (app *VolnixApp) GetConsensusKeeper() *consensuskeeper.Keeper {
	return app.consensusKeeper
}

// AppBasicManager returns the app's basic module manager.
func (app *VolnixApp) AppBasicManager() *module.Manager {
	return app.mm
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	// For now, return empty map since we don't have standard modules yet
	return make(map[string][]string)
}

// initParamsKeeper function removed - not used in current implementation

// ABCI methods for CometBFT compatibility

// ApplySnapshotChunk implements the ABCI interface with context
func (app *VolnixApp) ApplySnapshotChunk(ctx context.Context, req *abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	// For now, return a simple response - snapshot functionality can be added later
	return &abci.ResponseApplySnapshotChunk{
		Result: abci.ResponseApplySnapshotChunk_ACCEPT,
	}, nil
}

// LoadSnapshotChunk implements the ABCI interface with context
func (app *VolnixApp) LoadSnapshotChunk(ctx context.Context, req *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	// For now, return empty chunk - snapshot functionality can be added later
	return &abci.ResponseLoadSnapshotChunk{}, nil
}

// ListSnapshots implements the ABCI interface with context
func (app *VolnixApp) ListSnapshots(ctx context.Context, req *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	// For now, return empty list - snapshot functionality can be added later
	return &abci.ResponseListSnapshots{}, nil
}

// OfferSnapshot implements the ABCI interface with context
func (app *VolnixApp) OfferSnapshot(ctx context.Context, req *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	// For now, reject snapshots - snapshot functionality can be added later
	return &abci.ResponseOfferSnapshot{
		Result: abci.ResponseOfferSnapshot_REJECT,
	}, nil
}
