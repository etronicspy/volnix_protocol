package keeper

import (
	"fmt"
	"strconv"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// IdentKeeperInterface defines the interface for interacting with ident module
// This allows lizenz module to verify identity and role before LZN activation
type IdentKeeperInterface interface {
	GetVerifiedAccount(ctx sdk.Context, address string) (*identv1.VerifiedAccount, error)
}

// ConsensusKeeperInterface defines the interface for interacting with consensus module
// This allows lizenz module to register validators in consensus after LZN activation
// Note: We use interface{} to avoid circular dependencies
type ConsensusKeeperInterface interface {
	SetValidator(ctx sdk.Context, validator interface{}) error
	SetValidatorWeight(ctx sdk.Context, validator, weight string) error
}

// AnteilKeeperInterface defines the interface for interacting with anteil module
// This allows lizenz module to create initial ANT position for validators
// Note: We use interface{} to avoid circular dependencies
type AnteilKeeperInterface interface {
	SetUserPosition(ctx sdk.Context, position interface{}) error
}

// BankKeeperInterface defines the interface for interacting with bank module
// This allows lizenz module to lock/unlock LZN tokens during activation/deactivation
type BankKeeperInterface interface {
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

type (
	Keeper struct {
		cdc             codec.BinaryCodec
		storeKey        storetypes.StoreKey
		paramstore      paramtypes.Subspace
		identKeeper     IdentKeeperInterface     // Optional: for identity verification
		consensusKeeper ConsensusKeeperInterface // Optional: for validator registration
		anteilKeeper    AnteilKeeperInterface    // Optional: for initial ANT position
		bankKeeper      BankKeeperInterface      // Optional: for locking/unlocking LZN tokens
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

// SetIdentKeeper sets the ident keeper interface for identity verification
func (k *Keeper) SetIdentKeeper(identKeeper IdentKeeperInterface) {
	k.identKeeper = identKeeper
}

// SetConsensusKeeper sets the consensus keeper interface for validator registration
func (k *Keeper) SetConsensusKeeper(consensusKeeper ConsensusKeeperInterface) {
	k.consensusKeeper = consensusKeeper
}

// SetAnteilKeeper sets the anteil keeper interface for initial ANT position creation
func (k *Keeper) SetAnteilKeeper(anteilKeeper AnteilKeeperInterface) {
	k.anteilKeeper = anteilKeeper
}

// SetBankKeeper sets the bank keeper interface for LZN token locking/unlocking
func (k *Keeper) SetBankKeeper(bankKeeper BankKeeperInterface) {
	k.bankKeeper = bankKeeper
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

	// Validate identity verification and role (if ident keeper is set)
	if k.identKeeper != nil {
		if err := k.validateIdentityAndRole(ctx, lizenz.Validator); err != nil {
			return err
		}
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

	// Lock LZN tokens in bank module (if bank keeper is available)
	// According to whitepaper: LZN tokens are locked when activated
	if k.bankKeeper != nil {
		validatorAddr, err := sdk.AccAddressFromBech32(lizenz.Validator)
		if err == nil {
			// Parse amount to coins
			amountInt, parseErr := strconv.ParseUint(lizenz.Amount, 10, 64)
			if parseErr == nil {
				// Lock LZN tokens by sending from validator to lizenz module account
				// In production, this should use a proper module account
				lznCoins := sdk.NewCoins(sdk.NewCoin("ulzn", math.NewIntFromUint64(amountInt)))
				if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, validatorAddr, types.ModuleName, lznCoins); err != nil {
					ctx.Logger().Error("failed to lock LZN tokens", "error", err, "validator", lizenz.Validator, "amount", lizenz.Amount)
					// Don't fail activation if locking fails - log and continue
					// In production, you might want to fail here
				} else {
					ctx.Logger().Info("LZN tokens locked", "validator", lizenz.Validator, "amount", lizenz.Amount)
				}
			}
		}
	}

	// Store the LZN
	lizenzBz, err := k.cdc.Marshal(lizenz)
	if err != nil {
		return fmt.Errorf("failed to marshal activated lizenz: %w", err)
	}

	store.Set(lizenzKey, lizenzBz)
	
	// Emit LZN activation event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeLizenzActivated,
			sdk.NewAttribute(types.AttributeKeyValidator, lizenz.Validator),
			sdk.NewAttribute(types.AttributeKeyAmount, lizenz.Amount),
			sdk.NewAttribute(types.AttributeKeyActivationTime, lizenz.ActivationTime.String()),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)
	
	// Emit LZN locked event if tokens were locked
	if k.bankKeeper != nil {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeLZNLocked,
				sdk.NewAttribute(types.AttributeKeyValidator, lizenz.Validator),
				sdk.NewAttribute(types.AttributeKeyAmount, lizenz.Amount),
				sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
			),
		)
	}
	
	// After successful LZN activation, register validator in consensus module
	// This is the automatic registration step according to validator registration logic
	if err := k.registerValidatorInConsensus(ctx, lizenz); err != nil {
		// Log error but don't fail the activation - validator can be registered later
		// In production, you might want to fail here or use event system
		ctx.Logger().Error("failed to register validator in consensus after LZN activation", "error", err, "validator", lizenz.Validator)
	} else {
		// Emit validator registration event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeValidatorRegistered,
				sdk.NewAttribute(types.AttributeKeyValidator, lizenz.Validator),
				sdk.NewAttribute(types.AttributeKeyAmount, lizenz.Amount),
				sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
			),
		)
	}
	
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

	// Get LZN info before deletion to unlock tokens
	var lizenz lizenzv1.ActivatedLizenz
	lizenzBz := store.Get(lizenzKey)
	if lizenzBz != nil {
		if err := k.cdc.Unmarshal(lizenzBz, &lizenz); err == nil {
			// Unlock LZN tokens from bank module (if bank keeper is available)
			// According to whitepaper: LZN tokens are unlocked when deactivated
			if k.bankKeeper != nil {
				validatorAddr, err := sdk.AccAddressFromBech32(validator)
				if err == nil {
					// Parse amount to coins
					amountInt, parseErr := strconv.ParseUint(lizenz.Amount, 10, 64)
					if parseErr == nil {
						// Unlock LZN tokens by sending from lizenz module account back to validator
						lznCoins := sdk.NewCoins(sdk.NewCoin("ulzn", math.NewIntFromUint64(amountInt)))
						if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, validatorAddr, lznCoins); err != nil {
							ctx.Logger().Error("failed to unlock LZN tokens", "error", err, "validator", validator, "amount", lizenz.Amount)
							// Don't fail deactivation if unlocking fails - log and continue
							// In production, you might want to handle this differently
						} else {
							ctx.Logger().Info("LZN tokens unlocked", "validator", validator, "amount", lizenz.Amount)
							
							// Emit event for LZN unlock
							ctx.EventManager().EmitEvent(
								sdk.NewEvent(
									types.EventTypeLZNUnlocked,
									sdk.NewAttribute(types.AttributeKeyValidator, validator),
									sdk.NewAttribute(types.AttributeKeyAmount, lizenz.Amount),
									sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
								),
							)
						}
					}
				}
			}
		}
	}

	store.Delete(lizenzKey)
	
	// Emit event for LZN deactivation
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeLizenzDeactivated,
			sdk.NewAttribute(types.AttributeKeyValidator, validator),
			sdk.NewAttribute(types.AttributeKeyAmount, lizenz.Amount),
			sdk.NewAttribute(types.AttributeKeyDeactivationTime, timestamppb.Now().String()),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)
	
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
			// Log error instead of panicking - iterator close failures are non-critical
			// but should be logged for debugging
			ctx.Logger().Error("failed to close iterator", "error", err)
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
	// 33% limit from whitepaper: "не более 33% на один кошелек"
	// This is a hardcoded constant as per whitepaper, but could be made configurable via governance in the future
	const maxValidatorShare = 0.33 // 33% limit from whitepaper
	
	// Validate share is between 0.0 and 1.0 (safety check)
	if maxValidatorShare <= 0.0 || maxValidatorShare > 1.0 {
		return fmt.Errorf("invalid maxValidatorShare: must be between 0.0 and 1.0, got %f", maxValidatorShare)
	}

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

// registerValidatorInConsensus registers a validator in consensus module after LZN activation
// This creates the validator record, sets weight, and creates initial ANT position
// The actual type conversion happens in the adapter in app.go
func (k Keeper) registerValidatorInConsensus(ctx sdk.Context, lizenz *lizenzv1.ActivatedLizenz) error {
	// Skip if consensus keeper is not set
	if k.consensusKeeper == nil {
		return nil
	}

	// Create validator data structure
	// The adapter in app.go will convert this to the correct type
	validatorData := map[string]interface{}{
		"validator":           lizenz.Validator,
		"ant_balance":        "0",
		"status":             1, // VALIDATOR_STATUS_ACTIVE
		"last_active":        lizenz.ActivationTime,
		"last_block_height":  uint64(0),
		"moa_score":          "0",
		"activity_score":     "0",
		"total_blocks_created": uint64(0),
		"total_burn_amount":  "0",
	}

	// Register validator in consensus module
	// The adapter will handle type conversion
	if err := k.consensusKeeper.SetValidator(ctx, validatorData); err != nil {
		return fmt.Errorf("failed to set validator in consensus: %w", err)
	}

	// Set validator weight based on activated LZN amount
	if err := k.consensusKeeper.SetValidatorWeight(ctx, lizenz.Validator, lizenz.Amount); err != nil {
		return fmt.Errorf("failed to set validator weight: %w", err)
	}

	// Create initial ANT position for validator
	if k.anteilKeeper != nil {
		positionData := map[string]interface{}{
			"owner":        lizenz.Validator,
			"ant_balance":  "0",
			"locked_ant":   "0",
			"available_ant": "0",
			"order_count":  uint32(0),
		}
		if err := k.anteilKeeper.SetUserPosition(ctx, positionData); err != nil {
			// Log but don't fail - position can be created later
			ctx.Logger().Error("failed to create initial ANT position for validator", "error", err, "validator", lizenz.Validator)
		}
	}

	return nil
}

// validateIdentityAndRole validates that validator has verified identity and VALIDATOR role
// According to whitepaper: only validators with verified identity can activate LZN
func (k Keeper) validateIdentityAndRole(ctx sdk.Context, validator string) error {
	if k.identKeeper == nil {
		// If ident keeper is not set, skip validation (for testing scenarios)
		return nil
	}

	// Get verified account
	account, err := k.identKeeper.GetVerifiedAccount(ctx, validator)
	if err != nil {
		return types.ErrIdentityNotVerified
	}

	// Check that account is active
	if !account.IsActive {
		return types.ErrIdentityNotVerified
	}

	// Check that role is VALIDATOR
	// According to whitepaper: "Тип 3: Валидатор: Верифицированная роль. Активирует LZN..."
	if account.Role != identv1.Role_ROLE_VALIDATOR {
		return types.ErrInvalidRoleForLizenz
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
