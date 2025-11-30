package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	coreservice "cosmossdk.io/core/store"
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

	// bank module for token management
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

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
	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Application name
const Name = "volnix"

// kvStoreServiceWrapper wraps KVStoreKey to implement KVStoreService interface
// This is needed for bank keeper in Cosmos SDK v0.53
type kvStoreServiceWrapper struct {
	key *storetypes.KVStoreKey
}

// OpenKVStore opens a KVStore from the service
func (w *kvStoreServiceWrapper) OpenKVStore(ctx context.Context) coreservice.KVStore {
	// This will be called by bank keeper to get the store
	// The actual store access happens through sdk.Context
	// For now, we return a simple implementation
	return &kvStoreWrapper{key: w.key}
}

// kvStoreWrapper implements KVStore interface
type kvStoreWrapper struct {
	key *storetypes.KVStoreKey
}

func (w *kvStoreWrapper) Get(key []byte) ([]byte, error) {
	// This is a placeholder - actual implementation uses sdk.Context
	return nil, fmt.Errorf("kvStoreWrapper.Get should not be called directly")
}

func (w *kvStoreWrapper) Has(key []byte) (bool, error) {
	return false, fmt.Errorf("kvStoreWrapper.Has should not be called directly")
}

func (w *kvStoreWrapper) Set(key, value []byte) error {
	return fmt.Errorf("kvStoreWrapper.Set should not be called directly")
}

func (w *kvStoreWrapper) Delete(key []byte) error {
	return fmt.Errorf("kvStoreWrapper.Delete should not be called directly")
}

func (w *kvStoreWrapper) Iterator(start, end []byte) (coreservice.Iterator, error) {
	return nil, fmt.Errorf("kvStoreWrapper.Iterator should not be called directly")
}

func (w *kvStoreWrapper) ReverseIterator(start, end []byte) (coreservice.Iterator, error) {
	return nil, fmt.Errorf("kvStoreWrapper.ReverseIterator should not be called directly")
}

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

// ConsensusKeeperAdapter adapts consensus keeper to lizenz interface
// Converts map[string]interface{} to *consensusv1.Validator for interface compatibility
type ConsensusKeeperAdapter struct {
	keeper *consensuskeeper.Keeper
}

// SetValidator sets a validator from interface{} (map[string]interface{})
func (a *ConsensusKeeperAdapter) SetValidator(ctx sdk.Context, validator interface{}) error {
	// Type assert to map[string]interface{}
	validatorMap, ok := validator.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid validator type: expected map[string]interface{}, got %T", validator)
	}

	// Safely extract values from map
	validatorAddr, _ := validatorMap["validator"].(string)
	antBalance, _ := validatorMap["ant_balance"].(string)
	status, _ := validatorMap["status"].(int)
	lastActive, _ := validatorMap["last_active"].(*timestamppb.Timestamp)
	lastBlockHeight, _ := validatorMap["last_block_height"].(uint64)
	moaScore, _ := validatorMap["moa_score"].(string)
	activityScore, _ := validatorMap["activity_score"].(string)
	totalBlocksCreated, _ := validatorMap["total_blocks_created"].(uint64)
	totalBurnAmount, _ := validatorMap["total_burn_amount"].(string)

	// Create validator object
	validatorObj := &consensusv1.Validator{
		Validator:          validatorAddr,
		AntBalance:         antBalance,
		Status:             consensusv1.ValidatorStatus(status),
		LastActive:         lastActive,
		LastBlockHeight:    lastBlockHeight,
		MoaScore:           moaScore,
		ActivityScore:      activityScore,
		TotalBlocksCreated: totalBlocksCreated,
		TotalBurnAmount:    totalBurnAmount,
	}

	a.keeper.SetValidator(ctx, validatorObj)
	return nil
}

// SetValidatorWeight sets validator weight
func (a *ConsensusKeeperAdapter) SetValidatorWeight(ctx sdk.Context, validator, weight string) error {
	return a.keeper.SetValidatorWeight(ctx, validator, weight)
}

// AnteilKeeperAdapterForIdent adapts anteil keeper to ident interface
// Allows ident module to burn ANT when citizens are deactivated
type AnteilKeeperAdapterForIdent struct {
	keeper *anteilkeeper.Keeper
}

func (a *AnteilKeeperAdapterForIdent) BurnAntFromUser(ctx sdk.Context, user string) error {
	return a.keeper.BurnAntFromUser(ctx, user)
}

func (a *AnteilKeeperAdapterForIdent) GetUserPosition(ctx sdk.Context, user string) (interface{}, error) {
	return a.keeper.GetUserPosition(ctx, user)
}

// AnteilKeeperAdapterForLizenz adapts anteil keeper to lizenz interface
// Converts map[string]interface{} to *anteilv1.UserPosition for interface compatibility
type AnteilKeeperAdapterForLizenz struct {
	keeper *anteilkeeper.Keeper
}

// SetUserPosition sets user position from interface{} (map[string]interface{})
func (a *AnteilKeeperAdapterForLizenz) SetUserPosition(ctx sdk.Context, position interface{}) error {
	// Type assert to map[string]interface{}
	positionMap, ok := position.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid position type: expected map[string]interface{}, got %T", position)
	}

	// Safely extract values from map
	owner, _ := positionMap["owner"].(string)
	antBalance, _ := positionMap["ant_balance"].(string)
	lockedAnt, _ := positionMap["locked_ant"].(string)
	availableAnt, _ := positionMap["available_ant"].(string)

	// Create position object
	positionObj := &anteilv1.UserPosition{
		Owner:        owner,
		AntBalance:   antBalance,
		LockedAnt:    lockedAnt,
		AvailableAnt: availableAnt,
	}

	return a.keeper.SetUserPosition(ctx, positionObj)
}

// BankKeeperAdapterForConsensus adapts bank keeper to consensus interface
// Implements BankKeeperInterface for consensus module
type BankKeeperAdapterForConsensus struct {
	keeper bankkeeper.Keeper
}

// SendCoins sends coins from one account to another
func (a *BankKeeperAdapterForConsensus) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return a.keeper.SendCoins(ctx, fromAddr, toAddr, amt)
}

// MintCoins mints coins to a module account
func (a *BankKeeperAdapterForConsensus) MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	return a.keeper.MintCoins(ctx, moduleName, amt)
}

// SendCoinsFromModuleToAccount sends coins from a module account to a regular account
func (a *BankKeeperAdapterForConsensus) SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return a.keeper.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt)
}

// BankKeeperAdapterForGovernance adapts bank keeper to governance interface
// Implements BankKeeperInterface for governance module
type BankKeeperAdapterForGovernance struct {
	keeper bankkeeper.Keeper
}

// GetBalance returns the balance of a specific denomination for an account
func (a *BankKeeperAdapterForGovernance) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return a.keeper.GetBalance(ctx, addr, denom)
}

// GetAllBalances returns all balances for an account
func (a *BankKeeperAdapterForGovernance) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return a.keeper.GetAllBalances(ctx, addr)
}

// GetSupply returns the total supply of a denomination
func (a *BankKeeperAdapterForGovernance) GetSupply(ctx sdk.Context, denom string) sdk.Coin {
	return a.keeper.GetSupply(ctx, denom)
}

// BankKeeperAdapterForLizenz adapts bank keeper to lizenz interface
// Implements BankKeeperInterface for lizenz module
type BankKeeperAdapterForLizenz struct {
	keeper bankkeeper.Keeper
}

// SendCoins sends coins from one account to another
func (a *BankKeeperAdapterForLizenz) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return a.keeper.SendCoins(ctx, fromAddr, toAddr, amt)
}

// SendCoinsFromAccountToModule sends coins from an account to a module account
func (a *BankKeeperAdapterForLizenz) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return a.keeper.SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt)
}

// SendCoinsFromModuleToAccount sends coins from a module account to a regular account
func (a *BankKeeperAdapterForLizenz) SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return a.keeper.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt)
}

// GetBalance returns the balance of a specific denomination for an account
func (a *BankKeeperAdapterForLizenz) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return a.keeper.GetBalance(ctx, addr, denom)
}

// VolnixApp wires BaseApp with custom module keepers and services.
type VolnixApp struct {
	*baseapp.BaseApp

	appCodec codec.Codec

	// store keys
	keyParams     *storetypes.KVStoreKey
	tkeyParams    *storetypes.TransientStoreKey
	keyBank       *storetypes.KVStoreKey
	keyIdent      *storetypes.KVStoreKey
	keyLizenz     *storetypes.KVStoreKey
	keyAnteil     *storetypes.KVStoreKey
	keyConsensus  *storetypes.KVStoreKey
	keyGovernance *storetypes.KVStoreKey

	// keepers
	paramsKeeper paramskeeper.Keeper
	bankKeeper   bankkeeper.Keeper

	// custom module keepers
	identKeeper      *identkeeper.Keeper
	lizenzKeeper     *lizenzkeeper.Keeper
	anteilKeeper     *anteilkeeper.Keeper
	consensusKeeper  *consensuskeeper.Keeper
	governanceKeeper *governancekeeper.Keeper

	// module manager
	mm *module.Manager

	// IMPROVED: Upgrade manager for handling network upgrades
	upgradeManager *UpgradeManager

	// IMPROVED: Rate limiter for DDoS protection
	rateLimiter *RateLimiter

	// IMPROVED: Snapshot manager for State Sync
	snapshotManager *SnapshotManager
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
	keyBank := storetypes.NewKVStoreKey(banktypes.StoreKey)
	keyIdent := storetypes.NewKVStoreKey(identtypes.StoreKey)
	keyLizenz := storetypes.NewKVStoreKey(lizenztypes.StoreKey)
	keyAnteil := storetypes.NewKVStoreKey(anteiltypes.StoreKey)
	keyConsensus := storetypes.NewKVStoreKey(consensustypes.StoreKey)
	keyGovernance := storetypes.NewKVStoreKey(governancetypes.StoreKey)

	// Mount stores
	bapp.MountKVStores(map[string]*storetypes.KVStoreKey{
		paramtypes.StoreKey:      keyParams,
		banktypes.StoreKey:       keyBank,
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
	// Create subspaces for modules
	identSubspace := paramsKeeper.Subspace(identtypes.ModuleName)
	lizenzSubspace := paramsKeeper.Subspace(lizenztypes.ModuleName)
	anteilSubspace := paramsKeeper.Subspace(anteiltypes.ModuleName)
	consensusSubspace := paramsKeeper.Subspace(consensustypes.ModuleName)
	governanceSubspace := paramsKeeper.Subspace(governancetypes.ModuleName)

	// Bank keeper for token management (WRT, LZN, ANT)
	// In Cosmos SDK v0.53, bank keeper requires KVStoreService and AccountKeeper
	// We create a simple KVStoreService wrapper from KVStoreKey
	// For minimal integration, we use nil AccountKeeper (limits some functionality)
	// TODO: Add proper AccountKeeper integration for full functionality

	// Create KVStoreService wrapper
	// In v0.53, we need to implement KVStoreService interface
	// For now, we create a simple wrapper that uses the KVStoreKey
	bankStoreService := &kvStoreServiceWrapper{key: keyBank}

	// Create a valid authority address for bank keeper
	// In Cosmos SDK, authority must be a valid bech32 address
	// For now, we use a placeholder that will be set properly in production
	authorityAddr := sdk.AccAddress("authority123456789012345678901234567890") // 32 bytes minimum
	if len(authorityAddr) == 0 {
		// Fallback: create a valid bech32 address
		authorityAddr = sdk.AccAddress(make([]byte, 20)) // 20 bytes for standard address
	}
	authority := authorityAddr.String()
	if authority == "" {
		// If still empty, use a default valid address format
		authority = "volnix1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq" // Placeholder address
	}

	// Create bank keeper
	// Note: Without AccountKeeper, account-related operations are limited
	// But SendCoins, MintCoins, GetBalance operations should work
	bankKeeper := bankkeeper.NewBaseKeeper(
		encoding.Codec,
		bankStoreService,
		nil,               // account keeper - will be added later if needed
		map[string]bool{}, // blocked addresses
		authority,
		logger,
	)

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
	bankAdapterForGovernance := &BankKeeperAdapterForGovernance{keeper: bankKeeper}
	governanceKeeper.SetBankKeeper(bankAdapterForGovernance)

	// Set up consensus keeper dependencies
	// Consensus needs lizenz keeper for reward distribution and anteil keeper for ANT balances
	// Create adapter wrappers to convert types for interface compatibility
	lizenzAdapter := &LizenzKeeperAdapter{keeper: lizenzKeeper}
	anteilAdapter := &AnteilKeeperAdapter{keeper: anteilKeeper}
	consensusKeeper.SetLizenzKeeper(lizenzAdapter)
	consensusKeeper.SetAnteilKeeper(anteilAdapter)

	// Set ident keeper in anteil keeper for ANT distribution to citizens
	anteilKeeper.SetIdentKeeper(identKeeper)
	bankAdapterForConsensus := &BankKeeperAdapterForConsensus{keeper: bankKeeper}
	consensusKeeper.SetBankKeeper(bankAdapterForConsensus)

	// Lizenz needs consensus keeper for validator registration, anteil keeper for initial ANT position, and bank keeper for LZN locking
	// Create adapter wrappers to convert types for interface compatibility
	consensusAdapterForLizenz := &ConsensusKeeperAdapter{keeper: consensusKeeper}
	anteilAdapterForLizenz := &AnteilKeeperAdapterForLizenz{keeper: anteilKeeper}
	bankAdapterForLizenz := &BankKeeperAdapterForLizenz{keeper: bankKeeper}
	lizenzKeeper.SetConsensusKeeper(consensusAdapterForLizenz)
	lizenzKeeper.SetAnteilKeeper(anteilAdapterForLizenz)
	lizenzKeeper.SetBankKeeper(bankAdapterForLizenz)

	// Ident needs anteil keeper for burning ANT on citizen deactivation
	anteilAdapterForIdent := &AnteilKeeperAdapterForIdent{keeper: anteilKeeper}
	identKeeper.SetAnteilKeeper(anteilAdapterForIdent)

	// Interface registration temporarily disabled for CometBFT integration

	// Module manager (register Msg/Query services only at this stage)
	mm := module.NewManager(
		bank.NewAppModule(encoding.Codec, bankKeeper, nil, nil), // account keeper and blocked addresses not needed for basic operations
		ident.NewAppModule(identKeeper),
		lizenz.NewAppModule(lizenzKeeper),
		anteil.NewAppModule(anteilKeeper),
		consensus.NewConsensusAppModule(encoding.Codec, *consensusKeeper),
		governance.NewAppModule(governanceKeeper),
	)

	// IMPROVED: Create upgrade manager
	upgradeManager := NewUpgradeManager(logger)

	// IMPROVED: Create rate limiter with default configuration
	rateLimiter := NewRateLimiter(DefaultRateLimitConfig())

	// Create app instance
	app := &VolnixApp{
		BaseApp:          bapp,
		appCodec:         encoding.Codec,
		keyParams:        keyParams,
		tkeyParams:       tkeyParams,
		keyBank:          keyBank,
		keyIdent:         keyIdent,
		keyLizenz:        keyLizenz,
		keyAnteil:        keyAnteil,
		keyConsensus:     keyConsensus,
		keyGovernance:    keyGovernance,
		paramsKeeper:     paramsKeeper,
		bankKeeper:       bankKeeper,
		identKeeper:      identKeeper,
		lizenzKeeper:     lizenzKeeper,
		anteilKeeper:     anteilKeeper,
		consensusKeeper:  consensusKeeper,
		governanceKeeper: governanceKeeper,
		mm:               mm,
		upgradeManager:   upgradeManager,
		rateLimiter:      rateLimiter,
	}

	// IMPROVED: Create snapshot manager after app is created
	app.snapshotManager = NewSnapshotManager(app)

	// Register upgrade handlers with app reference
	SetupUpgradeHandlers(upgradeManager, app)

	// Register interfaces first
	basicManager := module.NewBasicManager(
		bank.AppModuleBasic{},
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

	// IMPROVED: Use AnteHandler with rate limiting support
	bapp.SetAnteHandler(app.createAnteHandler())

	// Set BeginBlocker and EndBlocker for all modules
	bapp.SetBeginBlocker(func(ctx sdk.Context) (sdk.BeginBlock, error) {
		// IMPROVED: Check for upgrades at the beginning of each block
		if app.upgradeManager != nil {
			if err := app.upgradeManager.CheckUpgradeNeeded(ctx, app); err != nil {
				// Log error but don't fail the block
				logger.Error("Upgrade check failed", "error", err)
			}
		}

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

		// CRITICAL: Return validators in ResponseInitChain
		// CometBFT uses this to verify validator consistency during replay
		// If validators are not returned, CometBFT will see mismatch during replay
		// This is required for proper P2P authentication between validators
		validators := make([]abci.ValidatorUpdate, len(req.Validators))
		for i, val := range req.Validators {
			validators[i] = abci.ValidatorUpdate{
				PubKey: val.PubKey,
				Power:  val.Power,
			}
		}

		return &abci.ResponseInitChain{
			Validators:      validators,
			ConsensusParams: req.ConsensusParams,
			AppHash:         []byte{},
		}, nil
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

// GetGovernanceKeeper returns the governance keeper.
func (app *VolnixApp) GetGovernanceKeeper() *governancekeeper.Keeper {
	return app.governanceKeeper
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
	if app.snapshotManager == nil {
		return &abci.ResponseApplySnapshotChunk{
			Result: abci.ResponseApplySnapshotChunk_ACCEPT,
		}, nil
	}

	// Apply chunk
	chunkHash := fmt.Sprintf("%x", req.Chunk)
	if err := app.snapshotManager.ApplyChunk(req.Index, req.Chunk, chunkHash); err != nil {
		return &abci.ResponseApplySnapshotChunk{
			Result:        abci.ResponseApplySnapshotChunk_RETRY,
			RefetchChunks: []uint32{req.Index},
		}, err
	}

	// Check if all chunks are received
	// This is a simplified check - in production, you'd track which chunks are received
	return &abci.ResponseApplySnapshotChunk{
		Result: abci.ResponseApplySnapshotChunk_ACCEPT,
	}, nil
}

// LoadSnapshotChunk implements the ABCI interface with context
func (app *VolnixApp) LoadSnapshotChunk(ctx context.Context, req *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	if app.snapshotManager == nil {
		return &abci.ResponseLoadSnapshotChunk{}, nil
	}

	// Get snapshot
	snapshot, exists := app.snapshotManager.GetSnapshot(uint64(req.Height))
	if !exists {
		return &abci.ResponseLoadSnapshotChunk{}, fmt.Errorf("snapshot not found at height %d", req.Height)
	}

	// Get chunk by index
	if req.Chunk >= snapshot.ChunkCount {
		return &abci.ResponseLoadSnapshotChunk{}, fmt.Errorf("chunk index %d out of range (max %d)", req.Chunk, snapshot.ChunkCount-1)
	}

	chunkHash := snapshot.ChunkHashes[req.Chunk]
	chunk, exists := app.snapshotManager.GetChunk(chunkHash)
	if !exists {
		return &abci.ResponseLoadSnapshotChunk{}, fmt.Errorf("chunk %s not found", chunkHash)
	}

	return &abci.ResponseLoadSnapshotChunk{
		Chunk: chunk,
	}, nil
}

// createAnteHandler creates an AnteHandler with rate limiting support
func (app *VolnixApp) createAnteHandler() sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		// IMPROVED: Apply rate limiting first (before other validations)
		// Skip rate limiting for simulation and recheck transactions
		if !simulate && ctx.IsCheckTx() && app.rateLimiter != nil {
			if err := app.rateLimiter.Allow(ctx, tx); err != nil {
				return ctx, fmt.Errorf("rate limit check failed: %w", err)
			}
		}

		// Continue with standard validation
		return ImprovedAnteHandler(ctx, tx, simulate)
	}
}

// ListSnapshots implements the ABCI interface with context
func (app *VolnixApp) ListSnapshots(ctx context.Context, req *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	if app.snapshotManager == nil {
		return &abci.ResponseListSnapshots{}, nil
	}

	snapshots := app.snapshotManager.ListSnapshots()
	abciSnapshots := make([]*abci.Snapshot, 0, len(snapshots))

	for _, snapshot := range snapshots {
		abciSnapshots = append(abciSnapshots, &abci.Snapshot{
			Height:   uint64(snapshot.Height),
			Format:   snapshot.Format,
			Chunks:   snapshot.ChunkCount,
			Hash:     snapshot.Hash,
			Metadata: []byte{}, // Additional metadata can be added here
		})
	}

	return &abci.ResponseListSnapshots{
		Snapshots: abciSnapshots,
	}, nil
}

// OfferSnapshot implements the ABCI interface with context
func (app *VolnixApp) OfferSnapshot(ctx context.Context, req *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	if app.snapshotManager == nil {
		return &abci.ResponseOfferSnapshot{
			Result: abci.ResponseOfferSnapshot_REJECT,
		}, nil
	}

	// Extract chunk hashes from metadata if available
	// For now, we'll accept the snapshot and process chunks as they arrive
	return &abci.ResponseOfferSnapshot{
		Result: abci.ResponseOfferSnapshot_ACCEPT,
	}, nil
}
