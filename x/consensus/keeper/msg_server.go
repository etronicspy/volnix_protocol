package keeper

import (
	"context"
	"fmt"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

var consensusModuleAuthority = authtypes.NewModuleAddress(types.ModuleName).String()

// MsgServer implements consensus module Msg service.
type MsgServer struct {
	k Keeper
	consensusv1.UnimplementedMsgServer
}

// NewMsgServer constructs a new MsgServer.
func NewMsgServer(k Keeper) MsgServer { return MsgServer{k: k} }

var _ consensusv1.MsgServer = MsgServer{}

// UpdateConsensusState updates the consensus state
func (s MsgServer) UpdateConsensusState(ctx context.Context, req *consensusv1.MsgUpdateConsensusState) (*consensusv1.MsgUpdateConsensusStateResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.Authority != consensusModuleAuthority {
		return nil, types.ErrUnauthorized
	}

	err := s.k.UpdateConsensusState(sdkCtx, req.CurrentHeight, req.TotalAntBurned, req.ActiveValidators)
	if err != nil {
		return nil, err
	}

	return &consensusv1.MsgUpdateConsensusStateResponse{}, nil
}

// SetValidatorWeight sets validator weight
func (s MsgServer) SetValidatorWeight(ctx context.Context, req *consensusv1.MsgSetValidatorWeight) (*consensusv1.MsgSetValidatorWeightResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.Authority != consensusModuleAuthority {
		return nil, types.ErrUnauthorized
	}

	err := s.k.SetValidatorWeight(sdkCtx, req.Validator, req.Weight)
	if err != nil {
		return nil, err
	}

	return &consensusv1.MsgSetValidatorWeightResponse{}, nil
}

// DeclarePerHeightBurn handles a validator's burn declaration for the current block height (§5.4).
// Validator chooses s_i (priority stake) and b_i (burn for fee share) with s_i + b_i ≤ L_i.
func (s MsgServer) DeclarePerHeightBurn(ctx context.Context, req *consensusv1.MsgDeclarePerHeightBurn) (*consensusv1.MsgDeclarePerHeightBurnResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	if req.Validator == "" {
		return nil, fmt.Errorf("validator address is required")
	}

	// Verify the validator is active
	v, err := s.k.GetValidator(sdkCtx, req.Validator)
	if err != nil {
		return nil, fmt.Errorf("validator not found: %w", err)
	}
	if v.Status != consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE {
		return nil, fmt.Errorf("validator is not active")
	}

	burn := &consensusv1.PerHeightBurn{
		Validator:   req.Validator,
		BlockHeight: height,
		SI:          req.SI,
		BI:          req.BI,
		IncI:        "0", // will be computed in ProcessPerHeightBurns
	}

	s.k.StorePendingBurn(sdkCtx, burn)

	return &consensusv1.MsgDeclarePerHeightBurnResponse{
		IncI:    "0",
		Success: true,
	}, nil
}

// RegisterConsensusMapping registers a consensus_pubkey <-> account mapping (§4.2).
// This creates the explicit, deterministic mapping between CometBFT consensus address
// and the account address where LZN and role are tracked.
func (s MsgServer) RegisterConsensusMapping(ctx context.Context, req *consensusv1.MsgRegisterConsensusMapping) (*consensusv1.MsgRegisterConsensusMappingResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if req.AccountAddress == "" {
		return nil, fmt.Errorf("account address is required")
	}
	if len(req.ConsensusPubkey) == 0 {
		return nil, fmt.Errorf("consensus pubkey is required")
	}

	// Derive consensus address from Ed25519 pubkey
	consAddr := sdk.ConsAddress(req.ConsensusPubkey[:20]).String()

	// Store mapping
	store := sdkCtx.KVStore(s.k.storeKey)
	mappingKey := []byte(fmt.Sprintf("mapping/cons/%s", consAddr))
	store.Set(mappingKey, []byte(req.AccountAddress))
	reverseMappingKey := []byte(fmt.Sprintf("mapping/acct/%s", req.AccountAddress))
	store.Set(reverseMappingKey, []byte(consAddr))

	// Update the validator record with consensus address
	v, err := s.k.GetValidator(sdkCtx, req.AccountAddress)
	if err == nil && v != nil {
		v.ConsensusAddress = consAddr
		s.k.SetValidator(sdkCtx, v)
	}

	return &consensusv1.MsgRegisterConsensusMappingResponse{
		ConsensusAddress: consAddr,
		Success:          true,
	}, nil
}
