package keeper

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
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
	var consensusParams types.ConsensusParams
	k.paramstore.GetParamSet(ctx, &consensusParams)
	if consensusParams.Params == nil {
		return *types.DefaultParams()
	}
	return *consensusParams.Params
}

// SetParams sets the parameters for the consensus module
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	consensusParams := types.NewConsensusParams(&params)
	k.paramstore.SetParamSet(ctx, consensusParams)
}

// SelectBlockProducer selects a block producer from a list of validators
func (k Keeper) SelectBlockProducer(ctx sdk.Context, validators []string) (string, error) {
	if len(validators) == 0 {
		return "", types.ErrNoValidators
	}

	// Simple random selection for now
	selectedIndex := rand.Intn(len(validators))
	return validators[selectedIndex], nil
}

// SelectBlockCreator selects the next block creator using weighted lottery
func (k Keeper) SelectBlockCreator(ctx sdk.Context, height uint64) (*consensusv1.BlockCreator, error) {
	validators := k.GetAllValidators(ctx)
	if len(validators) == 0 {
		return nil, types.ErrNoValidators
	}

	// Calculate weights based on ANT balance and activity
	weights := make([]uint64, len(validators))
	totalWeight := uint64(0)

	for i, validator := range validators {
		// Convert ANT balance to weight
		antBalance, err := strconv.ParseUint(validator.AntBalance, 10, 64)
		if err != nil {
			antBalance = 0
		}

		// Convert activity score to weight
		activityScore, err := strconv.ParseUint(validator.ActivityScore, 10, 64)
		if err != nil {
			activityScore = 0
		}

		// Weight = ANT balance + activity score
		weights[i] = antBalance + activityScore
		totalWeight += weights[i]
	}

	if totalWeight == 0 {
		// If no weights, select randomly
		selectedIndex := rand.Intn(len(validators))
		selectedValidator := validators[selectedIndex]

		blockCreator := &consensusv1.BlockCreator{
			Validator:     selectedValidator.Validator,
			AntBalance:    selectedValidator.AntBalance,
			ActivityScore: selectedValidator.ActivityScore,
			BurnAmount:    "0",
			BlockHeight:   height,
			SelectionTime: timestamppb.Now(),
		}

		k.SetBlockCreator(ctx, blockCreator)
		return blockCreator, nil
	}

	// Weighted random selection
	randomWeight := rand.Uint64() % totalWeight
	currentWeight := uint64(0)

	for i, weight := range weights {
		currentWeight += weight
		if randomWeight < currentWeight {
			selectedValidator := validators[i]

			blockCreator := &consensusv1.BlockCreator{
				Validator:     selectedValidator.Validator,
				AntBalance:    selectedValidator.AntBalance,
				ActivityScore: selectedValidator.ActivityScore,
				BurnAmount:    "0",
				BlockHeight:   height,
				SelectionTime: timestamppb.Now(),
			}

			k.SetBlockCreator(ctx, blockCreator)
			return blockCreator, nil
		}
	}

	// Fallback to first validator
	selectedValidator := validators[0]
	blockCreator := &consensusv1.BlockCreator{
		Validator:     selectedValidator.Validator,
		AntBalance:    selectedValidator.AntBalance,
		ActivityScore: selectedValidator.ActivityScore,
		BurnAmount:    "0",
		BlockHeight:   height,
		SelectionTime: timestamppb.Now(),
	}

	k.SetBlockCreator(ctx, blockCreator)
	return blockCreator, nil
}

// CalculateBlockTime calculates dynamic block time based on ANT activity
func (k Keeper) CalculateBlockTime(ctx sdk.Context, antAmount string) (time.Duration, error) {
	params := k.GetParams(ctx)

	antAmountInt, err := strconv.ParseUint(antAmount, 10, 64)
	if err != nil {
		return 0, types.ErrInvalidAntAmount
	}

	if antAmountInt == 0 {
		return 0, types.ErrInvalidAntAmount
	}

	// Parse base block time
	baseBlockTime, err := time.ParseDuration(params.BaseBlockTime)
	if err != nil {
		baseBlockTime = 5 * time.Second // Default fallback
	}

	// Calculate activity factor
	highThreshold := params.HighActivityThreshold
	lowThreshold := params.LowActivityThreshold

	var activityFactor float64
	if antAmountInt >= highThreshold {
		activityFactor = 0.5 // Faster blocks for high activity
	} else if antAmountInt >= lowThreshold {
		activityFactor = 0.75 // Moderate speed
	} else {
		activityFactor = 1.0 // Normal speed
	}

	// Calculate dynamic block time
	dynamicBlockTime := float64(baseBlockTime) * activityFactor

	return time.Duration(dynamicBlockTime), nil
}

// ProcessHalving processes halving event
func (k Keeper) ProcessHalving(ctx sdk.Context) error {
	store := ctx.KVStore(k.storeKey)
	halvingKey := types.KeyHalvingInfo()

	var halvingInfo types.HalvingInfo
	bz := store.Get(halvingKey)
	if bz == nil {
		// Initialize halving info if not exists
		halvingInfo = types.HalvingInfo{
			LastHalvingHeight: 0,
			NextHalvingHeight: 1000000, // Example: every 1M blocks
			HalvingInterval:   1000000,
		}
	} else {
		k.cdc.MustUnmarshal(bz, &halvingInfo)
	}

	currentHeight := uint64(ctx.BlockHeight())
	if currentHeight >= halvingInfo.NextHalvingHeight {
		// Process halving
		halvingInfo.LastHalvingHeight = halvingInfo.NextHalvingHeight
		halvingInfo.NextHalvingHeight += halvingInfo.HalvingInterval

		// Store updated halving info
		bz = k.cdc.MustMarshal(&halvingInfo)
		store.Set(halvingKey, bz)
	}

	return nil
}

// GetHalvingInfo returns halving information
func (k Keeper) GetHalvingInfo(ctx sdk.Context) (types.HalvingInfo, error) {
	store := ctx.KVStore(k.storeKey)
	halvingKey := types.KeyHalvingInfo()

	var halvingInfo types.HalvingInfo
	bz := store.Get(halvingKey)
	if bz == nil {
		// Return default halving info if not exists
		return types.HalvingInfo{
			LastHalvingHeight: 0,
			NextHalvingHeight: 100000,
			HalvingInterval:   100000,
		}, nil
	}

	k.cdc.MustUnmarshal(bz, &halvingInfo)
	return halvingInfo, nil
}

// SetHalvingInfo sets halving information
func (k Keeper) SetHalvingInfo(ctx sdk.Context, halvingInfo types.HalvingInfo) error {
	store := ctx.KVStore(k.storeKey)
	halvingKey := types.KeyHalvingInfo()

	bz := k.cdc.MustMarshal(&halvingInfo)
	store.Set(halvingKey, bz)
	return nil
}

// GetConsensusState returns current consensus state
func (k Keeper) GetConsensusState(ctx sdk.Context) (types.ConsensusState, error) {
	store := ctx.KVStore(k.storeKey)
	consensusKey := types.KeyConsensusState()

	var consensusState types.ConsensusState
	bz := store.Get(consensusKey)
	if bz == nil {
		// Return default consensus state if not exists
		return types.ConsensusState{
			CurrentHeight:    uint64(ctx.BlockHeight()),
			TotalAntBurned:   "0",
			LastBlockTime:    timestamppb.Now(),
			ActiveValidators: []string{},
		}, nil
	}

	k.cdc.MustUnmarshal(bz, &consensusState)
	return consensusState, nil
}

// SetConsensusState sets consensus state
func (k Keeper) SetConsensusState(ctx sdk.Context, state types.ConsensusState) error {
	store := ctx.KVStore(k.storeKey)
	consensusKey := types.KeyConsensusState()

	bz := k.cdc.MustMarshal(&state)
	store.Set(consensusKey, bz)
	return nil
}

// UpdateConsensusState updates consensus state
func (k Keeper) UpdateConsensusState(ctx sdk.Context, height uint64, totalAntBurned string, activeValidators []string) error {
	consensusState, err := k.GetConsensusState(ctx)
	if err != nil {
		return err
	}

	consensusState.CurrentHeight = height
	consensusState.TotalAntBurned = totalAntBurned
	consensusState.ActiveValidators = activeValidators

	return k.SetConsensusState(ctx, consensusState)
}

// GetValidatorWeight returns validator weight
func (k Keeper) GetValidatorWeight(ctx sdk.Context, validator string) (string, error) {
	if validator == "" {
		return "", types.ErrEmptyValidatorAddress
	}

	store := ctx.KVStore(k.storeKey)
	key := types.GetValidatorWeightKey(validator)
	bz := store.Get(key)
	if bz == nil {
		return "0", nil
	}

	var validatorWeight types.ValidatorWeight
	if err := k.cdc.Unmarshal(bz, &validatorWeight); err != nil {
		return "0", err
	}

	return validatorWeight.Weight, nil
}

// SetValidatorWeight sets validator weight
func (k Keeper) SetValidatorWeight(ctx sdk.Context, validator, weight string) error {
	if validator == "" {
		return types.ErrEmptyValidatorAddress
	}

	store := ctx.KVStore(k.storeKey)
	key := types.GetValidatorWeightKey(validator)

	// Create ValidatorWeight struct
	validatorWeight := types.ValidatorWeight{
		Validator: validator,
		Weight:    weight,
	}

	// Marshal and store
	bz, err := k.cdc.Marshal(&validatorWeight)
	if err != nil {
		return err
	}
	store.Set(key, bz)
	return nil
}

// GetAllValidatorWeights returns all validator weights
func (k Keeper) GetAllValidatorWeights(ctx sdk.Context) ([]types.ValidatorWeight, error) {
	store := ctx.KVStore(k.storeKey)
	prefix := types.KeyValidatorWeightPrefix

	var weights []types.ValidatorWeight
	iterator := store.Iterator(prefix, append(prefix, 0xFF))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var weight types.ValidatorWeight
		if err := k.cdc.Unmarshal(iterator.Value(), &weight); err != nil {
			continue // Skip invalid data
		}
		weights = append(weights, weight)
	}

	return weights, nil
}

// SetValidator sets a validator
func (k Keeper) SetValidator(ctx sdk.Context, validator *consensusv1.Validator) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetValidatorKey(validator.Validator)
	bz := k.cdc.MustMarshal(validator)
	store.Set(key, bz)
}

// GetValidator returns a validator
func (k Keeper) GetValidator(ctx sdk.Context, validator string) (*consensusv1.Validator, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetValidatorKey(validator)
	bz := store.Get(key)
	if bz == nil {
		return nil, fmt.Errorf("validator %s not found", validator)
	}

	var validatorData consensusv1.Validator
	err := k.cdc.Unmarshal(bz, &validatorData)
	if err != nil {
		return nil, err
	}

	return &validatorData, nil
}

// GetAllValidators returns all validators
func (k Keeper) GetAllValidators(ctx sdk.Context) []*consensusv1.Validator {
	store := ctx.KVStore(k.storeKey)
	iterator := store.Iterator(types.KeyValidatorPrefix, append(types.KeyValidatorPrefix, 0xFF))
	defer iterator.Close()

	var validators []*consensusv1.Validator
	for ; iterator.Valid(); iterator.Next() {
		var validator consensusv1.Validator
		k.cdc.MustUnmarshal(iterator.Value(), &validator)
		validators = append(validators, &validator)
	}

	return validators
}

// SetBlockCreator sets block creator
func (k Keeper) SetBlockCreator(ctx sdk.Context, blockCreator *consensusv1.BlockCreator) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetBlockCreatorKey(blockCreator.BlockHeight)
	bz := k.cdc.MustMarshal(blockCreator)
	store.Set(key, bz)
}

// GetBlockCreator returns block creator
func (k Keeper) GetBlockCreator(ctx sdk.Context, height uint64) (*consensusv1.BlockCreator, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetBlockCreatorKey(height)
	bz := store.Get(key)
	if bz == nil {
		return nil, fmt.Errorf("block creator for height %d not found", height)
	}

	var blockCreator consensusv1.BlockCreator
	err := k.cdc.Unmarshal(bz, &blockCreator)
	if err != nil {
		return nil, err
	}

	return &blockCreator, nil
}

// BeginBlocker processes begin block logic
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	// Update consensus state
	currentHeight := uint64(ctx.BlockHeight())
	validators := k.GetAllValidators(ctx)

	var activeValidators []string
	for _, validator := range validators {
		if validator.Status == consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE {
			activeValidators = append(activeValidators, validator.Validator)
		}
	}

	// Get current total ANT burned
	totalAntBurned, err := k.calculateTotalBurnedTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate total burned tokens: %w", err)
	}

	err = k.UpdateConsensusState(ctx, currentHeight, totalAntBurned, activeValidators)
	if err != nil {
		return err
	}

	// Process halving if needed
	err = k.ProcessHalving(ctx)
	if err != nil {
		return err
	}

	return nil
}

// EndBlocker processes end block logic
func (k Keeper) EndBlocker(ctx sdk.Context) error {
	// Update consensus state
	currentHeight := uint64(ctx.BlockHeight())
	validators := k.GetAllValidators(ctx)

	var activeValidators []string
	for _, validator := range validators {
		if validator.Status == consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE {
			activeValidators = append(activeValidators, validator.Validator)
		}
	}

	// Get current total ANT burned
	totalAntBurned, err := k.calculateTotalBurnedTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate total burned tokens: %w", err)
	}

	err = k.UpdateConsensusState(ctx, currentHeight, totalAntBurned, activeValidators)
	if err != nil {
		return err
	}

	return nil
}

// InitGenesis initializes genesis state
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, *genState.Params)

	for _, validator := range genState.Validators {
		k.SetValidator(ctx, validator)
	}

	for _, blockCreator := range genState.BlockCreators {
		k.SetBlockCreator(ctx, blockCreator)
	}

	// Set default halving info
	halvingInfo := types.HalvingInfo{
		LastHalvingHeight: 0,
		HalvingInterval:   210000,
		NextHalvingHeight: 210000,
	}
	err := k.SetHalvingInfo(ctx, halvingInfo)
	if err != nil {
		// Log error but don't fail genesis
		ctx.Logger().Error("failed to set default halving info", "error", err)
	}

	// Set default consensus state
	err = k.UpdateConsensusState(ctx, 0, "0", []string{})
	if err != nil {
		// Log error but don't fail genesis
		ctx.Logger().Error("failed to set default consensus state", "error", err)
	}
}

// ExportGenesis exports genesis state
func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	params := k.GetParams(ctx)
	validators := k.GetAllValidators(ctx)

	var blockCreators []*consensusv1.BlockCreator
	store := ctx.KVStore(k.storeKey)
	iterator := store.Iterator(types.KeyBlockCreatorPrefix, append(types.KeyBlockCreatorPrefix, 0xFF))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var blockCreator consensusv1.BlockCreator
		k.cdc.MustUnmarshal(iterator.Value(), &blockCreator)
		blockCreators = append(blockCreators, &blockCreator)
	}

	return types.GenesisState{
		Params:         &params,
		Validators:     validators,
		BlockCreators:  blockCreators,
		BurnProofs:     []*consensusv1.BurnProof{},
		ActivityScores: []*consensusv1.ActivityScore{},
	}
}
// calculateTotalBurnedTokens calculates the total amount of ANT tokens burned
func (k Keeper) calculateTotalBurnedTokens(ctx sdk.Context) (string, error) {
	// Get all validator weights
	weights, err := k.GetAllValidatorWeights(ctx)
	if err != nil {
		return "0", fmt.Errorf("failed to get validator weights: %w", err)
	}

	// Sum up all burned tokens (using weight as proxy for burned tokens)
	totalBurned := 0.0
	for _, weight := range weights {
		if weight.Weight != "" && weight.Weight != "0" {
			weightFloat, err := strconv.ParseFloat(weight.Weight, 64)
			if err != nil {
				continue // Skip invalid weights
			}
			totalBurned += weightFloat
		}
	}

	return fmt.Sprintf("%.8f", totalBurned), nil
}

