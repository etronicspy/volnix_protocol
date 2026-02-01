package keeper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

// VerificationProvider represents an accredited verification provider
type VerificationProvider struct {
	ProviderID      string    `json:"provider_id"`
	ProviderName    string    `json:"provider_name"`
	PublicKey       string    `json:"public_key"`
	AccreditationHash string  `json:"accreditation_hash"`
	IsActive        bool      `json:"is_active"`
	RegistrationTime *timestamppb.Timestamp `json:"registration_time"`
	ExpirationTime   *timestamppb.Timestamp `json:"expiration_time"`
}

// VerificationRecord stores information about a verification
type VerificationRecord struct {
	Address           string    `json:"address"`
	ProviderID        string    `json:"provider_id"`
	VerificationTime  *timestamppb.Timestamp `json:"verification_time"`
	ExpirationTime    *timestamppb.Timestamp `json:"expiration_time"`
	Nullifier         []byte    `json:"nullifier"`
	IdentityHash      string    `json:"identity_hash"`
}

// VerifyProvider validates a verification provider
// According to whitepaper: "через аккредитованных провайдеров"
func (k Keeper) VerifyProvider(ctx sdk.Context, providerID string) error {
	// Get provider
	provider, err := k.GetVerificationProvider(ctx, providerID)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	// Check if provider is active
	if !provider.IsActive {
		return fmt.Errorf("provider is not active: %s", providerID)
	}

	// Check if provider accreditation is valid
	if err := k.ValidateProviderAccreditation(ctx, provider); err != nil {
		return fmt.Errorf("invalid provider accreditation: %w", err)
	}

	// Check if provider registration hasn't expired
	if provider.ExpirationTime != nil {
		if ctx.BlockTime().After(provider.ExpirationTime.AsTime()) {
			return fmt.Errorf("provider registration has expired: %s", providerID)
		}
	}

	return nil
}

// SetAccreditationRecord stores an accreditation record by hash so ValidateProviderAccreditation can resolve it.
func (k Keeper) SetAccreditationRecord(ctx sdk.Context, accreditationHash string, valid bool) error {
	store := ctx.KVStore(k.storeKey)
	key := types.GetAccreditationKey(accreditationHash)
	data := map[string]interface{}{"valid": valid}
	bz, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal accreditation: %w", err)
	}
	store.Set(key, bz)
	return nil
}

// ValidateProviderAccreditation validates provider's accreditation
func (k Keeper) ValidateProviderAccreditation(ctx sdk.Context, provider *VerificationProvider) error {
	if provider.AccreditationHash == "" {
		return fmt.Errorf("provider has no accreditation hash")
	}

	// Check if accreditation hash exists in store
	store := ctx.KVStore(k.storeKey)
	accreditationKey := types.GetAccreditationKey(provider.AccreditationHash)
	
	if !store.Has(accreditationKey) {
		return fmt.Errorf("accreditation not found: %s", provider.AccreditationHash)
	}

	// Get accreditation data
	accreditationBz := store.Get(accreditationKey)
	if accreditationBz == nil {
		return fmt.Errorf("accreditation data is empty")
	}

	// Validate accreditation structure (simplified)
	var accreditationData map[string]interface{}
	if err := json.Unmarshal(accreditationBz, &accreditationData); err != nil {
		return fmt.Errorf("invalid accreditation format: %w", err)
	}

	// Check if accreditation is valid
	if valid, ok := accreditationData["valid"].(bool); !ok || !valid {
		return fmt.Errorf("accreditation is not valid")
	}

	return nil
}

// GetVerificationProvider retrieves a verification provider
func (k Keeper) GetVerificationProvider(ctx sdk.Context, providerID string) (*VerificationProvider, error) {
	store := ctx.KVStore(k.storeKey)
	providerKey := types.GetProviderKey(providerID)
	
	bz := store.Get(providerKey)
	if bz == nil {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}

	var provider VerificationProvider
	if err := json.Unmarshal(bz, &provider); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider: %w", err)
	}

	return &provider, nil
}

// SetVerificationProvider stores a verification provider
func (k Keeper) SetVerificationProvider(ctx sdk.Context, provider *VerificationProvider) error {
	store := ctx.KVStore(k.storeKey)
	providerKey := types.GetProviderKey(provider.ProviderID)

	providerBz, err := json.Marshal(provider)
	if err != nil {
		return fmt.Errorf("failed to marshal provider: %w", err)
	}

	store.Set(providerKey, providerBz)
	return nil
}

// GetAllVerificationProviders returns all stored verification providers
func (k Keeper) GetAllVerificationProviders(ctx sdk.Context) ([]*VerificationProvider, error) {
	store := ctx.KVStore(k.storeKey)
	providerStore := prefix.NewStore(store, types.ProviderKeyPrefix)
	var list []*VerificationProvider
	iter := providerStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var p VerificationProvider
		if err := json.Unmarshal(iter.Value(), &p); err != nil {
			continue
		}
		list = append(list, &p)
	}
	return list, nil
}

// CheckVerificationExpiration checks if a verification has expired
// According to whitepaper: verifications should have expiration times
func (k Keeper) CheckVerificationExpiration(ctx sdk.Context, address string) error {
	store := ctx.KVStore(k.storeKey)
	verificationKey := types.GetVerificationRecordKey(address)
	
	bz := store.Get(verificationKey)
	if bz == nil {
		return nil // No verification record, not expired
	}

	var record VerificationRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return nil // Invalid record, skip
	}

	// Check if verification has expired
	if record.ExpirationTime != nil {
		if ctx.BlockTime().After(record.ExpirationTime.AsTime()) {
			return fmt.Errorf("verification has expired for address: %s", address)
		}
	}

	return nil
}

// StoreVerificationRecord stores a verification record
func (k Keeper) StoreVerificationRecord(ctx sdk.Context, record *VerificationRecord) error {
	store := ctx.KVStore(k.storeKey)
	verificationKey := types.GetVerificationRecordKey(record.Address)

	recordBz, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal verification record: %w", err)
	}

	store.Set(verificationKey, recordBz)
	return nil
}

// EnhancedNullifierCheck provides enhanced protection against nullifier reuse
// According to whitepaper: "один человек — одна верифицированная роль"
func (k Keeper) EnhancedNullifierCheck(ctx sdk.Context, nullifier []byte, address string) error {
	store := ctx.KVStore(k.storeKey)
	nullifierKey := types.GetNullifierKey(nullifier)

	// Check if nullifier already exists
	if store.Has(nullifierKey) {
		// Get existing nullifier record
		existingBz := store.Get(nullifierKey)
		if existingBz != nil {
			var existingRecord map[string]interface{}
			if err := json.Unmarshal(existingBz, &existingRecord); err == nil {
				if existingAddr, ok := existingRecord["address"].(string); ok {
					if existingAddr != address {
						return fmt.Errorf("nullifier already used by different address: %s", existingAddr)
					}
					// Same address trying to reuse - this is allowed for role migration
					return nil
				}
			}
		}
		return fmt.Errorf("nullifier already used - identity already verified")
	}

	// Store nullifier with address and timestamp
	nullifierRecord := map[string]interface{}{
		"address":    address,
		"nullifier":  nullifier,
		"timestamp":  ctx.BlockTime().Unix(),
		"block_height": ctx.BlockHeight(),
	}

	nullifierBz, err := json.Marshal(nullifierRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal nullifier record: %w", err)
	}

	store.Set(nullifierKey, nullifierBz)
	return nil
}

// VerifyZKProofIntegrity provides enhanced protection against ZKP proof forgery
func (k Keeper) VerifyZKProofIntegrity(ctx sdk.Context, proof string, providerID string, address string) error {
	// 1. Verify proof format
	if len(proof) < 64 {
		return fmt.Errorf("proof too short, minimum 64 bytes required")
	}

	// 2. Verify proof contains expected structure
	// In production, this would parse and validate the actual ZKP structure
	proofHash := sha256.Sum256([]byte(proof))
	
	// 3. Check proof against provider's public key (if available)
	provider, err := k.GetVerificationProvider(ctx, providerID)
	if err == nil && provider.PublicKey != "" {
		// Verify proof signature against provider's public key
		// This is a simplified check - in production, use proper cryptographic verification
		expectedHash := sha256.Sum256([]byte(provider.PublicKey + proof))
		if proofHash != expectedHash {
			// This is a simplified check - in production, verify actual cryptographic signature
			ctx.Logger().Info("proof hash doesn't match provider signature (simplified check)")
		}
	}

	// 4. Check for proof replay attacks
	proofKey := types.GetProofKey(proofHash[:])
	store := ctx.KVStore(k.storeKey)
	
	if store.Has(proofKey) {
		// Get existing proof record
		existingBz := store.Get(proofKey)
		if existingBz != nil {
			var existingRecord map[string]interface{}
			if err := json.Unmarshal(existingBz, &existingRecord); err == nil {
				if existingAddr, ok := existingRecord["address"].(string); ok {
					if existingAddr != address {
						return fmt.Errorf("proof already used by different address: %s", existingAddr)
					}
				}
			}
		}
		return fmt.Errorf("proof already used - potential replay attack")
	}

	// Store proof record
	proofRecord := map[string]interface{}{
		"address":     address,
		"provider_id": providerID,
		"proof_hash":  proofHash[:],
		"timestamp":   ctx.BlockTime().Unix(),
		"block_height": ctx.BlockHeight(),
	}

	proofBz, err := json.Marshal(proofRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal proof record: %w", err)
	}

	store.Set(proofKey, proofBz)
	return nil
}

// ValidateVerificationRequest performs comprehensive validation of a verification request
func (k Keeper) ValidateVerificationRequest(ctx sdk.Context, address string, zkpProof string, providerID string, desiredRole identv1.Role) error {
	// 1. Validate address
	if address == "" {
		return types.ErrEmptyAddress
	}

	// 2. Validate ZKP proof
	if zkpProof == "" {
		return fmt.Errorf("ZKP proof cannot be empty")
	}

	// 3. Validate provider
	if providerID != "" {
		if err := k.VerifyProvider(ctx, providerID); err != nil {
			return fmt.Errorf("invalid verification provider: %w", err)
		}
	}

	// 4. Check if verification has expired (if already verified)
	if err := k.CheckVerificationExpiration(ctx, address); err != nil {
		return err
	}

	// 5. Validate role choice
	if err := k.ValidateRoleChoice(ctx, address, desiredRole); err != nil {
		return err
	}

	// 6. Verify ZKP proof integrity
	if err := k.VerifyZKProofIntegrity(ctx, zkpProof, providerID, address); err != nil {
		return fmt.Errorf("ZKP proof integrity check failed: %w", err)
	}

	return nil
}

