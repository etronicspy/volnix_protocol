package keeper

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// AnteilKeeperInterface defines the interface for interacting with anteil module
// This allows ident module to burn ANT when citizens are deactivated
type AnteilKeeperInterface interface {
	BurnAntFromUser(ctx sdk.Context, user string) error
	GetUserPosition(ctx sdk.Context, user string) (interface{}, error)
}

// AccountKeeperInterface allows ident to check if a blockchain account exists.
// Кошелёк считается зарегистрированным в блокчейне, если у него есть аккаунт (проведены транзакции).
type AccountKeeperInterface interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) interface{}
}

type (
	Keeper struct {
		cdc             codec.BinaryCodec
		storeKey        storetypes.StoreKey
		paramstore      paramtypes.Subspace
		anteilKeeper    AnteilKeeperInterface    // Optional: for burning ANT on supplier deactivation
		accountKeeper   AccountKeeperInterface   // Optional: to check blockchain account existence
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

// SetAnteilKeeper sets the anteil keeper interface for burning ANT on deactivation
func (k *Keeper) SetAnteilKeeper(anteilKeeper AnteilKeeperInterface) {
	k.anteilKeeper = anteilKeeper
}

// SetAccountKeeper sets the account keeper to check blockchain account existence
func (k *Keeper) SetAccountKeeper(accountKeeper AccountKeeperInterface) {
	k.accountKeeper = accountKeeper
}

// HasBlockchainAccount returns true if the address has a blockchain account (has conducted transactions).
// reqCtx must be the request context.Context (from msg handler) — auth keeper needs it to unwrap sdk.Context.
func (k Keeper) HasBlockchainAccount(reqCtx context.Context, address string) bool {
	if k.accountKeeper == nil {
		return false
	}
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return false
	}
	return k.accountKeeper.GetAccount(reqCtx, addr) != nil
}

// ReleaseIdentityHash releases the identity hash mapping for a deactivated account
// According to whitepaper: "ZKP-идентификатор освобождается для возможной повторной верификации"
func (k Keeper) ReleaseIdentityHash(ctx sdk.Context, address string) error {
	// Get account to retrieve identity hash
	account, err := k.GetVerifiedAccount(ctx, address)
	if err != nil {
		// If account doesn't exist, nothing to release
		ctx.Logger().Info("Account not found, nothing to release", "address", address)
		return nil
	}

	identityHash := account.IdentityHash
	if identityHash == "" {
		// No identity hash to release
		ctx.Logger().Info("No identity hash to release", "address", address)
		return nil
	}

	// Remove identity hash mapping from store
	store := ctx.KVStore(k.storeKey)
	identityHashKey := types.GetIdentityHashKey(identityHash)
	
	if store.Has(identityHashKey) {
		store.Delete(identityHashKey)
		ctx.Logger().Info("Identity hash released", "address", address, "identity_hash", identityHash)
		
		// Emit event for identity hash release
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"ident.identity_hash_released",
				sdk.NewAttribute("address", address),
				sdk.NewAttribute("identity_hash", identityHash),
				sdk.NewAttribute("reason", "deactivation"),
			),
		)
	} else {
		ctx.Logger().Info("Identity hash key not found in store", "address", address, "identity_hash", identityHash)
	}

	return nil
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

	identityHashKey := types.GetIdentityHashKey(account.IdentityHash)
	if store.Has(identityHashKey) {
		// Get the existing address that uses this identity hash
		existingAddress := store.Get(identityHashKey)
		return fmt.Errorf("%w: identity hash %s is already used by address %s", 
			types.ErrDuplicateIdentityHash, account.IdentityHash, string(existingAddress))
	}

	// OPTIMIZED: Check account limits (only if needed)
	// Skip limit check for roles that don't have limits or if limit is very high
	params := k.GetParams(ctx)
	if params.MaxActiveSuppliers < 10000 { // Only check if limit is reasonable
		if err := k.checkAccountLimits(ctx, account.Role, params); err != nil {
			return err
		}
	}

	// Store the account
	accountBz, err := k.cdc.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	store.Set(accountKey, accountBz)
	
	store.Set(identityHashKey, []byte(account.Address))
	
	return nil
}

// CreateAccountFromVerification creates a verified account from ZKP proof.
	// Used when a blockchain-registered wallet (has conducted transactions) upgrades to Supplier/Validator.
func (k Keeper) CreateAccountFromVerification(ctx sdk.Context, address, zkpProof, verificationProvider string, desiredRole identv1.Role) error {
	if err := k.ValidateVerificationRequest(ctx, address, zkpProof, verificationProvider, desiredRole); err != nil {
		return err
	}
	identityHash := fmt.Sprintf("hash-%s", zkpProof[:min(16, len(zkpProof))])
	if err := k.CheckDuplicateIdentityHash(ctx, identityHash, address); err != nil {
		return err
	}
	account := types.NewVerifiedAccount(address, desiredRole, identityHash, ctx.BlockTime())
	return k.SetVerifiedAccount(ctx, account)
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
	// Activity is updated in AnteHandler when user signs a tx (per whitepaper)
	return nil
}

// checkAccountActivity checks account activity and updates roles if needed
// checkAccountActivity implements §5.3 MOA — Minimum Obligation of Activity
// Supplier MOA (T_g): must place at least one sell order within the window.
// Validator MOA (T_v): must submit at least one per-height burn within the window.
// Sanction for Supplier: burn ANT → demote to Guest.
// Sanction for Validator: request LZN deactivation → remove from ValidatorSet.
func (k Keeper) checkAccountActivity(ctx sdk.Context) error {
	allAccounts, err := k.GetAllVerifiedAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get verified accounts: %w", err)
	}

	params := k.GetParams(ctx)
	currentTime := ctx.BlockTime()

	for _, account := range allAccounts {
		if !account.IsActive {
			continue
		}

		var moaWindow time.Duration
		var burnReason string
		var roleChangeReason string

		switch account.GetRole() {
		case identv1.Role_ROLE_SUPPLIER:
			moaWindow = params.MoaSupplierWindow
			burnReason = types.BurnReasonMOASupplier
			roleChangeReason = types.RoleChangeReasonMOASupplier
		case identv1.Role_ROLE_VALIDATOR:
			moaWindow = params.MoaValidatorWindow
			burnReason = types.BurnReasonMOAValidator
			roleChangeReason = types.RoleChangeReasonMOAValidator
		default:
			continue
		}

		if moaWindow == 0 {
			continue
		}

		// §5.3: check last qualifying MOA event, falling back to last_active
		lastEvent := account.GetLastActive().AsTime()
		if account.LastMoaEvent != nil {
			moaTime := account.LastMoaEvent.AsTime()
			if moaTime.After(lastEvent) {
				lastEvent = moaTime
			}
		}

		if currentTime.Sub(lastEvent) <= moaWindow {
			continue
		}

		// MOA violation detected
		oldRole := account.Role

		if account.Role == identv1.Role_ROLE_SUPPLIER {
			// §5.3(1): burn supplier ANT, demote to Guest
			if k.anteilKeeper != nil {
				_ = k.anteilKeeper.BurnAntFromUser(ctx, account.Address)
			}

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeAntBurned,
					sdk.NewAttribute(types.AttributeKeyAccount, account.Address),
					sdk.NewAttribute(types.AttributeKeyBurnReason, burnReason),
					sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
				),
			)
		}

		// Release identity hash
		if err := k.ReleaseIdentityHash(ctx, account.Address); err != nil {
			ctx.Logger().Error("Failed to release identity hash", "address", account.Address, "error", err)
		}

		// Demote to Guest
		account.Role = identv1.Role_ROLE_GUEST
		account.IsActive = false

		if err := k.UpdateVerifiedAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to update account after MOA sanction: %w", err)
		}

		// Emit role change event §6.3
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeRoleChanged,
				sdk.NewAttribute(types.AttributeKeyAccount, account.Address),
				sdk.NewAttribute(types.AttributeKeyOldRole, oldRole.String()),
				sdk.NewAttribute(types.AttributeKeyNewRole, identv1.Role_ROLE_GUEST.String()),
				sdk.NewAttribute(types.AttributeKeyRoleChangeReason, roleChangeReason),
				sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
			),
		)

		_ = burnReason // used above

		ctx.Logger().Info("MOA violation: account sanctioned",
			"address", account.Address,
			"old_role", oldRole.String(),
			"reason", roleChangeReason,
		)
	}

	return nil
}

// processRoleMigrations processes pending role migrations (e.g. created but not yet executed).
// Migrations created via MsgMigrateRole are executed immediately; this handles any deferred flow.
func (k Keeper) processRoleMigrations(ctx sdk.Context) error {
	migrations, err := k.GetAllRoleMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get role migrations: %w", err)
	}
	for _, m := range migrations {
		if m.IsCompleted {
			continue
		}
		if err := k.ExecuteRoleMigration(ctx, m.FromAddress, m.ToAddress); err != nil {
			ctx.Logger().Debug("role migration skipped",
				"from", m.FromAddress, "to", m.ToAddress, "error", err)
			continue
		}
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("ident.role_migrated",
				sdk.NewAttribute("from_address", m.FromAddress),
				sdk.NewAttribute("to_address", m.ToAddress)),
		)
	}
	return nil
}

// UpdateActivityForSigners updates LastActive for verified accounts that signed the tx.
// Per whitepaper: activity = signed transaction. Call from AnteHandler on successful tx.
func (k Keeper) UpdateActivityForSigners(ctx sdk.Context, signers []sdk.AccAddress) error {
	if len(signers) == 0 {
		return nil
	}
	currentTime := ctx.BlockTime()
	seen := make(map[string]bool)
	for _, signer := range signers {
		addr := signer.String()
		if seen[addr] {
			continue
		}
		seen[addr] = true
		account, err := k.GetVerifiedAccount(ctx, addr)
		if err != nil {
			continue // Not a verified account, skip
		}
		account.LastActive = &timestamppb.Timestamp{Seconds: currentTime.Unix()}
		if err := k.UpdateVerifiedAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to update account activity for %s: %w", addr, err)
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
			// Log error instead of panicking - iterator close failures are non-critical
			// but should be logged for debugging
			ctx.Logger().Error("failed to close iterator", "error", err)
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

// CheckDuplicateIdentityHash checks if an identity hash is already used by another address.
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
	case identv1.Role_ROLE_SUPPLIER:
		accounts, err := k.GetVerifiedAccountsByRole(ctx, identv1.Role_ROLE_SUPPLIER)
		if err != nil {
			return err
		}
		currentCount = uint64(len(accounts))
		maxCount = params.MaxActiveSuppliers

	case identv1.Role_ROLE_VALIDATOR:
		accounts, err := k.GetVerifiedAccountsByRole(ctx, identv1.Role_ROLE_VALIDATOR)
		if err != nil {
			return err
		}
		currentCount = uint64(len(accounts))
		maxCount = params.MaxActiveSuppliers

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

	types.UpdateAccountActivity(account, ctx.BlockTime())
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
	types.ChangeAccountRole(account, newRole, ctx.BlockTime())
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

	// SECURITY: Prevent downgrade from VALIDATOR to GUEST (suspicious)
	// This could indicate account compromise
	if oldRole == identv1.Role_ROLE_VALIDATOR && newRole == identv1.Role_ROLE_GUEST {
		return fmt.Errorf("direct downgrade from validator to guest is not allowed for security reasons")
	}
	
	// SECURITY: Prevent downgrade from SUPPLIER to GUEST without proper procedure
	// This should go through deactivation process
	if oldRole == identv1.Role_ROLE_SUPPLIER && newRole == identv1.Role_ROLE_GUEST {
		return fmt.Errorf("direct downgrade from supplier to guest is not allowed, use deactivation process")
	}

	return nil
}

// ValidateRoleChangeProof validates ZKP proof for role change
// This prevents unauthorized role escalation attacks
func (k Keeper) ValidateRoleChangeProof(ctx sdk.Context, address string, identityHash string, zkpProof string, newRole identv1.Role) error {
	// Basic validation
	if zkpProof == "" {
		return fmt.Errorf("ZKP proof cannot be empty")
	}
	
	if identityHash == "" {
		return fmt.Errorf("identity hash cannot be empty")
	}
	
	// Verify proof has correct format
	if len(zkpProof) < 16 {
		return fmt.Errorf("ZKP proof is too short")
	}

	providerID := ""
	account, err := k.GetVerifiedAccount(ctx, address)
	if err == nil && account != nil {
		providerID = account.VerificationProvider
	}

	// Structured proofs include public_inputs that bind the request to
	// concrete identity/address/target-role and are checked here.
	if parsed, structured, err := parseZKPProofString(zkpProof); err == nil && structured {
		if !strings.Contains(parsed.PublicInputs, address) {
			return fmt.Errorf("proof public inputs do not include address")
		}
		if !strings.Contains(parsed.PublicInputs, identityHash) {
			return fmt.Errorf("proof public inputs do not include identity hash")
		}
		if !strings.Contains(parsed.PublicInputs, newRole.String()) {
			return fmt.Errorf("proof public inputs do not include target role")
		}
	}

	// Replay/integrity verification (also validates structured proof hash).
	if err := k.VerifyZKProofIntegrity(ctx, zkpProof, providerID, address); err != nil {
		return fmt.Errorf("role change proof integrity failed: %w", err)
	}

	ctx.Logger().Info("Role change ZKP proof validated",
		"address", address, 
		"identity_hash", identityHash, 
		"new_role", newRole.String(),
		"provider", providerID)

	return nil
}

// ValidateRoleChoice validates that the role choice during verification is valid
// According to whitepaper, user must choose between ROLE_SUPPLIER or ROLE_VALIDATOR
func (k Keeper) ValidateRoleChoice(ctx sdk.Context, address string, desiredRole identv1.Role) error {
	_, err := k.GetVerifiedAccount(ctx, address)
	if err == nil {
		return types.ErrAlreadyVerified
	}
	if !errors.Is(err, types.ErrAccountNotFound) {
		return fmt.Errorf("failed to check existing account: %w", err)
	}

	// Validate that role is either SUPPLIER or VALIDATOR
	if desiredRole != identv1.Role_ROLE_SUPPLIER && desiredRole != identv1.Role_ROLE_VALIDATOR {
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

	// Remove old identity hash index to allow migration to use the same hash
	store := ctx.KVStore(k.storeKey)
	if sourceAccount.IdentityHash == migration.MigrationHash {
		// If using the same identity hash, remove the old mapping
		oldIdentityHashKey := types.GetIdentityHashKey(sourceAccount.IdentityHash)
		store.Delete(oldIdentityHashKey)
	}

	// Create target account with same role
	targetAccount := &identv1.VerifiedAccount{
		Address:              toAddress,
		Role:                 sourceAccount.Role,
		VerificationDate:     timestamppb.New(ctx.BlockTime()),
		LastActive:           timestamppb.New(ctx.BlockTime()),
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
	migration.MigrationDate = timestamppb.New(ctx.BlockTime())
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
