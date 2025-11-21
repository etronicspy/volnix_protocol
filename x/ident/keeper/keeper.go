package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"google.golang.org/protobuf/types/known/timestamppb"

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

	// IMPROVED: Check for duplicate identity hash
	// This prevents the same identity from being used by multiple addresses
	identityHashKey := types.GetIdentityHashKey(account.IdentityHash)
	if store.Has(identityHashKey) {
		// Get the existing address that uses this identity hash
		existingAddress := store.Get(identityHashKey)
		return fmt.Errorf("%w: identity hash %s is already used by address %s", 
			types.ErrDuplicateIdentityHash, account.IdentityHash, string(existingAddress))
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
	
	// IMPROVED: Store identity hash mapping to prevent duplicates
	store.Set(identityHashKey, []byte(account.Address))
	
	return nil
}

// ========================================
// BLOCK PROCESSORS - BeginBlocker/EndBlocker Logic
// ========================================

// BeginBlocker processes events at the beginning of each block
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	// Check account activity and update roles if needed
	if err := k.checkAccountActivity(ctx); err != nil {
		return fmt.Errorf("failed to check account activity: %w", err)
	}

	// Process role migrations
	if err := k.processRoleMigrations(ctx); err != nil {
		return fmt.Errorf("failed to process role migrations: %w", err)
	}

	return nil
}

// EndBlocker processes events at the end of each block
func (k Keeper) EndBlocker(ctx sdk.Context) error {
	// Update account activity timestamps
	if err := k.updateAccountActivity(ctx); err != nil {
		return fmt.Errorf("failed to update account activity: %w", err)
	}

	return nil
}

// checkAccountActivity checks account activity and updates roles if needed
func (k Keeper) checkAccountActivity(ctx sdk.Context) error {
	allAccounts, err := k.GetAllVerifiedAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get verified accounts: %w", err)
	}

	// Get params for activity periods
	params := k.GetParams(ctx)
	currentTime := ctx.BlockTime()

	for _, account := range allAccounts {
		// Check if account has been inactive for too long
		lastActivity := account.GetLastActive().AsTime()

		var activityPeriod time.Duration
		switch account.GetRole() {
		case identv1.Role_ROLE_CITIZEN:
			activityPeriod = params.CitizenActivityPeriod
		case identv1.Role_ROLE_VALIDATOR:
			activityPeriod = params.ValidatorActivityPeriod
		default:
			continue // Skip guests
		}

		if currentTime.Sub(lastActivity) > activityPeriod {
			// Downgrade role to guest
			account.Role = identv1.Role_ROLE_GUEST

			// Update account in store
			if err := k.UpdateVerifiedAccount(ctx, account); err != nil {
				return fmt.Errorf("failed to update inactive account: %w", err)
			}
		}
	}

	return nil
}

// processRoleMigrations processes pending role migrations
func (k Keeper) processRoleMigrations(ctx sdk.Context) error {
	// This would process any pending role migrations
	// For now, it's a placeholder for future implementation
	return nil
}

// updateAccountActivity updates activity timestamps for all accounts
func (k Keeper) updateAccountActivity(ctx sdk.Context) error {
	allAccounts, err := k.GetAllVerifiedAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get verified accounts: %w", err)
	}

	currentTime := ctx.BlockTime()

	for _, account := range allAccounts {
		// Update last activity timestamp
		account.LastActive = &timestamppb.Timestamp{Seconds: currentTime.Unix()}

		// Update in store
		if err := k.UpdateVerifiedAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to update account activity: %w", err)
		}
	}

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

	// IMPROVED: Check for duplicate identity hash if identity hash changed
	// Get existing account to compare identity hash
	var existingAccount identv1.VerifiedAccount
	existingAccountBz := store.Get(accountKey)
	if err := k.cdc.Unmarshal(existingAccountBz, &existingAccount); err == nil {
		// If identity hash changed, check for duplicates
		if existingAccount.IdentityHash != account.IdentityHash {
			identityHashKey := types.GetIdentityHashKey(account.IdentityHash)
			if store.Has(identityHashKey) {
				existingAddress := string(store.Get(identityHashKey))
				if existingAddress != account.Address {
					return fmt.Errorf("%w: identity hash %s is already used by address %s", 
						types.ErrDuplicateIdentityHash, account.IdentityHash, existingAddress)
				}
			}
			// Remove old identity hash mapping
			oldIdentityHashKey := types.GetIdentityHashKey(existingAccount.IdentityHash)
			store.Delete(oldIdentityHashKey)
			// Set new identity hash mapping
			store.Set(identityHashKey, []byte(account.Address))
		}
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

// IMPROVED: CheckDuplicateIdentityHash checks if an identity hash is already used by another address
func (k Keeper) CheckDuplicateIdentityHash(ctx sdk.Context, identityHash string, currentAddress string) error {
	store := ctx.KVStore(k.storeKey)
	identityHashKey := types.GetIdentityHashKey(identityHash)
	
	if store.Has(identityHashKey) {
		existingAddress := string(store.Get(identityHashKey))
		// Allow if it's the same address (for updates)
		if existingAddress != currentAddress {
			return fmt.Errorf("%w: identity hash %s is already used by address %s", 
				types.ErrDuplicateIdentityHash, identityHash, existingAddress)
		}
	}
	
	return nil
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
		maxCount = params.MaxIdentitiesPerAddress

	case identv1.Role_ROLE_VALIDATOR:
		accounts, err := k.GetVerifiedAccountsByRole(ctx, identv1.Role_ROLE_VALIDATOR)
		if err != nil {
			return err
		}
		currentCount = uint64(len(accounts))
		maxCount = params.MaxIdentitiesPerAddress

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

// ValidateRoleChoice validates that the role choice during verification is valid
// According to whitepaper, user must choose between ROLE_CITIZEN or ROLE_VALIDATOR
func (k Keeper) ValidateRoleChoice(ctx sdk.Context, address string, desiredRole identv1.Role) error {
	// Check if address already has a verified account
	if _, err := k.GetVerifiedAccount(ctx, address); err == nil {
		return types.ErrAlreadyVerified
	}

	// Validate that role is either CITIZEN or VALIDATOR
	if desiredRole != identv1.Role_ROLE_CITIZEN && desiredRole != identv1.Role_ROLE_VALIDATOR {
		return types.ErrInvalidRoleChoice
	}

	// Role cannot be GUEST or UNSPECIFIED
	if desiredRole == identv1.Role_ROLE_GUEST || desiredRole == identv1.Role_ROLE_UNSPECIFIED {
		return types.ErrInvalidRoleChoice
	}

	return nil
}

// SetRoleMigration sets a role migration request
func (k Keeper) SetRoleMigration(ctx sdk.Context, migration *identv1.RoleMigration) error {
	store := ctx.KVStore(k.storeKey)
	migrationKey := types.GetRoleMigrationKey(migration.FromAddress, migration.ToAddress)

	migrationBz, err := k.cdc.Marshal(migration)
	if err != nil {
		return err
	}
	store.Set(migrationKey, migrationBz)

	return nil
}

// GetRoleMigration retrieves a role migration by addresses
func (k Keeper) GetRoleMigration(ctx sdk.Context, fromAddress, toAddress string) (*identv1.RoleMigration, error) {
	store := ctx.KVStore(k.storeKey)
	migrationKey := types.GetRoleMigrationKey(fromAddress, toAddress)

	if !store.Has(migrationKey) {
		return nil, types.ErrRoleMigrationNotFound
	}

	migrationBz := store.Get(migrationKey)
	var migration identv1.RoleMigration
	if err := k.cdc.Unmarshal(migrationBz, &migration); err != nil {
		return nil, err
	}

	return &migration, nil
}

// ExecuteRoleMigration executes a role migration
func (k Keeper) ExecuteRoleMigration(ctx sdk.Context, fromAddress, toAddress string) error {
	migration, err := k.GetRoleMigration(ctx, fromAddress, toAddress)
	if err != nil {
		return err
	}

	// Check if migration is valid
	if migration.IsCompleted {
		return types.ErrInvalidMigrationStatus
	}

	// Get source account
	sourceAccount, err := k.GetVerifiedAccount(ctx, fromAddress)
	if err != nil {
		return err
	}

	// Create target account with same role
	targetAccount := &identv1.VerifiedAccount{
		Address:              toAddress,
		Role:                 sourceAccount.Role,
		VerificationDate:     timestamppb.Now(),
		LastActive:           timestamppb.Now(),
		IsActive:             true,
		IdentityHash:         migration.MigrationHash,
		VerificationProvider: sourceAccount.VerificationProvider,
	}

	// Set target account
	if err := k.SetVerifiedAccount(ctx, targetAccount); err != nil {
		return err
	}

	// Deactivate source account
	sourceAccount.IsActive = false
	if err := k.UpdateVerifiedAccount(ctx, sourceAccount); err != nil {
		return err
	}

	// Update migration status
	migration.IsCompleted = true
	migration.MigrationDate = timestamppb.Now()
	return k.SetRoleMigration(ctx, migration)
}

// GetAllRoleMigrations returns all role migrations
func (k Keeper) GetAllRoleMigrations(ctx sdk.Context) ([]*identv1.RoleMigration, error) {
	store := ctx.KVStore(k.storeKey)
	prefix := types.RoleMigrationKeyPrefix

	var migrations []*identv1.RoleMigration
	iterator := store.Iterator(prefix, append(prefix, 0xFF))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var migration identv1.RoleMigration
		if err := k.cdc.Unmarshal(iterator.Value(), &migration); err != nil {
			continue
		}
		migrations = append(migrations, &migration)
	}

	return migrations, nil
}
