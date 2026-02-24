package app

import (
	"context"
	"fmt"
	"sync"

	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	sdklog "cosmossdk.io/log"
)

// UpgradePlan represents an upgrade plan
type UpgradePlan struct {
	Name   string
	Height int64
	Info   string
}

// UpgradeHandler is a function that handles an upgrade
type UpgradeHandler func(ctx sdk.Context, plan UpgradePlan, app *VolnixApp) error

// UpgradeManager manages upgrade handlers and migrations
type UpgradeManager struct {
	handlers     map[string]UpgradeHandler
	pendingPlans map[int64]*UpgradePlan // height -> plan
	logger       sdklog.Logger
	mu           sync.RWMutex
}

var (
	upgradeManagerOnce sync.Once
	upgradeManagerMu   sync.RWMutex
)

// NewUpgradeManager creates a new upgrade manager
func NewUpgradeManager(logger sdklog.Logger) *UpgradeManager {
	return &UpgradeManager{
		handlers:     make(map[string]UpgradeHandler),
		pendingPlans: make(map[int64]*UpgradePlan),
		logger:       logger,
	}
}

// RegisterUpgradeHandler registers an upgrade handler for a specific version
func (um *UpgradeManager) RegisterUpgradeHandler(name string, handler UpgradeHandler) {
	um.handlers[name] = handler
	um.logger.Info("Registered upgrade handler", "name", name)
}

// GetUpgradeHandler returns the upgrade handler for a specific version
func (um *UpgradeManager) GetUpgradeHandler(name string) (UpgradeHandler, bool) {
	handler, exists := um.handlers[name]
	return handler, exists
}

// ExecuteUpgrade executes an upgrade plan
func (um *UpgradeManager) ExecuteUpgrade(ctx sdk.Context, plan UpgradePlan, app *VolnixApp) error {
	um.logger.Info("Executing upgrade", "name", plan.Name, "height", plan.Height, "info", plan.Info)
	
	handler, exists := um.handlers[plan.Name]
	if !exists {
		return fmt.Errorf("upgrade handler not found for version: %s", plan.Name)
	}
	
	if err := handler(ctx, plan, app); err != nil {
		um.logger.Error("Upgrade failed", "name", plan.Name, "error", err)
		return fmt.Errorf("upgrade execution failed: %w", err)
	}
	
	um.logger.Info("Upgrade completed successfully", "name", plan.Name)
	return nil
}

// SetupUpgradeHandlers registers all upgrade handlers for the application
func SetupUpgradeHandlers(um *UpgradeManager, app *VolnixApp) {
	// Register upgrade handlers for different versions
	// Example: v0.2.0 upgrade
	um.RegisterUpgradeHandler("v0.2.0", func(ctx sdk.Context, plan UpgradePlan, app *VolnixApp) error {
		return MigrateToV0_2_0(ctx, app)
	})
	
	// Example: v0.3.0 upgrade (commented out, ready to use)
	um.RegisterUpgradeHandler("v0.3.0", func(ctx sdk.Context, plan UpgradePlan, app *VolnixApp) error {
		return MigrateToV0_3_0(ctx, app)
	})
	
	// Add more upgrade handlers as needed
}

// SetupSDKUpgradeHandlers registers handlers in the standard Cosmos SDK x/upgrade keeper.
// This path is used by the app runtime and supersedes the custom UpgradeManager scheduler.
func SetupSDKUpgradeHandlers(uk *upgradekeeper.Keeper, app *VolnixApp) {
	uk.SetUpgradeHandler("v0.2.0", func(goCtx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		if err := MigrateToV0_2_0(ctx, app); err != nil {
			return nil, err
		}
		return fromVM, nil
	})

	uk.SetUpgradeHandler("v0.3.0", func(goCtx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		if err := MigrateToV0_3_0(ctx, app); err != nil {
			return nil, err
		}
		return fromVM, nil
	})
}

// MigrateToV0_3_0 performs state migration to version 0.3.0
func MigrateToV0_3_0(ctx sdk.Context, app *VolnixApp) error {
	// Example: Migrate all modules
	if err := migrateIdentModuleV0_3_0(ctx, app); err != nil {
		return fmt.Errorf("ident module migration failed: %w", err)
	}
	
	if err := migrateLizenzModuleV0_3_0(ctx, app); err != nil {
		return fmt.Errorf("lizenz module migration failed: %w", err)
	}
	
	if err := migrateAnteilModuleV0_3_0(ctx, app); err != nil {
		return fmt.Errorf("anteil module migration failed: %w", err)
	}
	
	if err := migrateConsensusModuleV0_3_0(ctx, app); err != nil {
		return fmt.Errorf("consensus module migration failed: %w", err)
	}
	
	return nil
}

// migrateIdentModuleV0_3_0 migrates ident module to v0.3.0
// Re-saves all accounts so newly added proto fields get serialized with defaults.
func migrateIdentModuleV0_3_0(ctx sdk.Context, app *VolnixApp) error {
	accounts, err := app.identKeeper.GetAllVerifiedAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get verified accounts: %w", err)
	}

	migrated := 0
	for _, account := range accounts {
		if err := app.identKeeper.SetVerifiedAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to re-save account %s: %w", account.Address, err)
		}
		migrated++
	}

	ctx.Logger().Info("ident module migrated to v0.3.0", "accounts_migrated", migrated)
	return nil
}

// migrateLizenzModuleV0_3_0 migrates lizenz module to v0.3.0
// Re-saves all licenses so newly added proto fields get serialized with defaults.
func migrateLizenzModuleV0_3_0(ctx sdk.Context, app *VolnixApp) error {
	lizenzs, err := app.lizenzKeeper.GetAllActivatedLizenz(ctx)
	if err != nil {
		return fmt.Errorf("failed to get activated lizenzs: %w", err)
	}

	migrated := 0
	for _, lz := range lizenzs {
		if err := app.lizenzKeeper.SetActivatedLizenz(ctx, lz); err != nil {
			return fmt.Errorf("failed to re-save lizenz for %s: %w", lz.Validator, err)
		}
		migrated++
	}

	ctx.Logger().Info("lizenz module migrated to v0.3.0", "licenses_migrated", migrated)
	return nil
}

// migrateAnteilModuleV0_3_0 migrates anteil module to v0.3.0
// Updates module params to include new fields with defaults.
func migrateAnteilModuleV0_3_0(ctx sdk.Context, app *VolnixApp) error {
	params := app.anteilKeeper.GetParams(ctx)

	if params.StakingRewardRate == "" {
		params.StakingRewardRate = "0.05"
	}
	if params.LiquidityPoolFee == "" {
		params.LiquidityPoolFee = "0.003"
	}
	if params.MarketMakingBuyDiscount == "" {
		params.MarketMakingBuyDiscount = "0.99"
	}
	if params.MarketMakingSellPremium == "" {
		params.MarketMakingSellPremium = "1.01"
	}
	if params.MarketMakingOrderSize == "" {
		params.MarketMakingOrderSize = "1000.0"
	}

	app.anteilKeeper.SetParams(ctx, params)
	ctx.Logger().Info("anteil module migrated to v0.3.0")
	return nil
}

// migrateConsensusModuleV0_3_0 migrates consensus module to v0.3.0
// Re-saves all validators so newly added proto fields get serialized with defaults.
func migrateConsensusModuleV0_3_0(ctx sdk.Context, app *VolnixApp) error {
	validators := app.consensusKeeper.GetAllValidators(ctx)

	migrated := 0
	for _, v := range validators {
		app.consensusKeeper.SetValidator(ctx, v)
		migrated++
	}

	ctx.Logger().Info("consensus module migrated to v0.3.0", "validators_migrated", migrated)
	return nil
}

// MigrateToV0_2_0 performs state migration to version 0.2.0
// This is an example migration - customize based on actual needs
func MigrateToV0_2_0(ctx sdk.Context, app *VolnixApp) error {
	// Example: Migrate ident module
	if err := migrateIdentModuleV0_2_0(ctx, app); err != nil {
		return fmt.Errorf("ident module migration failed: %w", err)
	}
	
	// Example: Migrate lizenz module
	if err := migrateLizenzModuleV0_2_0(ctx, app); err != nil {
		return fmt.Errorf("lizenz module migration failed: %w", err)
	}
	
	// Add more module migrations as needed
	
	return nil
}

// migrateIdentModuleV0_2_0 migrates the ident module to v0.2.0
// Re-saves all accounts to populate newly added proto fields.
func migrateIdentModuleV0_2_0(ctx sdk.Context, app *VolnixApp) error {
	accounts, err := app.identKeeper.GetAllVerifiedAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get verified accounts: %w", err)
	}

	for _, account := range accounts {
		if err := app.identKeeper.SetVerifiedAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to re-save account %s: %w", account.Address, err)
		}
	}

	ctx.Logger().Info("ident module migrated to v0.2.0", "accounts", len(accounts))
	return nil
}

// migrateLizenzModuleV0_2_0 migrates the lizenz module to v0.2.0
// Re-saves all licenses to populate newly added proto fields.
func migrateLizenzModuleV0_2_0(ctx sdk.Context, app *VolnixApp) error {
	lizenzs, err := app.lizenzKeeper.GetAllActivatedLizenz(ctx)
	if err != nil {
		return fmt.Errorf("failed to get activated lizenzs: %w", err)
	}

	for _, lz := range lizenzs {
		if err := app.lizenzKeeper.SetActivatedLizenz(ctx, lz); err != nil {
			return fmt.Errorf("failed to re-save lizenz for %s: %w", lz.Validator, err)
		}
	}

	ctx.Logger().Info("lizenz module migrated to v0.2.0", "licenses", len(lizenzs))
	return nil
}

// ScheduleUpgrade schedules an upgrade at a specific height
func (um *UpgradeManager) ScheduleUpgrade(plan UpgradePlan) error {
	um.mu.Lock()
	defer um.mu.Unlock()
	
	// Check if upgrade handler exists
	if _, exists := um.handlers[plan.Name]; !exists {
		return fmt.Errorf("upgrade handler not found for version: %s", plan.Name)
	}
	
	// Check if there's already an upgrade scheduled at this height
	if existing, exists := um.pendingPlans[plan.Height]; exists {
		return fmt.Errorf("upgrade already scheduled at height %d: %s", plan.Height, existing.Name)
	}
	
	um.pendingPlans[plan.Height] = &plan
	um.logger.Info("Upgrade scheduled", "name", plan.Name, "height", plan.Height)
	
	return nil
}

// GetScheduledUpgrade returns a scheduled upgrade at a specific height
func (um *UpgradeManager) GetScheduledUpgrade(height int64) (*UpgradePlan, bool) {
	um.mu.RLock()
	defer um.mu.RUnlock()
	
	plan, exists := um.pendingPlans[height]
	return plan, exists
}

// CancelUpgrade cancels a scheduled upgrade
func (um *UpgradeManager) CancelUpgrade(height int64) error {
	um.mu.Lock()
	defer um.mu.Unlock()
	
	if _, exists := um.pendingPlans[height]; !exists {
		return fmt.Errorf("no upgrade scheduled at height %d", height)
	}
	
	delete(um.pendingPlans, height)
	um.logger.Info("Upgrade cancelled", "height", height)
	
	return nil
}

// CheckUpgradeNeeded checks if an upgrade is needed at the current block height
func (um *UpgradeManager) CheckUpgradeNeeded(ctx sdk.Context, app *VolnixApp) error {
	currentHeight := ctx.BlockHeight()
	
	um.mu.RLock()
	plan, exists := um.pendingPlans[currentHeight]
	um.mu.RUnlock()
	
	if exists && plan != nil {
		um.logger.Info("Upgrade triggered", "name", plan.Name, "height", currentHeight)
		
		if err := um.ExecuteUpgrade(ctx, *plan, app); err != nil {
			um.logger.Error("Upgrade execution failed", "error", err)
			return err
		}
		
		// Remove executed plan
		um.mu.Lock()
		delete(um.pendingPlans, currentHeight)
		um.mu.Unlock()
		
		um.logger.Info("Upgrade completed and removed from schedule", "name", plan.Name)
	}
	
	return nil
}

// GetPendingUpgrades returns all pending upgrade plans
func (um *UpgradeManager) GetPendingUpgrades() []*UpgradePlan {
	um.mu.RLock()
	defer um.mu.RUnlock()
	
	plans := make([]*UpgradePlan, 0, len(um.pendingPlans))
	for _, plan := range um.pendingPlans {
		plans = append(plans, plan)
	}
	
	return plans
}



