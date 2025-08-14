package keeper

import (
	"fmt"

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

	// Parse amount (simplified validation)
	// In real implementation, this would parse sdk.Coin and validate
	if amount < params.MinLznAmount {
		return types.ErrBelowMinAmount
	}
	if amount > params.MaxLznAmount {
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
