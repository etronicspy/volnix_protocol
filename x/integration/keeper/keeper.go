package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	consensuskeeper "github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/integration/types"
	lizenzkeeper "github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
)

// Keeper manages the integration between modules
type Keeper struct {
	identKeeper     identkeeper.Keeper
	lizenzKeeper    lizenzkeeper.Keeper
	anteilKeeper    anteilkeeper.Keeper
	consensusKeeper consensuskeeper.Keeper

	integrationManager *types.IntegrationManager
}

// NewKeeper creates a new integration keeper
func NewKeeper(
	identKeeper identkeeper.Keeper,
	lizenzKeeper lizenzkeeper.Keeper,
	anteilKeeper anteilkeeper.Keeper,
	consensusKeeper consensuskeeper.Keeper,
) *Keeper {

	im := types.NewIntegrationManager()

	// Register all modules with their dependencies
	im.RegisterModule("ident", []string{})
	im.RegisterModule("lizenz", []string{"ident"})
	im.RegisterModule("anteil", []string{"ident", "lizenz"})
	im.RegisterModule("consensus", []string{"ident", "lizenz", "anteil"})

	return &Keeper{
		identKeeper:        identKeeper,
		lizenzKeeper:       lizenzKeeper,
		anteilKeeper:       anteilKeeper,
		consensusKeeper:    consensusKeeper,
		integrationManager: im,
	}
}

// GetValidatorIntegrationStatus gets the complete integration status for a validator
func (k Keeper) GetValidatorIntegrationStatus(ctx sdk.Context, validator string) (*types.ValidatorIntegrationStatus, error) {

	// Get status from each module
	identAccount, err := k.identKeeper.GetVerifiedAccount(ctx, validator)
	if err != nil {
		// Log error but continue
		k.integrationManager.UpdateModuleHealth("ident", 50, err.Error())
	}

	lizenzLicense, err := k.lizenzKeeper.GetActivatedLizenz(ctx, validator)
	if err != nil {
		k.integrationManager.UpdateModuleHealth("lizenz", 50, err.Error())
	}

	// Get user position from anteil module
	anteilPosition, err := k.getAnteilUserPosition(ctx, validator)
	if err != nil {
		// Log error but continue with nil position
		ctx.Logger().Error("failed to get anteil position", "validator", validator, "error", err)
		anteilPosition = nil
	}
	// anteilPosition, err := k.anteilKeeper.GetUserPosition(ctx, validator)
	// if err != nil {
	//	k.integrationManager.UpdateModuleHealth("anteil", 50, err.Error())
	// }

	consensusValidator, err := k.consensusKeeper.GetValidator(ctx, validator)
	if consensusValidator == nil {
		k.integrationManager.UpdateModuleHealth("consensus", 50, "validator not found")
	}

	// Create integration status
	status := types.GetValidatorIntegrationStatus(
		validator,
		identAccount,
		lizenzLicense,
		anteilPosition, // User position from anteil module
		consensusValidator,
	)

	// Log cross-module event
	k.integrationManager.AddCrossModuleEvent(
		"validator_status_check",
		"integration",
		"all",
		fmt.Sprintf("Status check for validator %s", validator),
		validator,
	)

	return status, nil
}

// ValidateCrossModuleOperation validates operations that affect multiple modules
func (k Keeper) ValidateCrossModuleOperation(ctx sdk.Context, operation string, validator string) error {

	// Get current integration status
	status, err := k.GetValidatorIntegrationStatus(ctx, validator)
	if err != nil {
		return fmt.Errorf("failed to get integration status: %w", err)
	}

	// Validate based on operation type
	switch operation {
	case "consensus_participation":
		if status.IdentStatus == nil || !status.IdentStatus.IsActive {
			return fmt.Errorf("validator %s must have active identity verification", validator)
		}
		if status.LizenzStatus == nil {
			return fmt.Errorf("validator %s must have activated LZN license", validator)
		}

	case "ant_market_access":
		if status.IdentStatus == nil || !status.IdentStatus.IsActive {
			return fmt.Errorf("validator %s must have active identity verification", validator)
		}
		if status.LizenzStatus == nil {
			return fmt.Errorf("validator %s must have activated LZN license", validator)
		}

	case "role_migration":
		if status.IdentStatus == nil || !status.IdentStatus.IsActive {
			return fmt.Errorf("validator %s must have active identity verification", validator)
		}

	default:
		return fmt.Errorf("unknown operation type: %s", operation)
	}

	return nil
}

// ProcessCrossModuleEvent processes events that affect multiple modules
func (k Keeper) ProcessCrossModuleEvent(ctx sdk.Context, event *types.CrossModuleEvent) error {

	// Log the event
	k.integrationManager.AddCrossModuleEvent(
		event.EventType,
		event.SourceModule,
		event.TargetModule,
		event.EventData,
		event.Validator,
	)

	// Process based on event type
	switch event.EventType {
	case "identity_verified":
		// Update related modules when identity is verified
		return k.handleIdentityVerified(ctx, event.Validator)

	case "lzn_activated":
		// Update related modules when LZN is activated
		return k.handleLizenzActivated(ctx, event.Validator)

	case "consensus_participation":
		// Update related modules when consensus participation changes
		return k.handleConsensusParticipation(ctx, event.Validator)

	default:
		// Unknown event type, just log it
		return nil
	}
}

// handleIdentityVerified handles identity verification events
func (k Keeper) handleIdentityVerified(ctx sdk.Context, validator string) error {

	// Update integration manager health
	k.integrationManager.UpdateModuleHealth("ident", 100, "")

	// Log cross-module event
	k.integrationManager.AddCrossModuleEvent(
		"identity_verified_processed",
		"integration",
		"ident",
		"Identity verification processed",
		validator,
	)

	return nil
}

// handleLizenzActivated handles LZN activation events
func (k Keeper) handleLizenzActivated(ctx sdk.Context, validator string) error {

	// Update integration manager health
	k.integrationManager.UpdateModuleHealth("lizenz", 100, "")

	// Log cross-module event
	k.integrationManager.AddCrossModuleEvent(
		"lzn_activated_processed",
		"integration",
		"lizenz",
		"LZN activation processed",
		validator,
	)

	return nil
}

// handleConsensusParticipation handles consensus participation events
func (k Keeper) handleConsensusParticipation(ctx sdk.Context, validator string) error {

	// Update integration manager health
	k.integrationManager.UpdateModuleHealth("consensus", 100, "")

	// Log cross-module event
	k.integrationManager.AddCrossModuleEvent(
		"consensus_participation_processed",
		"integration",
		"consensus",
		"Consensus participation processed",
		validator,
	)

	return nil
}

// GetIntegrationManager returns the integration manager
func (k Keeper) GetIntegrationManager() *types.IntegrationManager {
	return k.integrationManager
}

// GetModuleHealth returns the health status of a specific module
func (k Keeper) GetModuleHealth(moduleName string) (*types.ModuleIntegration, error) {
	if module, exists := k.integrationManager.Modules[moduleName]; exists {
		return module, nil
	}
	return nil, fmt.Errorf("module %s not found", moduleName)
}

// GetAllModulesHealth returns the health status of all modules
func (k Keeper) GetAllModulesHealth() map[string]*types.ModuleIntegration {
	return k.integrationManager.Modules
}
// getAnteilUserPosition retrieves user position from anteil module
func (k Keeper) getAnteilUserPosition(ctx sdk.Context, userAddress string) (*anteilv1.UserPosition, error) {
	// Create a mock user position for now
	// In production, this would query the actual anteil keeper
	position := &anteilv1.UserPosition{
		User:           userAddress,
		TotalAntAmount: "1000.0",
		AvgBuyPrice:    "1.5",
		TotalValue:     "1500.0",
		LastUpdated:    ctx.BlockTime().Unix(),
	}
	
	return position, nil
}