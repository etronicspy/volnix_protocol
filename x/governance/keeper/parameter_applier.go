package keeper

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/volnix-protocol/volnix-protocol/x/governance/types"
)

// ParameterChange is aliased from governancev1.ParameterChange in keeper.go
// This file uses it for applying changes

// ModuleKeeperInterface is deprecated - use specific keeper interfaces instead
// Kept for backward compatibility
type ModuleKeeperInterface interface {
	GetParams(ctx sdk.Context) interface{} // Returns module-specific params
	SetParams(ctx sdk.Context, params interface{}) error // Sets module-specific params
}

// ApplyParameterChange applies a parameter change to the appropriate module
// According to whitepaper: only governable parameters can be changed
func (k Keeper) ApplyParameterChange(ctx sdk.Context, change *ParameterChange) error {
	// Validate that parameter is governable
	if !types.IsGovernable(change.Module, change.Parameter) {
		return types.ErrConstitutionalParameter
	}

	// Apply change based on module
	switch change.Module {
	case "lizenz":
		return k.applyLizenzParameterChange(ctx, change)
	case "anteil":
		return k.applyAnteilParameterChange(ctx, change)
	case "consensus":
		return k.applyConsensusParameterChange(ctx, change)
	case "governance":
		return k.applyGovernanceParameterChange(ctx, change)
	default:
		return fmt.Errorf("unknown module: %s", change.Module)
	}
}

// applyLizenzParameterChange applies a parameter change to lizenz module
func (k Keeper) applyLizenzParameterChange(ctx sdk.Context, change *ParameterChange) error {
	if k.lizenzKeeper == nil {
		return fmt.Errorf("lizenz keeper not set")
	}

	// Get current params
	currentParams := k.lizenzKeeper.GetParams(ctx)

	// Apply change based on parameter name
	switch change.Parameter {
	case "activity_coefficient":
		currentParams.ActivityCoefficient = change.NewValue
	case "min_lzn_amount":
		currentParams.MinLznAmount = change.NewValue
	case "max_lzn_amount":
		currentParams.MaxLznAmount = change.NewValue
	case "inactivity_period":
		// Parse duration string (e.g., "8760h" for 1 year)
		duration, err := time.ParseDuration(change.NewValue)
		if err != nil {
			return fmt.Errorf("invalid duration format: %w", err)
		}
		currentParams.InactivityPeriod = duration
	case "deactivation_period":
		duration, err := time.ParseDuration(change.NewValue)
		if err != nil {
			return fmt.Errorf("invalid duration format: %w", err)
		}
		currentParams.DeactivationPeriod = duration
	default:
		return fmt.Errorf("unknown lizenz parameter: %s", change.Parameter)
	}

	// Update params
	k.lizenzKeeper.SetParams(ctx, currentParams)

	ctx.Logger().Info("lizenz parameter updated successfully",
		"parameter", change.Parameter,
		"old_value", change.OldValue,
		"new_value", change.NewValue)

	return nil
}

// applyAnteilParameterChange applies a parameter change to anteil module
func (k Keeper) applyAnteilParameterChange(ctx sdk.Context, change *ParameterChange) error {
	if k.anteilKeeper == nil {
		return fmt.Errorf("anteil keeper not set")
	}

	// Get current params
	currentParams := k.anteilKeeper.GetParams(ctx)

	// Apply change based on parameter name
	switch change.Parameter {
	case "min_ant_amount":
		currentParams.MinAntAmount = change.NewValue
	case "max_ant_amount":
		currentParams.MaxAntAmount = change.NewValue
	case "trading_fee_rate":
		currentParams.TradingFeeRate = change.NewValue
	case "max_open_orders":
		value, err := strconv.ParseUint(change.NewValue, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid uint32 value: %w", err)
		}
		currentParams.MaxOpenOrders = uint32(value)
	default:
		return fmt.Errorf("unknown anteil parameter: %s", change.Parameter)
	}

	// Update params
	k.anteilKeeper.SetParams(ctx, currentParams)

	ctx.Logger().Info("anteil parameter updated successfully",
		"parameter", change.Parameter,
		"old_value", change.OldValue,
		"new_value", change.NewValue)

	return nil
}

// applyConsensusParameterChange applies a parameter change to consensus module
func (k Keeper) applyConsensusParameterChange(ctx sdk.Context, change *ParameterChange) error {
	if k.consensusKeeper == nil {
		return fmt.Errorf("consensus keeper not set")
	}

	// Get current params
	currentParams := k.consensusKeeper.GetParams(ctx)

	// Apply change based on parameter name
	switch change.Parameter {
	case "base_block_time":
		// BaseBlockTime is stored as a string (e.g., "5s")
		// Validate it's a valid duration string
		_, err := time.ParseDuration(change.NewValue)
		if err != nil {
			return fmt.Errorf("invalid duration format: %w", err)
		}
		currentParams.BaseBlockTime = change.NewValue
	case "high_activity_threshold":
		// HighActivityThreshold is uint64
		value, err := strconv.ParseUint(change.NewValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint64 value: %w", err)
		}
		currentParams.HighActivityThreshold = value
	case "low_activity_threshold":
		// LowActivityThreshold is uint64
		value, err := strconv.ParseUint(change.NewValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint64 value: %w", err)
		}
		currentParams.LowActivityThreshold = value
	default:
		return fmt.Errorf("unknown consensus parameter: %s", change.Parameter)
	}

	// Update params
	k.consensusKeeper.SetParams(ctx, currentParams)

	ctx.Logger().Info("consensus parameter updated successfully",
		"parameter", change.Parameter,
		"old_value", change.OldValue,
		"new_value", change.NewValue)

	return nil
}

// applyGovernanceParameterChange applies a parameter change to governance module (meta-governance)
func (k Keeper) applyGovernanceParameterChange(ctx sdk.Context, change *ParameterChange) error {
	ctx.Logger().Info("applying governance parameter change",
		"parameter", change.Parameter,
		"old_value", change.OldValue,
		"new_value", change.NewValue)

	// Get current governance params
	currentGovParams := k.GetParams(ctx)

	// Apply change based on parameter name
	switch change.Parameter {
	case "voting_period":
		// Parse and update voting period
		duration, err := time.ParseDuration(change.NewValue)
		if err != nil {
			return fmt.Errorf("invalid duration format for voting_period: %w", err)
		}
		if duration <= 0 {
			return fmt.Errorf("voting_period must be positive")
		}
		currentGovParams.VotingPeriod = duration
	case "timelock_period":
		// Parse and update timelock period
		duration, err := time.ParseDuration(change.NewValue)
		if err != nil {
			return fmt.Errorf("invalid duration format for timelock_period: %w", err)
		}
		if duration <= 0 {
			return fmt.Errorf("timelock_period must be positive")
		}
		currentGovParams.TimelockPeriod = duration
	case "min_deposit":
		// Validate that min_deposit is a valid numeric string
		if _, err := strconv.ParseUint(change.NewValue, 10, 64); err != nil {
			return fmt.Errorf("invalid min_deposit value: %w", err)
		}
		currentGovParams.MinDeposit = change.NewValue
	case "quorum":
		// Validate that quorum is a valid decimal between 0 and 1
		quorum, err := strconv.ParseFloat(change.NewValue, 64)
		if err != nil {
			return fmt.Errorf("invalid quorum value: %w", err)
		}
		if quorum < 0 || quorum > 1 {
			return fmt.Errorf("quorum must be between 0 and 1")
		}
		currentGovParams.Quorum = change.NewValue
	case "threshold":
		// Validate that threshold is a valid decimal between 0 and 1
		threshold, err := strconv.ParseFloat(change.NewValue, 64)
		if err != nil {
			return fmt.Errorf("invalid threshold value: %w", err)
		}
		if threshold < 0 || threshold > 1 {
			return fmt.Errorf("threshold must be between 0 and 1")
		}
		currentGovParams.Threshold = change.NewValue
	default:
		return fmt.Errorf("unknown governance parameter: %s", change.Parameter)
	}

	// Update params
	k.SetParams(ctx, currentGovParams)

	ctx.Logger().Info("governance parameter updated successfully",
		"parameter", change.Parameter,
		"new_value", change.NewValue)

	return nil
}

// ValidateParameterChange validates a parameter change before applying it
func (k Keeper) ValidateParameterChange(ctx sdk.Context, change *ParameterChange) error {
	// Check if parameter is governable
	if !types.IsGovernable(change.Module, change.Parameter) {
		return types.ErrConstitutionalParameter
	}

	// Validate module name
	if change.Module == "" {
		return fmt.Errorf("module cannot be empty")
	}

	// Validate parameter name
	if change.Parameter == "" {
		return fmt.Errorf("parameter cannot be empty")
	}

	// Validate new value
	if change.NewValue == "" {
		return fmt.Errorf("new value cannot be empty")
	}

	// Additional validation based on parameter type
	govParam := types.GetGovernableParameter(change.Module, change.Parameter)
	if govParam != nil {
		switch govParam.Type {
		case "string":
			// String validation - just check it's not empty
			if change.NewValue == "" {
				return fmt.Errorf("string parameter cannot be empty")
			}
		case "uint64", "uint32":
			// Numeric validation - try to parse as uint64
			if _, err := strconv.ParseUint(change.NewValue, 10, 64); err != nil {
				return fmt.Errorf("invalid numeric value: %w", err)
			}
		case "duration":
			// Duration validation - parse and check it's positive
			if change.NewValue == "" {
				return fmt.Errorf("duration parameter cannot be empty")
			}
			duration, err := time.ParseDuration(change.NewValue)
			if err != nil {
				return fmt.Errorf("invalid duration format: %w", err)
			}
			if duration <= 0 {
				return fmt.Errorf("duration must be positive")
			}
		}
	}

	return nil
}

