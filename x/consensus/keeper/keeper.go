package keeper

import (
	"fmt"
	"math/rand"
	"time"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

type (
	Keeper struct {
		cdc      codec.BinaryCodec
		storeKey storetypes.StoreKey
		memKey   storetypes.StoreKey
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
) *Keeper {
	return &Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		memKey:   memKey,
	}
}

// SelectBlockCreator selects the next block creator using PoVB consensus
func (k Keeper) SelectBlockCreator(ctx sdk.Context, blockHeight int64) (*types.BlockCreator, error) {
	// Get all active validators
	validators := k.GetActiveValidators(ctx)
	if len(validators) == 0 {
		return nil, fmt.Errorf("no active validators available")
	}

	// Calculate activity scores for all validators
	activityScores := k.CalculateActivityScores(ctx, validators)
	if len(activityScores) == 0 {
		return nil, fmt.Errorf("failed to calculate activity scores")
	}

	// Select block creator using weighted lottery based on activity scores
	selectedValidator := k.SelectValidatorByWeightedLottery(ctx, activityScores)
	if selectedValidator == nil {
		return nil, fmt.Errorf("failed to select validator")
	}

	// Create BlockCreator record
	blockCreator := &types.BlockCreator{
		Validator:     selectedValidator.Validator,
		AntBalance:    selectedValidator.AntBalance,
		ActivityScore: selectedValidator.ActivityScore,
		BurnAmount:    "0", // Will be set when block is created
		BlockHeight:   uint64(blockHeight),
		SelectionTime: timestamppb.Now(),
	}

	// Store the block creator
	k.SetBlockCreator(ctx, blockCreator)

	return blockCreator, nil
}

// CalculateActivityScores calculates activity scores for all validators
func (k Keeper) CalculateActivityScores(ctx sdk.Context, validators []*types.Validator) []*types.ActivityScore {
	var activityScores []*types.ActivityScore

	for _, validator := range validators {
		// Get ANT balance
		antBalance := k.GetValidatorANTBalance(ctx, validator.Validator)
		
		// Get blocks created count
		blocksCreated := validator.TotalBlocksCreated
		
		// Get transactions processed (simplified for now)
		transactionsProcessed := uint64(0)
		
		// Calculate activity score based on ANT balance and activity
		score := k.CalculateValidatorScore(antBalance, blocksCreated, transactionsProcessed)
		
		activityScore := &types.ActivityScore{
			Validator:            validator.Validator,
			Score:               score,
			AntBalance:          antBalance,
			BlocksCreated:       blocksCreated,
			TransactionsProcessed: transactionsProcessed,
			LastUpdate:          timestamppb.Now(),
		}
		
		activityScores = append(activityScores, activityScore)
		
		// Update validator's activity score
		validator.ActivityScore = fmt.Sprintf("%d", score)
		k.SetValidator(ctx, validator)
	}

	return activityScores
}

// BeginBlocker processes events at the beginning of each block
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	// Select block creator for next block
	blockHeight := ctx.BlockHeight() + 1
	blockCreator, err := k.SelectBlockCreator(ctx, blockHeight)
	if err != nil {
		return fmt.Errorf("failed to select block creator: %w", err)
	}
	
	// Emit event for block creator selection
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"block_creator_selected",
			sdk.NewAttribute("validator", blockCreator.Validator),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", blockCreator.BlockHeight)),
			sdk.NewAttribute("activity_score", blockCreator.ActivityScore),
		),
	)
	
	return nil
}

// EndBlocker processes events at the end of each block
func (k Keeper) EndBlocker(ctx sdk.Context) error {
	// Update activity scores and process burn proofs
	// This will be implemented in future stages
	return nil
}

// SelectValidatorByWeightedLottery selects validator using weighted lottery
func (k Keeper) SelectValidatorByWeightedLottery(ctx sdk.Context, activityScores []*types.ActivityScore) *types.Validator {
	if len(activityScores) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := uint64(0)
	for _, score := range activityScores {
		totalWeight += score.Score
	}

	if totalWeight == 0 {
		// Fallback to random selection
		rand.Seed(time.Now().UnixNano())
		randomIndex := rand.Intn(len(activityScores))
		selectedScore := activityScores[randomIndex]
		return k.GetValidator(ctx, selectedScore.Validator)
	}

	// Generate random number
	rand.Seed(time.Now().UnixNano())
	randomWeight := rand.Uint64() % totalWeight

	// Find validator based on weight
	currentWeight := uint64(0)
	for _, score := range activityScores {
		currentWeight += score.Score
		if randomWeight < currentWeight {
			return k.GetValidator(ctx, score.Validator)
		}
	}

	// Fallback to first validator
	return k.GetValidator(ctx, activityScores[0].Validator)
}

// CalculateValidatorScore calculates the activity score for a validator
func (k Keeper) CalculateValidatorScore(antBalance uint64, blocksCreated uint64, transactionsProcessed uint64) uint64 {
	// Base score from ANT balance (1 point per 1000 ANT)
	baseScore := antBalance / 1000
	
	// Bonus for blocks created (10 points per block)
	blockBonus := blocksCreated * 10
	
	// Bonus for transactions processed (1 point per 100 transactions)
	txBonus := transactionsProcessed / 100
	
	// Total score
	totalScore := baseScore + blockBonus + txBonus
	
	// Ensure minimum score
	if totalScore < 100 {
		totalScore = 100
	}
	
	return totalScore
}

// GetValidatorANTBalance gets the ANT balance for a validator
func (k Keeper) GetValidatorANTBalance(ctx sdk.Context, validator string) uint64 {
	// This would integrate with the anteil module
	// For now, return a default value
	return 1000000 // 1M ANT
}

// GetActiveValidators gets all active validators
func (k Keeper) GetActiveValidators(ctx sdk.Context) []*types.Validator {
	var validators []*types.Validator
	
	store := ctx.KVStore(k.storeKey)
	validatorStore := prefix.NewStore(store, types.KeyPrefix(types.ValidatorKey))
	
	pageRes, err := query.Paginate(validatorStore, &query.PageRequest{Limit: 1000}, func(key []byte, value []byte) error {
		var validator types.Validator
		if err := k.cdc.Unmarshal(value, &validator); err != nil {
			return err
		}
		
		if validator.Status == 1 { // VALIDATOR_STATUS_ACTIVE
			validators = append(validators, &validator)
		}
		
		return nil
	})
	
	if err != nil {
		return nil
	}
	
	_ = pageRes
	return validators
}

// SetBlockCreator stores a block creator
func (k Keeper) SetBlockCreator(ctx sdk.Context, blockCreator *types.BlockCreator) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshal(blockCreator)
	store.Set(types.KeyPrefix(types.BlockCreatorKey+blockCreator.Validator), b)
}

// GetBlockCreator gets a block creator by validator
func (k Keeper) GetBlockCreator(ctx sdk.Context, validator string) (*types.BlockCreator, error) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.KeyPrefix(types.BlockCreatorKey + validator))
	if b == nil {
		return nil, fmt.Errorf("block creator not found for validator: %s", validator)
	}
	
	var blockCreator types.BlockCreator
	k.cdc.MustUnmarshal(b, &blockCreator)
	return &blockCreator, nil
}

// SetValidator stores a validator
func (k Keeper) SetValidator(ctx sdk.Context, validator *types.Validator) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshal(validator)
	store.Set(types.KeyPrefix(types.ValidatorKey+validator.Validator), b)
}

// GetValidator gets a validator by address
func (k Keeper) GetValidator(ctx sdk.Context, validator string) *types.Validator {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.KeyPrefix(types.ValidatorKey + validator))
	if b == nil {
		return nil
	}
	
	var val types.Validator
	k.cdc.MustUnmarshal(b, &val)
	return &val
}

// InitGenesis initializes the consensus module genesis state
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	// Set params
	k.SetParams(ctx, genState.Params)
	
	// Set validators
	for _, validator := range genState.Validators {
		k.SetValidator(ctx, validator)
	}
	
	// Set block creators
	for _, blockCreator := range genState.BlockCreators {
		k.SetBlockCreator(ctx, blockCreator)
	}
	
	// Set burn proofs
	for _, burnProof := range genState.BurnProofs {
		k.SetBurnProof(ctx, burnProof)
	}
	
	// Set activity scores
	for _, activityScore := range genState.ActivityScores {
		k.SetActivityScore(ctx, activityScore)
	}
}

// ExportGenesis exports the consensus module genesis state
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()
	
	// Get params
	params := k.GetParams(ctx)
	genesis.Params = params
	
	// Get validators
	validators := k.GetAllValidators(ctx)
	genesis.Validators = validators
	
	// Get block creators
	blockCreators := k.GetAllBlockCreators(ctx)
	genesis.BlockCreators = blockCreators
	
	// Get burn proofs
	burnProofs := k.GetAllBurnProofs(ctx)
	genesis.BurnProofs = burnProofs
	
	// Get activity scores
	activityScores := k.GetAllActivityScores(ctx)
	genesis.ActivityScores = activityScores
	
	return genesis
}

// SetParams sets the consensus module parameters
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshal(params)
	store.Set(types.ParamsKey, b)
}

// GetParams gets the consensus module parameters
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.ParamsKey)
	if b == nil {
		return types.DefaultParams()
	}
	
	var params types.Params
	k.cdc.MustUnmarshal(b, &params)
	return &params
}

// SetBurnProof stores a burn proof
func (k Keeper) SetBurnProof(ctx sdk.Context, burnProof *types.BurnProof) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshal(burnProof)
	store.Set(types.KeyPrefix(types.BurnProofKey+burnProof.Validator), b)
}

// SetActivityScore stores an activity score
func (k Keeper) SetActivityScore(ctx sdk.Context, activityScore *types.ActivityScore) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshal(activityScore)
	store.Set(types.KeyPrefix(types.ActivityScoreKey+activityScore.Validator), b)
}

// GetAllValidators gets all validators
func (k Keeper) GetAllValidators(ctx sdk.Context) []*types.Validator {
	var validators []*types.Validator
	
	store := ctx.KVStore(k.storeKey)
	validatorStore := prefix.NewStore(store, types.KeyPrefix(types.ValidatorKey))
	
	iterator := validatorStore.Iterator(nil, nil)
	defer iterator.Close()
	
	for ; iterator.Valid(); iterator.Next() {
		var validator types.Validator
		k.cdc.MustUnmarshal(iterator.Value(), &validator)
		validators = append(validators, &validator)
	}
	
	return validators
}

// GetAllBlockCreators gets all block creators
func (k Keeper) GetAllBlockCreators(ctx sdk.Context) []*types.BlockCreator {
	var blockCreators []*types.BlockCreator
	
	store := ctx.KVStore(k.storeKey)
	blockCreatorStore := prefix.NewStore(store, types.KeyPrefix(types.BlockCreatorKey))
	
	iterator := blockCreatorStore.Iterator(nil, nil)
	defer iterator.Close()
	
	for ; iterator.Valid(); iterator.Next() {
		var blockCreator types.BlockCreator
		k.cdc.MustUnmarshal(iterator.Value(), &blockCreator)
		blockCreators = append(blockCreators, &blockCreator)
	}
	
	return blockCreators
}

// GetAllBurnProofs gets all burn proofs
func (k Keeper) GetAllBurnProofs(ctx sdk.Context) []*types.BurnProof {
	var burnProofs []*types.BurnProof
	
	store := ctx.KVStore(k.storeKey)
	burnProofStore := prefix.NewStore(store, types.KeyPrefix(types.BurnProofKey))
	
	iterator := burnProofStore.Iterator(nil, nil)
	defer iterator.Close()
	
	for ; iterator.Valid(); iterator.Next() {
		var burnProof types.BurnProof
		k.cdc.MustUnmarshal(iterator.Value(), &burnProof)
		burnProofs = append(burnProofs, &burnProof)
	}
	
	return burnProofs
}

// GetAllActivityScores gets all activity scores
func (k Keeper) GetAllActivityScores(ctx sdk.Context) []*types.ActivityScore {
	var activityScores []*types.ActivityScore
	
	store := ctx.KVStore(k.storeKey)
	activityScoreStore := prefix.NewStore(store, types.KeyPrefix(types.ActivityScoreKey))
	
	iterator := activityScoreStore.Iterator(nil, nil)
	defer iterator.Close()
	
	for ; iterator.Valid(); iterator.Next() {
		var activityScore types.ActivityScore
		k.cdc.MustUnmarshal(iterator.Value(), &activityScore)
		activityScores = append(activityScores, &activityScore)
	}
	
	return activityScores
}
