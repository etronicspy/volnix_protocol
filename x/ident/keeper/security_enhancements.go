package keeper

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

func computeStructuredProofHash(proofData, publicInputs, verificationKey string) string {
	sum := sha256.Sum256([]byte(proofData + "|" + publicInputs + "|" + verificationKey))
	return hex.EncodeToString(sum[:])
}

func decodeProviderPublicKey(pub string) ([]byte, error) {
	pub = strings.TrimSpace(pub)
	if pub == "" {
		return nil, fmt.Errorf("provider public key is empty")
	}

	// Try base64 first (recommended wire format)
	if bz, err := base64.StdEncoding.DecodeString(pub); err == nil && len(bz) == ed25519.PublicKeySize {
		return bz, nil
	}
	if bz, err := base64.RawStdEncoding.DecodeString(pub); err == nil && len(bz) == ed25519.PublicKeySize {
		return bz, nil
	}

	// Fallback: hex-encoded key
	if bz, err := hex.DecodeString(pub); err == nil && len(bz) == ed25519.PublicKeySize {
		return bz, nil
	}

	return nil, fmt.Errorf("unsupported provider public key format")
}

func parseZKPProofString(proof string) (*identv1.ZKPProof, bool, error) {
	trimmed := strings.TrimSpace(proof)
	if trimmed == "" {
		return nil, false, fmt.Errorf("proof is empty")
	}

	// 1) JSON/protojson (frontend-friendly)
	if strings.HasPrefix(trimmed, "{") {
		var parsed identv1.ZKPProof
		if err := protojson.Unmarshal([]byte(trimmed), &parsed); err == nil {
			return &parsed, true, nil
		}
	}

	// 2) Base64-encoded protobuf (binary-safe transport)
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil {
		var parsed identv1.ZKPProof
		if err := proto.Unmarshal(decoded, &parsed); err == nil {
			return &parsed, true, nil
		}
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil {
		var parsed identv1.ZKPProof
		if err := proto.Unmarshal(decoded, &parsed); err == nil {
			return &parsed, true, nil
		}
	}

	return nil, false, nil
}

// VerificationRecord stores information about a verification.
// Kept as a Go struct since no proto message exists for it yet.
type VerificationRecord struct {
	Address          string                 `json:"address"`
	ProviderID       string                 `json:"provider_id"`
	VerificationTime *timestamppb.Timestamp `json:"verification_time"`
	ExpirationTime   *timestamppb.Timestamp `json:"expiration_time"`
	Nullifier        []byte                 `json:"nullifier"`
	IdentityHash     string                 `json:"identity_hash"`
}

// VerifyProvider validates a verification provider.
func (k Keeper) VerifyProvider(ctx sdk.Context, providerID string) error {
	provider, err := k.GetVerificationProvider(ctx, providerID)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	if !provider.IsAccredited {
		return fmt.Errorf("provider is not accredited: %s", providerID)
	}

	if err := k.ValidateProviderAccreditation(ctx, provider); err != nil {
		return fmt.Errorf("invalid provider accreditation: %w", err)
	}

	return nil
}

// SetAccreditationRecord stores an accreditation record by hash.
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

// ValidateProviderAccreditation validates provider's accreditation.
func (k Keeper) ValidateProviderAccreditation(ctx sdk.Context, provider *identv1.VerificationProvider) error {
	if provider.AccreditationHash == "" {
		return fmt.Errorf("provider has no accreditation hash")
	}

	store := ctx.KVStore(k.storeKey)
	accreditationKey := types.GetAccreditationKey(provider.AccreditationHash)

	if !store.Has(accreditationKey) {
		return fmt.Errorf("accreditation not found: %s", provider.AccreditationHash)
	}

	accreditationBz := store.Get(accreditationKey)
	if accreditationBz == nil {
		return fmt.Errorf("accreditation data is empty")
	}

	var accreditationData map[string]interface{}
	if err := json.Unmarshal(accreditationBz, &accreditationData); err != nil {
		return fmt.Errorf("invalid accreditation format: %w", err)
	}

	if valid, ok := accreditationData["valid"].(bool); !ok || !valid {
		return fmt.Errorf("accreditation is not valid")
	}

	return nil
}

// GetVerificationProvider retrieves a verification provider using protobuf encoding.
func (k Keeper) GetVerificationProvider(ctx sdk.Context, providerID string) (*identv1.VerificationProvider, error) {
	store := ctx.KVStore(k.storeKey)
	providerKey := types.GetProviderKey(providerID)

	bz := store.Get(providerKey)
	if bz == nil {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}

	var provider identv1.VerificationProvider
	if err := proto.Unmarshal(bz, &provider); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider: %w", err)
	}

	return &provider, nil
}

// SetVerificationProvider stores a verification provider using protobuf encoding.
func (k Keeper) SetVerificationProvider(ctx sdk.Context, provider *identv1.VerificationProvider) error {
	store := ctx.KVStore(k.storeKey)
	providerKey := types.GetProviderKey(provider.ProviderId)

	providerBz, err := proto.Marshal(provider)
	if err != nil {
		return fmt.Errorf("failed to marshal provider: %w", err)
	}

	store.Set(providerKey, providerBz)
	return nil
}

// GetAllVerificationProviders returns all stored verification providers.
func (k Keeper) GetAllVerificationProviders(ctx sdk.Context) ([]*identv1.VerificationProvider, error) {
	store := ctx.KVStore(k.storeKey)
	providerStore := prefix.NewStore(store, types.ProviderKeyPrefix)
	var list []*identv1.VerificationProvider
	iter := providerStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var p identv1.VerificationProvider
		if err := proto.Unmarshal(iter.Value(), &p); err != nil {
			continue
		}
		list = append(list, &p)
	}
	return list, nil
}

// CheckVerificationExpiration checks if a verification has expired.
func (k Keeper) CheckVerificationExpiration(ctx sdk.Context, address string) error {
	store := ctx.KVStore(k.storeKey)
	verificationKey := types.GetVerificationRecordKey(address)

	bz := store.Get(verificationKey)
	if bz == nil {
		return nil
	}

	var record VerificationRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return nil
	}

	if record.ExpirationTime != nil {
		if ctx.BlockTime().After(record.ExpirationTime.AsTime()) {
			return fmt.Errorf("verification has expired for address: %s", address)
		}
	}

	return nil
}

// StoreVerificationRecord stores a verification record.
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

// EnhancedNullifierCheck provides enhanced protection against nullifier reuse.
func (k Keeper) EnhancedNullifierCheck(ctx sdk.Context, nullifier []byte, address string) error {
	store := ctx.KVStore(k.storeKey)
	nullifierKey := types.GetNullifierKey(nullifier)

	if store.Has(nullifierKey) {
		existingBz := store.Get(nullifierKey)
		if existingBz != nil {
			var existingRecord map[string]interface{}
			if err := json.Unmarshal(existingBz, &existingRecord); err == nil {
				if existingAddr, ok := existingRecord["address"].(string); ok {
					if existingAddr != address {
						return fmt.Errorf("nullifier already used by different address: %s", existingAddr)
					}
					return nil
				}
			}
		}
		return fmt.Errorf("nullifier already used - identity already verified")
	}

	nullifierRecord := map[string]interface{}{
		"address":      address,
		"nullifier":    nullifier,
		"timestamp":    ctx.BlockTime().Unix(),
		"block_height": ctx.BlockHeight(),
	}

	nullifierBz, err := json.Marshal(nullifierRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal nullifier record: %w", err)
	}

	store.Set(nullifierKey, nullifierBz)
	return nil
}

// VerifyZKProofIntegrity provides replay-attack protection for ZKP proofs.
// Cryptographic verification is deferred until ZKP library integration.
func (k Keeper) VerifyZKProofIntegrity(ctx sdk.Context, proof string, providerID string, address string) error {
	if len(proof) < 16 {
		return fmt.Errorf("proof too short, minimum 16 bytes required")
	}

	parsedProof, isStructuredProof, err := parseZKPProofString(proof)
	if err != nil {
		return fmt.Errorf("invalid proof format: %w", err)
	}

	proofHashBytes := sha256.Sum256([]byte(proof))
	proofHashHex := hex.EncodeToString(proofHashBytes[:])
	if isStructuredProof {
		if parsedProof.ProofData == "" || parsedProof.PublicInputs == "" {
			return fmt.Errorf("structured proof must include proof_data and public_inputs")
		}
		expectedHash := computeStructuredProofHash(parsedProof.ProofData, parsedProof.PublicInputs, parsedProof.VerificationKey)
		if !strings.EqualFold(expectedHash, parsedProof.ProofHash) {
			return fmt.Errorf("structured proof hash mismatch")
		}
		if parsedProof.CreatedAt == nil {
			return fmt.Errorf("structured proof must include created_at")
		}
		proofTime := parsedProof.CreatedAt.AsTime()
		if proofTime.After(ctx.BlockTime().Add(2 * time.Minute)) {
			return fmt.Errorf("proof timestamp is in the future")
		}
		if ctx.BlockTime().Sub(proofTime) > 24*time.Hour {
			return fmt.Errorf("proof has expired")
		}

		decodedHash, err := hex.DecodeString(strings.ToLower(parsedProof.ProofHash))
		if err == nil && len(decodedHash) == sha256.Size {
			copy(proofHashBytes[:], decodedHash)
			proofHashHex = strings.ToLower(parsedProof.ProofHash)
		}
	}

	proofKey := types.GetProofKey(proofHashBytes[:])
	store := ctx.KVStore(k.storeKey)

	if store.Has(proofKey) {
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

	if isStructuredProof && providerID != "" && parsedProof.ProviderSignature != "" {
		provider, err := k.GetVerificationProvider(ctx, providerID)
		if err != nil {
			return fmt.Errorf("provider signature verification failed: %w", err)
		}
		pubKey, err := decodeProviderPublicKey(provider.ProviderPublicKey)
		if err != nil {
			return fmt.Errorf("provider signature verification failed: %w", err)
		}
		sig, err := base64.StdEncoding.DecodeString(parsedProof.ProviderSignature)
		if err != nil {
			if sigHex, hexErr := hex.DecodeString(parsedProof.ProviderSignature); hexErr == nil {
				sig = sigHex
			} else {
				return fmt.Errorf("provider signature is not valid base64/hex")
			}
		}
		payload := []byte(parsedProof.ProofHash + "|" + parsedProof.PublicInputs + "|" + parsedProof.ProofData + "|" + parsedProof.VerificationKey)
		if !ed25519.Verify(pubKey, payload, sig) {
			return fmt.Errorf("provider signature verification failed: invalid signature")
		}
	}

	proofRecord := map[string]interface{}{
		"address":      address,
		"provider_id":  providerID,
		"proof_hash":   proofHashHex,
		"timestamp":    ctx.BlockTime().Unix(),
		"block_height": ctx.BlockHeight(),
		"is_structured": isStructuredProof,
	}

	proofBz, err := json.Marshal(proofRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal proof record: %w", err)
	}

	store.Set(proofKey, proofBz)
	return nil
}

// ValidateVerificationRequest performs comprehensive validation of a verification request.
func (k Keeper) ValidateVerificationRequest(ctx sdk.Context, address string, zkpProof string, providerID string, desiredRole identv1.Role) error {
	if address == "" {
		return types.ErrEmptyAddress
	}

	if zkpProof == "" {
		return fmt.Errorf("ZKP proof cannot be empty")
	}

	if providerID != "" {
		if err := k.VerifyProvider(ctx, providerID); err != nil {
			return fmt.Errorf("invalid verification provider: %w", err)
		}
	}

	if err := k.CheckVerificationExpiration(ctx, address); err != nil {
		return err
	}

	if err := k.ValidateRoleChoice(ctx, address, desiredRole); err != nil {
		return err
	}

	if err := k.VerifyZKProofIntegrity(ctx, zkpProof, providerID, address); err != nil {
		return fmt.Errorf("ZKP proof integrity check failed: %w", err)
	}

	return nil
}
