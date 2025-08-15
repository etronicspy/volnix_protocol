package keeper

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
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

// GetParams returns the current parameters for the consensus module
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var params types.Params
	k.paramstore.GetParamSet(ctx, types.NewConsensusParams(&params))
	return params
}

// SetParams sets the parameters for the consensus module
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, types.NewConsensusParams(&params))
}

// SelectBlockCreator selects the next block creator using PoVB consensus
func (k Keeper) SelectBlockCreator(ctx sdk.Context) (string, error) {
	// Get all active validators with their ANT balances
	validators, err := k.GetActiveValidators(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get active validators: %w", err)
	}

	if len(validators) == 0 {
		return "", fmt.Errorf("no active validators available")
	}

	// Simple random selection for now
	rand.Seed(time.Now().UnixNano())
	selectedIndex := rand.Intn(len(validators))
	selectedValidator := validators[selectedIndex]
	
	return selectedValidator.Validator, nil
}

// GetActiveValidators retrieves only active validators
func (k Keeper) GetActiveValidators(ctx sdk.Context) ([]*types.Validator, error) {
	// For now, return a placeholder validator
	return []*types.Validator{
		{
			Validator: "vxvaloper1placeholder",
			Status:    consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
		},
	}, nil
}

// BeginBlocker processes events at the beginning of each block
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	// Select block creator for next block
	blockCreator, err := k.SelectBlockCreator(ctx)
	if err != nil {
		return fmt.Errorf("failed to select block creator: %w", err)
	}
	
	// Store block creator for this block
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBlockCreatorSelected,
			sdk.NewAttribute(types.AttributeKeyBlockCreator, blockCreator),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)
	
	return nil
}

// EndBlocker processes events at the end of each block
func (k Keeper) EndBlocker(ctx sdk.Context) error {
	// For now, just return success
	return nil
}

// InitGenesis initializes the consensus module's genesis state
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, *genState.Params)
	// Initialize validators if needed
}

// ExportGenesis exports the consensus module's genesis state
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()
	params := k.GetParams(ctx)
	genesis.Params = &params
	return genesis
}

// MsgServerImpl implements the MsgServer interface for the consensus module
type MsgServerImpl struct {
	Keeper
	consensusv1.UnimplementedMsgServer
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
func NewMsgServerImpl(keeper Keeper) consensusv1.MsgServer {
	return &MsgServerImpl{Keeper: keeper}
}

// SelectBlockCreator implements the MsgServer interface
func (k MsgServerImpl) SelectBlockCreator(ctx context.Context, msg *consensusv1.MsgSelectBlockCreator) (*consensusv1.MsgSelectBlockCreatorResponse, error) {
	// For now, just return a placeholder response
	return &consensusv1.MsgSelectBlockCreatorResponse{
		SelectedValidator: "vxvaloper1placeholder",
	}, nil
}
