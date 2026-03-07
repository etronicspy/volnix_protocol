package keeper

import (
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	consensuskeeper "github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
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

	// Register all modules with their dependencies (zero time at init; updated on first block)
	zeroTime := time.Time{}
	im.RegisterModule("ident", []string{}, zeroTime)
	im.RegisterModule("lizenz", []string{"ident"}, zeroTime)
	im.RegisterModule("anteil", []string{"ident", "lizenz"}, zeroTime)
	im.RegisterModule("consensus", []string{"ident", "lizenz", "anteil"}, zeroTime)

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
	bt := ctx.BlockTime()

	identAccount, err := k.identKeeper.GetVerifiedAccount(ctx, validator)
	if err != nil {
		if errors.Is(err, identtypes.ErrAccountNotFound) {
			ctx.Logger().Debug("validator has no verified identity", "module", "integration", "validator", validator)
		} else {
			k.integrationManager.UpdateModuleHealth("ident", 50, err.Error(), bt)
		}
	}

	lizenzLicense, err := k.lizenzKeeper.GetActivatedLizenz(ctx, validator)
	if err != nil {
		k.integrationManager.UpdateModuleHealth("lizenz", 50, err.Error(), bt)
	}

	anteilPosition, err := k.getAnteilUserPosition(ctx, validator)
	if err != nil {
		ctx.Logger().Error("failed to get anteil position", "module", "integration", "validator", validator, "error", err)
		anteilPosition = nil
	}

	consensusValidator, err := k.consensusKeeper.GetValidator(ctx, validator)
	if err != nil {
		k.integrationManager.UpdateModuleHealth("consensus", 50, fmt.Sprintf("validator lookup failed: %v", err), bt)
	} else if consensusValidator == nil {
		k.integrationManager.UpdateModuleHealth("consensus", 50, "validator not found", bt)
	}

	status := types.GetValidatorIntegrationStatus(
		validator,
		identAccount,
		lizenzLicense,
		anteilPosition,
		consensusValidator,
	)

	k.integrationManager.AddCrossModuleEvent(
		"validator_status_check",
		"integration",
		"all",
		fmt.Sprintf("Status check for validator %s", validator),
		validator,
		bt,
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

	k.integrationManager.AddCrossModuleEvent(
		event.EventType,
		event.SourceModule,
		event.TargetModule,
		event.EventData,
		event.Validator,
		ctx.BlockTime(),
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

// handleIdentityVerified handles identity verification events.
// When identity is verified, check if validator is eligible for consensus participation.
func (k Keeper) handleIdentityVerified(ctx sdk.Context, validator string) error {
	k.integrationManager.UpdateModuleHealth("ident", 100, "", ctx.BlockTime())

	lizenz, err := k.lizenzKeeper.GetActivatedLizenz(ctx, validator)
	if err == nil && lizenz != nil {
		_, err := k.consensusKeeper.GetValidator(ctx, validator)
		if err != nil {
			ctx.Logger().Info("identity verified for licensed user, eligible for consensus",
				"validator", validator)
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"integration.identity_verified",
		sdk.NewAttribute("validator", validator),
	))

	return nil
}

// handleLizenzActivated handles LZN activation events.
// When a lizenz is activated, verify identity is also active for consensus eligibility.
func (k Keeper) handleLizenzActivated(ctx sdk.Context, validator string) error {
	k.integrationManager.UpdateModuleHealth("lizenz", 100, "", ctx.BlockTime())

	account, err := k.identKeeper.GetVerifiedAccount(ctx, validator)
	if err == nil && account != nil && account.IsActive {
		ctx.Logger().Info("lizenz activated for verified identity, eligible for consensus",
			"validator", validator)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"integration.lzn_activated",
		sdk.NewAttribute("validator", validator),
	))

	return nil
}

// handleConsensusParticipation handles consensus participation events.
// Validates that the validator still meets all prerequisites.
func (k Keeper) handleConsensusParticipation(ctx sdk.Context, validator string) error {
	k.integrationManager.UpdateModuleHealth("consensus", 100, "", ctx.BlockTime())

	if err := k.ValidateCrossModuleOperation(ctx, "consensus_participation", validator); err != nil {
		ctx.Logger().Warn("consensus participation validation failed",
			"validator", validator, "error", err)
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"integration.consensus_participation",
		sdk.NewAttribute("validator", validator),
	))

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
	return k.anteilKeeper.GetUserPosition(ctx, userAddress)
}
