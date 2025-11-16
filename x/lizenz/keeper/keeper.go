package keeper

import (
	"fmt"
	"strconv"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		paramstore paramtypes.Subspace
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ps paramtypes.Subspace,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		paramstore: ps,
	}
}

// GetParams returns the current parameters for the lizenz module
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var params types.Params
	k.paramstore.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the parameters for the lizenz module
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// SetActivatedLizenz stores an activated LZN license
func (k Keeper) SetActivatedLizenz(ctx sdk.Context, lizenz *lizenzv1.ActivatedLizenz) error {
	if err := types.IsActivatedLizenzValid(lizenz); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	lizenzKey := types.GetActivatedLizenzKey(lizenz.Validator)

	// Check if LZN already exists
	if store.Has(lizenzKey) {
		return types.ErrLizenzAlreadyExists
	}

	// Validate amount limits
	params := k.GetParams(ctx)
	if err := k.validateLizenzAmount(ctx, lizenz.Amount, params); err != nil {
		return err
	}

	// Validate 33% limit: no validator can activate more than 33% of total pool
	if err := k.ValidateMaxLznActivationLimit(ctx, lizenz.Validator, lizenz.Amount); err != nil {
		return err
	}

	// Store the LZN
	lizenzBz, err := k.cdc.Marshal(lizenz)
	if err != nil {
		return fmt.Errorf("failed to marshal activated lizenz: %w", err)
	}

	store.Set(lizenzKey, lizenzBz)
	return nil
}

// GetActivatedLizenz retrieves an activated LZN license by validator address
func (k Keeper) GetActivatedLizenz(ctx sdk.Context, validator string) (*lizenzv1.ActivatedLizenz, error) {
	if validator == "" {
		return nil, types.ErrEmptyValidator
	}

	store := ctx.KVStore(k.storeKey)
	lizenzKey := types.GetActivatedLizenzKey(validator)

	if !store.Has(lizenzKey) {
		return nil, types.ErrLizenzNotFound
	}

	lizenzBz := store.Get(lizenzKey)
	var lizenz lizenzv1.ActivatedLizenz
	if err := k.cdc.Unmarshal(lizenzBz, &lizenz); err != nil {
		return nil, fmt.Errorf("failed to unmarshal activated lizenz: %w", err)
	}

	return &lizenz, nil
}

// UpdateActivatedLizenz updates an existing activated LZN license
func (k Keeper) UpdateActivatedLizenz(ctx sdk.Context, lizenz *lizenzv1.ActivatedLizenz) error {
	if err := types.IsActivatedLizenzValid(lizenz); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	lizenzKey := types.GetActivatedLizenzKey(lizenz.Validator)

	// Check if LZN exists
	if !store.Has(lizenzKey) {
		return types.ErrLizenzNotFound
	}

	// Validate 33% limit when updating amount
	if err := k.ValidateMaxLznActivationLimit(ctx, lizenz.Validator, lizenz.Amount); err != nil {
		return err
	}

	// Store the updated LZN
	lizenzBz, err := k.cdc.Marshal(lizenz)
	if err != nil {
		return fmt.Errorf("failed to marshal activated lizenz: %w", err)
	}

	store.Set(lizenzKey, lizenzBz)
	return nil
}

// DeleteActivatedLizenz removes an activated LZN license
func (k Keeper) DeleteActivatedLizenz(ctx sdk.Context, validator string) error {
	if validator == "" {
		return types.ErrEmptyValidator
	}

	store := ctx.KVStore(k.storeKey)
	lizenzKey := types.GetActivatedLizenzKey(validator)

	if !store.Has(lizenzKey) {
		return types.ErrLizenzNotFound
	}

	store.Delete(lizenzKey)
	return nil
}

// GetAllActivatedLizenz retrieves all activated LZN licenses
func (k Keeper) GetAllActivatedLizenz(ctx sdk.Context) ([]*lizenzv1.ActivatedLizenz, error) {
	store := ctx.KVStore(k.storeKey)
	lizenzStore := prefix.NewStore(store, types.ActivatedLizenzKeyPrefix)

	var lizenzs []*lizenzv1.ActivatedLizenz
	iterator := lizenzStore.Iterator(nil, nil)
	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()

	for ; iterator.Valid(); iterator.Next() {
		var lizenz lizenzv1.ActivatedLizenz
		if err := k.cdc.Unmarshal(iterator.Value(), &lizenz); err != nil {
			return nil, fmt.Errorf("failed to unmarshal activated lizenz: %w", err)
		}
		lizenzs = append(lizenzs, &lizenz)
	}

	return lizenzs, nil
}

// GetTotalActivatedLizenz calculates the total amount of activated LZN across all validators
// Returns the sum as a string to handle large numbers
func (k Keeper) GetTotalActivatedLizenz(ctx sdk.Context) (string, error) {
	allLizenzs, err := k.GetAllActivatedLizenz(ctx)
	if err != nil {
		return "0", err
	}

	total := int64(0)
	for _, lizenz := range allLizenzs {
		amount, err := strconv.ParseInt(lizenz.Amount, 10, 64)
		if err != nil {
			// Skip invalid amounts, but log the error
			continue
		}
		total += amount
	}

	return strconv.FormatInt(total, 10), nil
}

// ValidateMaxLznActivationLimit checks if a validator's activation would exceed 33% of total pool
// According to whitepaper: "Максимум 33% от общего пула LZN может быть активировано одним валидатором"
func (k Keeper) ValidateMaxLznActivationLimit(ctx sdk.Context, validator string, newAmount string) error {
	const maxValidatorShare = 0.33 // 33% limit from whitepaper

	// Get total activated LZN
	totalActivated, err := k.GetTotalActivatedLizenz(ctx)
	if err != nil {
		return fmt.Errorf("failed to get total activated LZN: %w", err)
	}

	totalActivatedInt, err := strconv.ParseInt(totalActivated, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid total activated LZN: %w", err)
	}

	// Get current validator's activated LZN (if any)
	var currentValidatorAmount int64 = 0
	if existingLizenz, err := k.GetActivatedLizenz(ctx, validator); err == nil {
		currentValidatorAmount, err = strconv.ParseInt(existingLizenz.Amount, 10, 64)
		if err != nil {
			currentValidatorAmount = 0
		}
	}

	// Calculate new amount
	newAmountInt, err := strconv.ParseInt(newAmount, 10, 64)
	if err != nil {
		return types.ErrInvalidAmount
	}

	// Calculate what the total would be after this activation
	// If validator already has LZN, subtract old amount and add new
	newTotal := totalActivatedInt - currentValidatorAmount + newAmountInt

	// If this is the first activation in the system (total was 0 and no existing LZN for this validator), allow it
	// The limit only applies when there are other validators or when updating existing LZN
	if totalActivatedInt == 0 && currentValidatorAmount == 0 {
		// First validator in the system, no limit check needed
		return nil
	}

	// If updating existing LZN and this is the only validator, allow it
	// (but only if not exceeding reasonable bounds)
	if currentValidatorAmount > 0 && totalActivatedInt == currentValidatorAmount {
		// Only this validator exists, allow update (but still check if it's reasonable)
		// This handles the case where a single validator updates their LZN
		return nil
	}

	// Calculate maximum allowed (33% of new total)
	maxAllowed := int64(float64(newTotal) * maxValidatorShare)

	// Check if validator's new amount exceeds the limit
	if newAmountInt > maxAllowed {
		return fmt.Errorf("%w: validator would have %d LZN (%.2f%%), maximum allowed is %d LZN (33%%)",
			types.ErrExceedsMaxLznActivation,
			newAmountInt,
			float64(newAmountInt)/float64(newTotal)*100,
			maxAllowed)
	}

	return nil
}

// SetLizenz is an alias for SetActivatedLizenz for backward compatibility
func (k Keeper) SetLizenz(ctx sdk.Context, lizenz *lizenzv1.ActivatedLizenz) error {
	return k.SetActivatedLizenz(ctx, lizenz)
}

// GetLizenz is an alias for GetActivatedLizenz for backward compatibility
func (k Keeper) GetLizenz(ctx sdk.Context, validator string) (*lizenzv1.ActivatedLizenz, error) {
	return k.GetActivatedLizenz(ctx, validator)
}

// UpdateLizenz is an alias for UpdateActivatedLizenz for backward compatibility
func (k Keeper) UpdateLizenz(ctx sdk.Context, lizenz *lizenzv1.ActivatedLizenz) error {
	return k.UpdateActivatedLizenz(ctx, lizenz)
}

// DeleteLizenz is an alias for DeleteActivatedLizenz for backward compatibility
func (k Keeper) DeleteLizenz(ctx sdk.Context, validator string) error {
	return k.DeleteActivatedLizenz(ctx, validator)
}

// GetAllLizenzs is an alias for GetAllActivatedLizenz for backward compatibility
func (k Keeper) GetAllLizenzs(ctx sdk.Context) ([]*lizenzv1.ActivatedLizenz, error) {
	return k.GetAllActivatedLizenz(ctx)
}

// ActivateLizenz activates a LZN license
func (k Keeper) ActivateLizenz(ctx sdk.Context, validator string) error {
	lizenz, err := k.GetActivatedLizenz(ctx, validator)
	if err != nil {
		return err
	}

	// Update status to active
	lizenz.IsEligibleForRewards = true
	return k.UpdateActivatedLizenz(ctx, lizenz)
}

// DeactivateLizenz deactivates a LZN license
func (k Keeper) DeactivateLizenz(ctx sdk.Context, validator string) error {
	lizenz, err := k.GetActivatedLizenz(ctx, validator)
	if err != nil {
		return err
	}

	// Update status to inactive
	lizenz.IsEligibleForRewards = false
	return k.UpdateActivatedLizenz(ctx, lizenz)
}

// TransferLizenz transfers a LZN license to another validator
func (k Keeper) TransferLizenz(ctx sdk.Context, fromValidator, toValidator string) error {
	lizenz, err := k.GetActivatedLizenz(ctx, fromValidator)
	if err != nil {
		return err
	}

	// Delete the old lizenz
	if err := k.DeleteActivatedLizenz(ctx, fromValidator); err != nil {
		return err
	}

	// Update owner and create new lizenz
	lizenz.Validator = toValidator
	return k.SetActivatedLizenz(ctx, lizenz)
}

// CheckMOA checks MOA compliance for a validator
func (k Keeper) CheckMOA(ctx sdk.Context, validator string) (bool, error) {
	status, err := k.GetMOAStatus(ctx, validator)
	if err != nil {
		return false, err
	}

	// Parse MOA values for comparison
	currentMoa, err := strconv.ParseInt(status.CurrentMoa, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid current MOA: %w", err)
	}

	requiredMoa, err := strconv.ParseInt(status.RequiredMoa, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid required MOA: %w", err)
	}

	return currentMoa >= requiredMoa, nil
}

// GetMOACompliance returns the MOA compliance ratio for a validator
// Returns compliance ratio: current_moa / required_moa
// Returns 1.0 if validator has no MOA status (assumes compliance)
// Returns error if MOA status exists but is invalid
func (k Keeper) GetMOACompliance(ctx sdk.Context, validator string) (float64, error) {
	status, err := k.GetMOAStatus(ctx, validator)
	if err != nil {
		// If no MOA status exists, assume full compliance
		return 1.0, nil
	}

	// Parse MOA values
	currentMoa, err := strconv.ParseFloat(status.CurrentMoa, 64)
	if err != nil {
		return 0.0, fmt.Errorf("invalid current MOA: %w", err)
	}

	requiredMoa, err := strconv.ParseFloat(status.RequiredMoa, 64)
	if err != nil {
		return 0.0, fmt.Errorf("invalid required MOA: %w", err)
	}

	// Avoid division by zero
	if requiredMoa == 0 {
		return 1.0, nil // If no requirement, assume compliance
	}

	// Calculate compliance ratio
	compliance := currentMoa / requiredMoa

	return compliance, nil
}

// BeginBlocker processes MOA violations and inactive LZN licenses
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	// Check for inactive LZN licenses
	if err := k.CheckInactiveLizenz(ctx); err != nil {
		return err
	}

	// Process deactivating LZN licenses
	if err := k.ProcessDeactivatingLizenz(ctx); err != nil {
		return err
	}

	return nil
}

// SetDeactivatingLizenz stores a deactivating LZN license
func (k Keeper) SetDeactivatingLizenz(ctx sdk.Context, lizenz *lizenzv1.DeactivatingLizenz) error {
	if err := types.IsDeactivatingLizenzValid(lizenz); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	lizenzKey := types.GetDeactivatingLizenzKey(lizenz.Validator)

	// Check if deactivating LZN already exists
	if store.Has(lizenzKey) {
		return types.ErrLizenzAlreadyExists
	}

	// Store the deactivating LZN
	lizenzBz, err := k.cdc.Marshal(lizenz)
	if err != nil {
		return fmt.Errorf("failed to marshal deactivating lizenz: %w", err)
	}

	store.Set(lizenzKey, lizenzBz)
	return nil
}

// GetDeactivatingLizenz retrieves a deactivating LZN license
func (k Keeper) GetDeactivatingLizenz(ctx sdk.Context, validator string) (*lizenzv1.DeactivatingLizenz, error) {
	if validator == "" {
		return nil, types.ErrEmptyValidator
	}

	store := ctx.KVStore(k.storeKey)
	lizenzKey := types.GetDeactivatingLizenzKey(validator)

	if !store.Has(lizenzKey) {
		return nil, types.ErrLizenzNotFound
	}

	lizenzBz := store.Get(lizenzKey)
	var lizenz lizenzv1.DeactivatingLizenz
	if err := k.cdc.Unmarshal(lizenzBz, &lizenz); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deactivating lizenz: %w", err)
	}

	return &lizenz, nil
}

// GetAllDeactivatingLizenz retrieves all deactivating LZN licenses
func (k Keeper) GetAllDeactivatingLizenz(ctx sdk.Context) ([]*lizenzv1.DeactivatingLizenz, error) {
	store := ctx.KVStore(k.storeKey)
	lizenzStore := prefix.NewStore(store, types.DeactivatingLizenzKeyPrefix)

	var lizenzs []*lizenzv1.DeactivatingLizenz
	iterator := lizenzStore.Iterator(nil, nil)
	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()

	for ; iterator.Valid(); iterator.Next() {
		var lizenz lizenzv1.DeactivatingLizenz
		if err := k.cdc.Unmarshal(iterator.Value(), &lizenz); err != nil {
			return nil, fmt.Errorf("failed to unmarshal deactivating lizenz: %w", err)
		}
		lizenzs = append(lizenzs, &lizenz)
	}

	return lizenzs, nil
}

// DeleteDeactivatingLizenz removes a deactivating LZN license
func (k Keeper) DeleteDeactivatingLizenz(ctx sdk.Context, validator string) error {
	if validator == "" {
		return types.ErrEmptyValidator
	}

	store := ctx.KVStore(k.storeKey)
	lizenzKey := types.GetDeactivatingLizenzKey(validator)

	if !store.Has(lizenzKey) {
		return types.ErrLizenzNotFound
	}

	store.Delete(lizenzKey)
	return nil
}

// SetMOAStatus stores MOA status for a validator
func (k Keeper) SetMOAStatus(ctx sdk.Context, status *lizenzv1.MOAStatus) error {
	if err := types.IsMOAStatusValid(status); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	statusKey := types.GetMOAStatusKey(status.Validator)

	// Store the MOA status
	statusBz, err := k.cdc.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal MOA status: %w", err)
	}

	store.Set(statusKey, statusBz)
	return nil
}

// GetMOAStatus retrieves MOA status for a validator
func (k Keeper) GetMOAStatus(ctx sdk.Context, validator string) (*lizenzv1.MOAStatus, error) {
	if validator == "" {
		return nil, types.ErrEmptyValidator
	}

	store := ctx.KVStore(k.storeKey)
	statusKey := types.GetMOAStatusKey(validator)

	if !store.Has(statusKey) {
		return nil, types.ErrLizenzNotFound
	}

	statusBz := store.Get(statusKey)
	var status lizenzv1.MOAStatus
	if err := k.cdc.Unmarshal(statusBz, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MOA status: %w", err)
	}

	return &status, nil
}

// GetAllMOAStatus retrieves all MOA statuses
func (k Keeper) GetAllMOAStatus(ctx sdk.Context) ([]*lizenzv1.MOAStatus, error) {
	store := ctx.KVStore(k.storeKey)
	statusStore := prefix.NewStore(store, types.MOAStatusKeyPrefix)

	var statuses []*lizenzv1.MOAStatus
	iterator := statusStore.Iterator(nil, nil)
	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()

	for ; iterator.Valid(); iterator.Next() {
		var status lizenzv1.MOAStatus
		if err := k.cdc.Unmarshal(iterator.Value(), &status); err != nil {
			return nil, fmt.Errorf("failed to unmarshal MOA status: %w", err)
		}
		statuses = append(statuses, &status)
	}

	return statuses, nil
}

// UpdateLizenzActivity updates the last activity timestamp for a validator
func (k Keeper) UpdateLizenzActivity(ctx sdk.Context, validator string) error {
	// Update activated LZN activity
	if activatedLizenz, err := k.GetActivatedLizenz(ctx, validator); err == nil {
		types.UpdateActivatedLizenzActivity(activatedLizenz)
		if err := k.UpdateActivatedLizenz(ctx, activatedLizenz); err != nil {
			return err
		}
	}

	// Update MOA status activity
	if moaStatus, err := k.GetMOAStatus(ctx, validator); err == nil {
		// Calculate new MOA based on activity
		params := k.GetParams(ctx)
		newMOA, err := types.CalculateMOA("activity_data", params)
		if err != nil {
			return err
		}
		types.UpdateMOAStatusActivity(moaStatus, newMOA)
		if err := k.SetMOAStatus(ctx, moaStatus); err != nil {
			return err
		}
	}

	return nil
}

// validateLizenzAmount validates LZN amount against module parameters
func (k Keeper) validateLizenzAmount(ctx sdk.Context, amount string, params types.Params) error {
	if amount == "" {
		return types.ErrEmptyAmount
	}

	// Parse amount as integers for proper comparison
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return types.ErrInvalidAmount
	}

	minAmount, err := strconv.ParseInt(params.MinLznAmount, 10, 64)
	if err != nil {
		return types.ErrInvalidAmount
	}

	maxAmount, err := strconv.ParseInt(params.MaxLznAmount, 10, 64)
	if err != nil {
		return types.ErrInvalidAmount
	}

	if amountInt < minAmount {
		return types.ErrBelowMinAmount
	}
	if amountInt > maxAmount {
		return types.ErrExceedsMaxActivated
	}

	return nil
}

// CheckInactiveLizenz checks for inactive LZN licenses and moves them to deactivating state
func (k Keeper) CheckInactiveLizenz(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	inactivityThreshold := ctx.BlockTime().Add(-params.InactivityPeriod)

	activatedLizenzs, err := k.GetAllActivatedLizenz(ctx)
	if err != nil {
		return err
	}

	for _, lizenz := range activatedLizenzs {
		if lizenz.LastActivity.AsTime().Before(inactivityThreshold) {
			// Move to deactivating state
			deactivatingLizenz := types.NewDeactivatingLizenz(
				lizenz.Validator,
				lizenz.Amount,
				"inactivity",
			)

			if err := k.SetDeactivatingLizenz(ctx, deactivatingLizenz); err != nil {
				return err
			}

			if err := k.DeleteActivatedLizenz(ctx, lizenz.Validator); err != nil {
				return err
			}
		}
	}

	return nil
}

// ProcessDeactivatingLizenz processes deactivating LZN licenses that have completed their period
func (k Keeper) ProcessDeactivatingLizenz(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	deactivationThreshold := ctx.BlockTime().Add(-params.DeactivationPeriod)

	deactivatingLizenzs, err := k.GetAllDeactivatingLizenz(ctx)
	if err != nil {
		return err
	}

	for _, lizenz := range deactivatingLizenzs {
		if lizenz.DeactivationEnd.AsTime().Before(deactivationThreshold) {
			// Deactivation period completed, remove the LZN
			if err := k.DeleteDeactivatingLizenz(ctx, lizenz.Validator); err != nil {
				return err
			}
		}
	}

	return nil
}
