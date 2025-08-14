package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
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

// GetParams returns the current parameters for the ident module
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var params types.Params
	k.paramstore.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the parameters for the ident module
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// SetVerifiedAccount stores a verified account in the store
func (k Keeper) SetVerifiedAccount(ctx sdk.Context, account *identv1.VerifiedAccount) error {
	if err := types.ValidateAccount(account); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	accountKey := types.GetVerifiedAccountKey(account.Address)

	// Check if account already exists
	if store.Has(accountKey) {
		return types.ErrAccountAlreadyExists
	}

	// Check account limits
	params := k.GetParams(ctx)
	if err := k.checkAccountLimits(ctx, account.Role, params); err != nil {
		return err
	}

	// Store the account
	accountBz, err := k.cdc.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	store.Set(accountKey, accountBz)
	return nil
}

// GetVerifiedAccount retrieves a verified account by address
func (k Keeper) GetVerifiedAccount(ctx sdk.Context, address string) (*identv1.VerifiedAccount, error) {
	store := ctx.KVStore(k.storeKey)
	accountKey := types.GetVerifiedAccountKey(address)

	if !store.Has(accountKey) {
		return nil, types.ErrAccountNotFound
	}

	accountBz := store.Get(accountKey)
	var account identv1.VerifiedAccount
	if err := k.cdc.Unmarshal(accountBz, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	return &account, nil
}

// UpdateVerifiedAccount updates an existing verified account
func (k Keeper) UpdateVerifiedAccount(ctx sdk.Context, account *identv1.VerifiedAccount) error {
	if err := types.ValidateAccount(account); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	accountKey := types.GetVerifiedAccountKey(account.Address)

	// Check if account exists
	if !store.Has(accountKey) {
		return types.ErrAccountNotFound
	}

	// Store the updated account
	accountBz, err := k.cdc.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	store.Set(accountKey, accountBz)
	return nil
}

// DeleteVerifiedAccount removes a verified account from the store
func (k Keeper) DeleteVerifiedAccount(ctx sdk.Context, address string) error {
	store := ctx.KVStore(k.storeKey)
	accountKey := types.GetVerifiedAccountKey(address)

	if !store.Has(accountKey) {
		return types.ErrAccountNotFound
	}

	store.Delete(accountKey)
	return nil
}

// GetAllVerifiedAccounts retrieves all verified accounts
func (k Keeper) GetAllVerifiedAccounts(ctx sdk.Context) ([]*identv1.VerifiedAccount, error) {
	store := ctx.KVStore(k.storeKey)
	accountStore := prefix.NewStore(store, types.VerifiedAccountKeyPrefix)

	var accounts []*identv1.VerifiedAccount
	iterator := accountStore.Iterator(nil, nil)
	defer func() {
		if err := iterator.Close(); err != nil {
			panic(fmt.Sprintf("failed to close iterator: %v", err))
		}
	}()

	for ; iterator.Valid(); iterator.Next() {
		var account identv1.VerifiedAccount
		if err := k.cdc.Unmarshal(iterator.Value(), &account); err != nil {
			return nil, fmt.Errorf("failed to unmarshal account: %w", err)
		}
		accounts = append(accounts, &account)
	}

	return accounts, nil
}

// GetVerifiedAccountsByRole retrieves all verified accounts with a specific role
func (k Keeper) GetVerifiedAccountsByRole(ctx sdk.Context, role identv1.Role) ([]*identv1.VerifiedAccount, error) {
	allAccounts, err := k.GetAllVerifiedAccounts(ctx)
	if err != nil {
		return nil, err
	}

	var filteredAccounts []*identv1.VerifiedAccount
	for _, account := range allAccounts {
		if account.Role == role {
			filteredAccounts = append(filteredAccounts, account)
		}
	}

	return filteredAccounts, nil
}

// checkAccountLimits verifies that account creation doesn't exceed limits
func (k Keeper) checkAccountLimits(ctx sdk.Context, role identv1.Role, params types.Params) error {
	var currentCount uint64
	var maxCount uint64

	switch role {
	case identv1.Role_ROLE_CITIZEN:
		accounts, err := k.GetVerifiedAccountsByRole(ctx, identv1.Role_ROLE_CITIZEN)
		if err != nil {
			return err
		}
		currentCount = uint64(len(accounts))
		maxCount = params.MaxCitizenAccounts

	case identv1.Role_ROLE_VALIDATOR:
		accounts, err := k.GetVerifiedAccountsByRole(ctx, identv1.Role_ROLE_VALIDATOR)
		if err != nil {
			return err
		}
		currentCount = uint64(len(accounts))
		maxCount = params.MaxValidatorAccounts

	default:
		return types.ErrInvalidRole
	}

	if currentCount >= maxCount {
		return fmt.Errorf("account limit exceeded for role %s: current %d, max %d", role, currentCount, maxCount)
	}

	return nil
}

// UpdateAccountActivity updates the last active timestamp for an account
func (k Keeper) UpdateAccountActivity(ctx sdk.Context, address string) error {
	account, err := k.GetVerifiedAccount(ctx, address)
	if err != nil {
		return err
	}

	types.UpdateAccountActivity(account)
	return k.UpdateVerifiedAccount(ctx, account)
}

// ChangeAccountRole changes the role of an existing account
func (k Keeper) ChangeAccountRole(ctx sdk.Context, address string, newRole identv1.Role) error {
	account, err := k.GetVerifiedAccount(ctx, address)
	if err != nil {
		return err
	}

	// Check if role change is allowed
	if err := k.validateRoleChange(ctx, account.Role, newRole); err != nil {
		return err
	}

	// Check account limits for new role
	params := k.GetParams(ctx)
	if err := k.checkAccountLimits(ctx, newRole, params); err != nil {
		return err
	}

	// Change role and update activity
	types.ChangeAccountRole(account, newRole)
	return k.UpdateVerifiedAccount(ctx, account)
}

// validateRoleChange checks if the role change is valid
func (k Keeper) validateRoleChange(ctx sdk.Context, oldRole, newRole identv1.Role) error {
	// Basic validation
	if newRole == identv1.Role_ROLE_UNSPECIFIED {
		return types.ErrInvalidRole
	}

	// Allow same role (no change)
	if oldRole == newRole {
		return nil
	}

	// Add specific business rules for role changes here
	// For example, only allow certain role transitions

	return nil
}
