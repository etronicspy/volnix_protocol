package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

// LizenzKeeperInterface defines the interface for interacting with lizenz module
// This allows consensus module to get information about activated LZN and MOA status
// Note: We use interface{} for GetAllActivatedLizenz to avoid circular dependencies
// The actual type is []*lizenzv1.ActivatedLizenz, but we can't import it here
type LizenzKeeperInterface interface {
	GetAllActivatedLizenz(ctx sdk.Context) ([]interface{}, error) // Returns list of activated LZN ([]*lizenzv1.ActivatedLizenz)
	GetTotalActivatedLizenz(ctx sdk.Context) (string, error)      // Returns total activated LZN
	GetMOACompliance(ctx sdk.Context, validator string) (float64, error) // Returns MOA compliance ratio (0.0 to 1.0+)
	UpdateRewardStats(ctx sdk.Context, validator string, rewardAmount uint64, blockHeight uint64, moaCompliance float64, penaltyMultiplier float64, baseReward uint64) error // Updates reward statistics
}

// AnteilKeeperInterface defines the interface for interacting with anteil module
// This allows consensus module to check ANT balances and burn ANT tokens
// Note: We use interface{} for UserPosition to avoid circular dependencies
type AnteilKeeperInterface interface {
	GetUserPosition(ctx sdk.Context, user string) (interface{}, error) // Returns UserPosition (anteilv1.UserPosition)
	SetUserPosition(ctx sdk.Context, position interface{}) error      // Sets UserPosition
	UpdateUserPosition(ctx sdk.Context, user string, antBalance string, orderCount uint32) error // Updates ANT balance
}

// BankKeeperInterface defines the interface for interacting with bank module
// This allows consensus module to send WRT rewards to validators
type BankKeeperInterface interface {
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeKey     storetypes.StoreKey
		paramstore   paramtypes.Subspace
		lizenzKeeper LizenzKeeperInterface // Optional: for reward distribution
		anteilKeeper AnteilKeeperInterface // Optional: for ANT balance management
		bankKeeper   BankKeeperInterface   // Optional: for sending WRT rewards
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

// SetLizenzKeeper sets the lizenz keeper interface for reward distribution
func (k *Keeper) SetLizenzKeeper(lizenzKeeper LizenzKeeperInterface) {
	k.lizenzKeeper = lizenzKeeper
}

// SetAnteilKeeper sets the anteil keeper interface for ANT balance management
func (k *Keeper) SetAnteilKeeper(anteilKeeper AnteilKeeperInterface) {
	k.anteilKeeper = anteilKeeper
}

// SetBankKeeper sets the bank keeper interface for sending WRT rewards
func (k *Keeper) SetBankKeeper(bankKeeper BankKeeperInterface) {
	k.bankKeeper = bankKeeper
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

// SelectBlockCreator selects the next block creator using blind auction
// According to whitepaper: "Право на создание блока и получение комиссий разыгрывается в каждом раунде через 'слепой аукцион с взвешенной лотереей'"
func (k Keeper) SelectBlockCreator(ctx sdk.Context, height uint64) (*consensusv1.BlockCreator, error) {
	validators := k.GetAllValidators(ctx)
	if len(validators) == 0 {
		return nil, types.ErrNoValidators
	}

	// Try to get winner from blind auction
	auction, err := k.GetBlindAuction(ctx, height)
	if err == nil && auction != nil && auction.Phase == consensusv1.AuctionPhase_AUCTION_PHASE_COMPLETE && auction.Winner != "" {
		// Use winner from blind auction
		winnerValidator, err := k.GetValidator(ctx, auction.Winner)
		if err == nil && winnerValidator != nil {
			blockCreator := &consensusv1.BlockCreator{
				Validator:     auction.Winner,
				AntBalance:    winnerValidator.AntBalance,
				ActivityScore: winnerValidator.ActivityScore,
				BurnAmount:    auction.WinningBid,
				BlockHeight:   height,
				SelectionTime: timestamppb.Now(),
			}

			k.SetBlockCreator(ctx, blockCreator)
			return blockCreator, nil
		}
	}

	// Fallback to weighted lottery if no auction winner
	// Calculate weights based on ANT balance, activity, and activated LZN
	weights := make([]uint64, len(validators))
	totalWeight := uint64(0)

	// Get activated LZN for validators (if lizenz keeper is available)
	validatorLZN := make(map[string]uint64)
	if k.lizenzKeeper != nil {
		allLizenzs, err := k.lizenzKeeper.GetAllActivatedLizenz(ctx)
		if err == nil {
			for _, lizenzInterface := range allLizenzs {
				validator, amount, err := extractLizenzInfo(lizenzInterface)
				if err == nil {
					amountInt, err := strconv.ParseUint(amount, 10, 64)
					if err == nil {
						validatorLZN[validator] = amountInt
					}
				}
			}
		}
	}

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

		// Get activated LZN for this validator
		activatedLZN := validatorLZN[validator.Validator]

		// Weight = ANT balance + activity score + activated LZN
		// LZN is important as it represents validator's stake in the network
		weights[i] = antBalance + activityScore + activatedLZN
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
			
			// Emit event for block creator selection
			weight := "0"
			if weightStr, err := k.GetValidatorWeight(ctx, blockCreator.Validator); err == nil {
				weight = weightStr
			}
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeBlockCreatorSelected,
					sdk.NewAttribute(types.AttributeKeyBlockCreator, blockCreator.Validator),
					sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", height)),
					sdk.NewAttribute(types.AttributeKeyPower, weight),
				),
			)
			
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
	
	// Emit event for block creator selection
	// Get validator weight if available
	weight := "0"
	if weightStr, err := k.GetValidatorWeight(ctx, blockCreator.Validator); err == nil {
		weight = weightStr
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBlockCreatorSelected,
			sdk.NewAttribute(types.AttributeKeyBlockCreator, blockCreator.Validator),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", height)),
			sdk.NewAttribute(types.AttributeKeyPower, weight),
		),
	)
	
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

	// Calculate activity factor from params
	highThreshold := params.HighActivityThreshold
	lowThreshold := params.LowActivityThreshold

	var activityFactor float64
	if antAmountInt >= highThreshold {
		// Parse activity factor for high activity
		activityFactor, err = strconv.ParseFloat(params.ActivityFactorHigh, 64)
		if err != nil || activityFactor == 0 {
			activityFactor = 0.5 // Default fallback
		}
	} else if antAmountInt >= lowThreshold {
		// Parse activity factor for medium activity
		activityFactor, err = strconv.ParseFloat(params.ActivityFactorMedium, 64)
		if err != nil || activityFactor == 0 {
			activityFactor = 0.75 // Default fallback
		}
	} else {
		// Parse activity factor for normal activity
		activityFactor, err = strconv.ParseFloat(params.ActivityFactorNormal, 64)
		if err != nil || activityFactor == 0 {
			activityFactor = 1.0 // Default fallback
		}
	}

	// Calculate dynamic block time
	dynamicBlockTime := float64(baseBlockTime) * activityFactor

	return time.Duration(dynamicBlockTime), nil
}

// RecordBlockTime records the time for a block
// This is used to calculate average block time for adaptive halving
func (k Keeper) RecordBlockTime(ctx sdk.Context, height uint64) error {
	store := ctx.KVStore(k.storeKey)
	blockTimeKey := types.GetBlockTimeKey(height)
	
	// Store block time as timestamp
	blockTime := ctx.BlockTime()
	timeBz, err := blockTime.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal block time: %w", err)
	}
	
	store.Set(blockTimeKey, timeBz)
	
	// Update average block time
	return k.updateAverageBlockTime(ctx)
}

// GetAverageBlockTime calculates and returns the average block time
// Uses a sliding window of recent blocks (e.g., last 1000 blocks)
func (k Keeper) GetAverageBlockTime(ctx sdk.Context) (time.Duration, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AverageBlockTimeKey)
	
	if bz == nil {
		// Return default if not calculated yet
		return 5 * time.Second, nil
	}
	
	// Parse stored duration (as nanoseconds) - use JSON encoding for simple types
	var avgNanos int64
	if err := json.Unmarshal(bz, &avgNanos); err != nil {
		return 5 * time.Second, nil
	}
	
	return time.Duration(avgNanos), nil
}

// updateAverageBlockTime updates the average block time based on recent blocks
func (k Keeper) updateAverageBlockTime(ctx sdk.Context) error {
	currentHeight := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)
	windowSize := params.AverageBlockTimeWindowSize
	if windowSize == 0 {
		windowSize = 1000 // Default fallback
	}
	
	startHeight := uint64(0)
	if currentHeight > windowSize {
		startHeight = currentHeight - windowSize
	}
	
	var totalDuration time.Duration
	blockCount := uint64(0)
	store := ctx.KVStore(k.storeKey)
	
	// Calculate average from recent blocks
	for h := startHeight + 1; h <= currentHeight; h++ {
		blockTimeKey := types.GetBlockTimeKey(h)
		timeBz := store.Get(blockTimeKey)
		
		if timeBz == nil {
			continue // Skip if block time not recorded
		}
		
		var blockTime time.Time
		if err := blockTime.UnmarshalBinary(timeBz); err != nil {
			continue // Skip invalid times
		}
		
		// Get previous block time
		if h > startHeight+1 {
			prevTimeKey := types.GetBlockTimeKey(h - 1)
			prevTimeBz := store.Get(prevTimeKey)
			if prevTimeBz != nil {
				var prevTime time.Time
				if err := prevTime.UnmarshalBinary(prevTimeBz); err == nil {
					duration := blockTime.Sub(prevTime)
					totalDuration += duration
					blockCount++
				}
			}
		}
	}
	
	if blockCount == 0 {
		return nil // Not enough data yet
	}
	
	// Calculate average
	averageBlockTime := totalDuration / time.Duration(blockCount)
	
	// Store average - use JSON encoding for simple types
	avgNanos := int64(averageBlockTime)
	avgBz, err := json.Marshal(avgNanos)
	if err != nil {
		return fmt.Errorf("failed to marshal average block time: %w", err)
	}
	
	store.Set(types.AverageBlockTimeKey, avgBz)
	
	return nil
}

// ProcessHalving processes halving event
// According to whitepaper: "Халвинг происходит строго каждые N блоков, но реальная дата адаптируется к динамическому времени блока"
func (k Keeper) ProcessHalving(ctx sdk.Context) error {
	store := ctx.KVStore(k.storeKey)
	halvingKey := types.KeyHalvingInfo()

	var halvingInfo consensusv1.HalvingInfo
	bz := store.Get(halvingKey)
	if bz == nil {
		// Initialize halving info if not exists
		halvingInfo = consensusv1.HalvingInfo{
			LastHalvingHeight: 0,
			NextHalvingHeight: HalvingInterval, // 210,000 blocks
			HalvingInterval:   HalvingInterval,
		}
	} else {
		k.cdc.MustUnmarshal(bz, &halvingInfo)
	}

	currentHeight := uint64(ctx.BlockHeight())
	currentTime := ctx.BlockTime()
	
	// Note: RecordBlockTime is called in EndBlocker, so we don't need to call it here

	// Check if halving should occur
	if currentHeight >= halvingInfo.NextHalvingHeight {
		// Process halving
		halvingInfo.LastHalvingHeight = halvingInfo.NextHalvingHeight
		// TODO: Uncomment after proto generation
		// halvingInfo.LastHalvingDate = timestamppb.New(currentTime)
		halvingInfo.NextHalvingHeight += halvingInfo.HalvingInterval

		// Calculate estimated next halving date based on average block time
		avgBlockTime, err := k.GetAverageBlockTime(ctx)
		if err == nil && avgBlockTime > 0 {
			blocksRemaining := halvingInfo.NextHalvingHeight - currentHeight
			estimatedDuration := avgBlockTime * time.Duration(blocksRemaining)
			estimatedDate := currentTime.Add(estimatedDuration)
			// TODO: Uncomment after proto generation
			// halvingInfo.EstimatedNextHalvingDate = timestamppb.New(estimatedDate)
			
			ctx.Logger().Info("halving processed",
				"height", currentHeight,
				"next_halving_height", halvingInfo.NextHalvingHeight,
				"estimated_date", estimatedDate,
				"average_block_time", avgBlockTime)
		}

		// Store updated halving info
		bz = k.cdc.MustMarshal(&halvingInfo)
		store.Set(halvingKey, bz)
	} else {
		// Update estimated next halving date even if halving didn't occur
		avgBlockTime, err := k.GetAverageBlockTime(ctx)
		if err == nil && avgBlockTime > 0 {
			blocksRemaining := halvingInfo.NextHalvingHeight - currentHeight
			estimatedDuration := avgBlockTime * time.Duration(blocksRemaining)
			estimatedDate := currentTime.Add(estimatedDuration)
			// TODO: Uncomment after proto generation
			// halvingInfo.EstimatedNextHalvingDate = timestamppb.New(estimatedDate)
			
			ctx.Logger().Info("halving date updated",
				"next_halving_height", halvingInfo.NextHalvingHeight,
				"estimated_date", estimatedDate,
				"average_block_time", avgBlockTime)
			
			// Store updated halving info with new estimated date
			bz = k.cdc.MustMarshal(&halvingInfo)
			store.Set(halvingKey, bz)
		}
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
		// Use HalvingInterval constant for consistency
		return types.HalvingInfo{
			LastHalvingHeight: 0,
			NextHalvingHeight: HalvingInterval,
			HalvingInterval:   HalvingInterval,
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
	
	// Record block time for adaptive halving calculation
	if err := k.RecordBlockTime(ctx, currentHeight); err != nil {
		ctx.Logger().Error("failed to record block time", "error", err)
	}
	
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

	// Process blind auction for current block
	// Transition from commit to reveal phase if needed
	auction, err := k.GetBlindAuction(ctx, currentHeight)
	if err == nil && auction != nil {
		// If auction is in commit phase and we have commits, transition to reveal
		if auction.Phase == consensusv1.AuctionPhase_AUCTION_PHASE_COMMIT && len(auction.Commits) > 0 {
			err = k.TransitionAuctionPhase(ctx, currentHeight)
			if err != nil {
				ctx.Logger().Error("failed to transition auction phase", "error", err)
			}
		}

		// If auction is in reveal phase and we have reveals, select winner
		if auction.Phase == consensusv1.AuctionPhase_AUCTION_PHASE_REVEAL && len(auction.Reveals) > 0 {
			winner, winningBid, err := k.SelectAuctionWinner(ctx, currentHeight)
			if err != nil {
				ctx.Logger().Error("failed to select auction winner", "error", err)
			} else {
				ctx.Logger().Info("blind auction winner selected", "height", currentHeight, "winner", winner, "bid", winningBid)
			}
		}
	}

	// Create blind auction for next block
	nextHeight := currentHeight + 1
	_, err = k.CreateBlindAuction(ctx, nextHeight)
	if err != nil {
		ctx.Logger().Error("failed to create blind auction for next block", "error", err, "height", nextHeight)
	}

	// Clean up old completed auctions (keep only last N blocks for history)
	// This prevents storage bloat from accumulating old auction data
	if err := k.CleanupOldAuctions(ctx, currentHeight); err != nil {
		ctx.Logger().Error("failed to cleanup old auctions", "error", err)
		// Don't fail EndBlocker if cleanup fails
	}

	// Distribute base rewards to validators (Circuit 1)
	// This happens after block creation, distributing passive income based on activated LZN
	err = k.DistributeBaseRewards(ctx, currentHeight)
	if err != nil {
		ctx.Logger().Error("failed to distribute base rewards", "error", err, "height", currentHeight)
		// Don't fail the block if reward distribution fails
	}

	// Process halving (adaptive halving based on block time)
	err = k.ProcessHalving(ctx)
	if err != nil {
		ctx.Logger().Error("failed to process halving", "error", err, "height", currentHeight)
		// Don't fail the block if halving processing fails
	}

	return nil
}

// InitGenesis initializes genesis state
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, *genState.Params)
	}

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

// ============================================================================
// Blind Auction Methods
// ============================================================================

// HashCommit creates a commit hash from nonce and bid amount
// commit_hash = SHA256(nonce + bid_amount)
func HashCommit(nonce, bidAmount string) string {
	data := fmt.Sprintf("%s:%s", nonce, bidAmount)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// VerifyCommit verifies that a reveal matches the commit hash
func VerifyCommit(commitHash, nonce, bidAmount string) bool {
	calculatedHash := HashCommit(nonce, bidAmount)
	return calculatedHash == commitHash
}

// GetBlindAuction returns a blind auction for a specific block height
func (k Keeper) GetBlindAuction(ctx sdk.Context, height uint64) (*consensusv1.BlindAuction, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetBlindAuctionKey(height)
	bz := store.Get(key)
	if bz == nil {
		return nil, types.ErrAuctionNotFound
	}

	var auction consensusv1.BlindAuction
	err := k.cdc.Unmarshal(bz, &auction)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal blind auction: %w", err)
	}

	return &auction, nil
}

// SetBlindAuction stores a blind auction
func (k Keeper) SetBlindAuction(ctx sdk.Context, auction *consensusv1.BlindAuction) error {
	store := ctx.KVStore(k.storeKey)
	key := types.GetBlindAuctionKey(auction.BlockHeight)

	bz, err := k.cdc.Marshal(auction)
	if err != nil {
		return fmt.Errorf("failed to marshal blind auction: %w", err)
	}

	store.Set(key, bz)
	return nil
}

// CreateBlindAuction creates a new blind auction for a block height
func (k Keeper) CreateBlindAuction(ctx sdk.Context, height uint64) (*consensusv1.BlindAuction, error) {
	// Check if auction already exists
	existing, err := k.GetBlindAuction(ctx, height)
	if err == nil && existing != nil {
		return existing, nil
	}

	auction := &consensusv1.BlindAuction{
		BlockHeight: height,
		Phase:       consensusv1.AuctionPhase_AUCTION_PHASE_COMMIT,
		Commits:     []*consensusv1.EncryptedBid{},
		Reveals:     []*consensusv1.BidReveal{},
		Winner:      "",
		WinningBid:   "0",
		StartTime:   timestamppb.Now(),
		EndTime:     nil,
	}

	err = k.SetBlindAuction(ctx, auction)
	if err != nil {
		return nil, err
	}

	return auction, nil
}

// CommitBid adds a committed bid to the auction
func (k Keeper) CommitBid(ctx sdk.Context, validator, commitHash string, height uint64) error {
	// Get or create auction
	auction, err := k.GetBlindAuction(ctx, height)
	if err != nil {
		// Create new auction if it doesn't exist
		auction, err = k.CreateBlindAuction(ctx, height)
		if err != nil {
			return err
		}
	}

	// Check if auction is in commit phase
	if auction.Phase != consensusv1.AuctionPhase_AUCTION_PHASE_COMMIT {
		return types.ErrAuctionNotInCommitPhase
	}

	// Check if validator already committed
	for _, commit := range auction.Commits {
		if commit.Validator == validator {
			return fmt.Errorf("validator %s already committed a bid", validator)
		}
	}

	// Validate commit hash
	if commitHash == "" || len(commitHash) != 64 { // SHA256 produces 64 hex chars
		return types.ErrInvalidCommitHash
	}

	// Add commit
	encryptedBid := &consensusv1.EncryptedBid{
		Validator:    validator,
		CommitHash:   commitHash,
		BlockHeight:  height,
		CommitTime:   timestamppb.Now(),
	}

	auction.Commits = append(auction.Commits, encryptedBid)

	if err := k.SetBlindAuction(ctx, auction); err != nil {
		return err
	}

	// Emit event for bid commit
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBidCommitted,
			sdk.NewAttribute(types.AttributeKeyValidator, validator),
			sdk.NewAttribute(types.AttributeKeyCommitHash, commitHash),
			sdk.NewAttribute(types.AttributeKeyAuctionHeight, fmt.Sprintf("%d", height)),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	return nil
}

// ValidateAuctionBid validates a bid to prevent manipulation
// According to whitepaper: "слепой аукцион с взвешенной лотереей"
func (k Keeper) ValidateAuctionBid(ctx sdk.Context, validator, bidAmount string) error {
	// 1. Validate bid amount is positive
	bidUint, err := strconv.ParseUint(bidAmount, 10, 64)
	if err != nil || bidUint == 0 {
		return fmt.Errorf("invalid bid amount: %s", bidAmount)
	}

	// 2. Check if validator has sufficient ANT balance
	if k.anteilKeeper != nil {
		positionInterface, err := k.anteilKeeper.GetUserPosition(ctx, validator)
		if err != nil {
			return fmt.Errorf("failed to get user position: %w", err)
		}
		
		balance, err := extractAntBalance(positionInterface)
		if err != nil {
			return fmt.Errorf("failed to extract ANT balance: %w", err)
		}
		
		balanceUint, err := strconv.ParseUint(balance, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid balance format: %w", err)
		}
		
		if balanceUint < bidUint {
			return fmt.Errorf("insufficient ANT balance: have %s, need %s", balance, bidAmount)
		}
	}

	// 3. Check for bid manipulation (prevent extremely large bids)
	// Use MaxBurnAmount from params
	params := k.GetParams(ctx)
	maxBurnAmountStr := params.MaxBurnAmount
	if maxBurnAmountStr != "" {
		// Parse MaxBurnAmount (remove "uvx" suffix if present)
		maxBurnAmountStr = strings.TrimSuffix(maxBurnAmountStr, "uvx")
		maxBurnAmountStr = strings.TrimSpace(maxBurnAmountStr)
		maxBid, err := strconv.ParseUint(maxBurnAmountStr, 10, 64)
		if err == nil && bidUint > maxBid {
			return fmt.Errorf("bid amount exceeds maximum: %d", maxBid)
		}
	}

	// 4. Check for rapid bid changes (potential manipulation)
	// Store bid history to detect suspicious patterns
	bidHistoryKey := types.GetBidHistoryKey(validator)
	store := ctx.KVStore(k.storeKey)
	
	// Get recent bid history
	bz := store.Get(bidHistoryKey)
	if bz != nil {
		var history []map[string]interface{}
		if err := json.Unmarshal(bz, &history); err == nil {
			// Check if there are too many rapid bid changes
			recentBids := 0
			currentTime := ctx.BlockTime().Unix()
			for _, entry := range history {
				if timestamp, ok := entry["timestamp"].(float64); ok {
					// Count bids in last 10 blocks (approximately)
					if currentTime-int64(timestamp) < 100 {
						recentBids++
					}
				}
			}
			
			// Prevent too many rapid bids (configurable limit)
			params := k.GetParams(ctx)
			rapidBidLimit := params.RapidBidLimit
			if rapidBidLimit == 0 {
				rapidBidLimit = 5 // Default fallback
			}
			if recentBids >= int(rapidBidLimit) {
				return fmt.Errorf("too many rapid bid changes detected - potential manipulation")
			}
		}
	}

	return nil
}

// RecordBidHistory records a bid in the validator's bid history
func (k Keeper) RecordBidHistory(ctx sdk.Context, validator, bidAmount string) {
	store := ctx.KVStore(k.storeKey)
	bidHistoryKey := types.GetBidHistoryKey(validator)
	
	// Get existing history
	var history []map[string]interface{}
	bz := store.Get(bidHistoryKey)
	if bz != nil {
		json.Unmarshal(bz, &history)
	}
	
	// Add new bid entry
	entry := map[string]interface{}{
		"bid_amount":  bidAmount,
		"timestamp":   ctx.BlockTime().Unix(),
		"block_height": ctx.BlockHeight(),
	}
	
	history = append(history, entry)
	
	// Keep only last N entries (configurable limit)
	params := k.GetParams(ctx)
	bidHistoryLimit := int(params.BidHistoryLimit)
	if bidHistoryLimit == 0 {
		bidHistoryLimit = 100 // Default fallback
	}
	if len(history) > bidHistoryLimit {
		history = history[len(history)-bidHistoryLimit:]
	}
	
	// Store updated history
	historyBz, err := json.Marshal(history)
	if err == nil {
		store.Set(bidHistoryKey, historyBz)
	}
}

// RevealBid reveals a bid in the auction
func (k Keeper) RevealBid(ctx sdk.Context, validator, nonce, bidAmount string, height uint64) error {
	// Get auction
	auction, err := k.GetBlindAuction(ctx, height)
	if err != nil {
		return err
	}

	// Check if auction is in reveal phase
	if auction.Phase != consensusv1.AuctionPhase_AUCTION_PHASE_REVEAL {
		return types.ErrAuctionNotInRevealPhase
	}

	// Find the commit for this validator
	var commit *consensusv1.EncryptedBid
	for _, c := range auction.Commits {
		if c.Validator == validator {
			commit = c
			break
		}
	}

	if commit == nil {
		return types.ErrBidNotCommitted
	}

	// Check if already revealed
	for _, reveal := range auction.Reveals {
		if reveal.Validator == validator {
			return types.ErrBidAlreadyRevealed
		}
	}

	// Verify commit hash matches reveal
	if !VerifyCommit(commit.CommitHash, nonce, bidAmount) {
		return types.ErrCommitHashMismatch
	}

	// Validate bid amount and check for manipulation
	if err := k.ValidateAuctionBid(ctx, validator, bidAmount); err != nil {
		return fmt.Errorf("bid validation failed: %w", err)
	}
	
	bidAmountInt, err := strconv.ParseUint(bidAmount, 10, 64)
	if err != nil || bidAmountInt == 0 {
		return types.ErrInvalidBidAmount
	}
	
	// Record bid in history for manipulation detection
	k.RecordBidHistory(ctx, validator, bidAmount)

	// Emit event for bid reveal
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBidRevealed,
			sdk.NewAttribute(types.AttributeKeyValidator, validator),
			sdk.NewAttribute(types.AttributeKeyBidAmount, bidAmount),
			sdk.NewAttribute(types.AttributeKeyNonce, nonce),
			sdk.NewAttribute(types.AttributeKeyAuctionHeight, fmt.Sprintf("%d", height)),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	// Check ANT balance if anteil keeper is available
	// According to whitepaper: validators must have sufficient ANT to bid
	if k.anteilKeeper != nil {
		positionInterface, err := k.anteilKeeper.GetUserPosition(ctx, validator)
		if err != nil {
			// If position doesn't exist, validator has no ANT balance
			return fmt.Errorf("validator %s has no ANT balance", validator)
		}

		antBalanceStr, err := extractAntBalance(positionInterface)
		if err != nil {
			return fmt.Errorf("failed to extract ANT balance: %w", err)
		}

		antBalance, err := strconv.ParseUint(antBalanceStr, 10, 64)
		if err != nil {
			antBalance = 0
		}

		// Check if validator has sufficient ANT balance for the bid
		if antBalance < bidAmountInt {
			return fmt.Errorf("insufficient ANT balance: have %d, need %d", antBalance, bidAmountInt)
		}
	}

	// Add reveal
	bidReveal := &consensusv1.BidReveal{
		Validator:    validator,
		Nonce:       nonce,
		BidAmount:   bidAmount,
		BlockHeight: height,
		RevealTime:  timestamppb.Now(),
	}

	auction.Reveals = append(auction.Reveals, bidReveal)

	return k.SetBlindAuction(ctx, auction)
}

// SelectAuctionWinner selects the winner of the blind auction using weighted lottery
// According to whitepaper: "Шанс на выигрыш пропорционален размеру ставки"
func (k Keeper) SelectAuctionWinner(ctx sdk.Context, height uint64) (string, string, error) {
	// Get auction
	auction, err := k.GetBlindAuction(ctx, height)
	if err != nil {
		return "", "", err
	}

	// Check if auction is in reveal phase
	if auction.Phase != consensusv1.AuctionPhase_AUCTION_PHASE_REVEAL {
		return "", "", fmt.Errorf("auction is not in reveal phase")
	}

	// Check if there are any reveals
	if len(auction.Reveals) == 0 {
		return "", "", fmt.Errorf("no bids revealed")
	}

	// Calculate total bid amount (sum of all revealed bids)
	totalBid := uint64(0)
	bidAmounts := make([]uint64, len(auction.Reveals))
	for i, reveal := range auction.Reveals {
		bidAmount, err := strconv.ParseUint(reveal.BidAmount, 10, 64)
		if err != nil {
			continue // Skip invalid bids
		}
		bidAmounts[i] = bidAmount
		totalBid += bidAmount
	}

	if totalBid == 0 {
		return "", "", fmt.Errorf("total bid amount is zero")
	}

	// Weighted random selection: chance proportional to bid amount
	randomWeight := rand.Uint64() % totalBid
	currentWeight := uint64(0)

	for i, bidAmount := range bidAmounts {
		currentWeight += bidAmount
		if randomWeight < currentWeight {
			winner := auction.Reveals[i]
			winnerValidator := winner.Validator
			winningBid := winner.BidAmount

			// Burn ANT from winner (according to whitepaper: "Только победитель аукциона фактически покупает права на ANT")
			if k.anteilKeeper != nil {
				positionInterface, err := k.anteilKeeper.GetUserPosition(ctx, winnerValidator)
				if err == nil {
					// Get current ANT balance
					currentBalanceStr, err := extractAntBalance(positionInterface)
					if err == nil {
						currentBalance, err := strconv.ParseUint(currentBalanceStr, 10, 64)
						if err == nil {
							winningBidInt, err := strconv.ParseUint(winningBid, 10, 64)
							if err == nil && currentBalance >= winningBidInt {
								// Burn ANT: subtract winning bid amount from balance
								newBalance := currentBalance - winningBidInt
								err = k.anteilKeeper.UpdateUserPosition(ctx, winnerValidator, fmt.Sprintf("%d", newBalance), 0)
								if err != nil {
									ctx.Logger().Error("failed to burn ANT from winner", "error", err, "winner", winnerValidator, "amount", winningBid)
								} else {
									ctx.Logger().Info("ANT burned from auction winner", "winner", winnerValidator, "amount", winningBid, "new_balance", newBalance)
									
									// Emit burn event for tracking
									ctx.EventManager().EmitEvent(
										sdk.NewEvent(
											types.EventTypeBurnExecuted,
											sdk.NewAttribute(types.AttributeKeyValidator, winnerValidator),
											sdk.NewAttribute(types.AttributeKeyBurnAmount, winningBid),
											sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", height)),
											sdk.NewAttribute(types.AttributeKeyNewBalance, fmt.Sprintf("%d", newBalance)),
											sdk.NewAttribute(types.AttributeKeyAuctionWinner, "true"),
										),
									)
								}
							}
						}
					}
				}
			}

			// Return ANT to other validators (those who revealed but didn't win)
			// According to whitepaper: only winner actually purchases ANT rights
			if k.anteilKeeper != nil {
				for j, reveal := range auction.Reveals {
					if j != i && reveal.Validator != winnerValidator {
						// Return ANT to non-winner validators (they already committed/revealed, but didn't win)
						// Note: In a blind auction, validators don't actually spend ANT until they win
						// This is just for accounting - the actual ANT is only burned from the winner
						ctx.Logger().Info("ANT returned to non-winner validator", "validator", reveal.Validator, "bid", reveal.BidAmount)
					}
				}
			}

			auction.Winner = winnerValidator
			auction.WinningBid = winningBid
			auction.Phase = consensusv1.AuctionPhase_AUCTION_PHASE_COMPLETE
			auction.EndTime = timestamppb.Now()

			err = k.SetBlindAuction(ctx, auction)
			if err != nil {
				return "", "", err
			}

			return winnerValidator, winningBid, nil
		}
	}

	// Fallback to first reveal (should not happen)
	winner := auction.Reveals[0]
	winnerValidator := winner.Validator
	winningBid := winner.BidAmount

	// Burn ANT from winner (fallback case)
	if k.anteilKeeper != nil {
		positionInterface, err := k.anteilKeeper.GetUserPosition(ctx, winnerValidator)
		if err == nil {
			currentBalanceStr, err := extractAntBalance(positionInterface)
			if err == nil {
				currentBalance, err := strconv.ParseUint(currentBalanceStr, 10, 64)
				if err == nil {
					winningBidInt, err := strconv.ParseUint(winningBid, 10, 64)
					if err == nil && currentBalance >= winningBidInt {
						newBalance := currentBalance - winningBidInt
						err = k.anteilKeeper.UpdateUserPosition(ctx, winnerValidator, fmt.Sprintf("%d", newBalance), 0)
						if err != nil {
							ctx.Logger().Error("failed to burn ANT from winner (fallback)", "error", err, "winner", winnerValidator, "amount", winningBid)
						} else {
							ctx.Logger().Info("ANT burned from auction winner (fallback)", "winner", winnerValidator, "amount", winningBid, "new_balance", newBalance)
							
							// Emit burn event for tracking
							ctx.EventManager().EmitEvent(
								sdk.NewEvent(
									types.EventTypeBurnExecuted,
									sdk.NewAttribute(types.AttributeKeyValidator, winnerValidator),
									sdk.NewAttribute(types.AttributeKeyBurnAmount, winningBid),
									sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", height)),
									sdk.NewAttribute(types.AttributeKeyNewBalance, fmt.Sprintf("%d", newBalance)),
									sdk.NewAttribute(types.AttributeKeyAuctionWinner, "true"),
									sdk.NewAttribute("fallback", "true"),
								),
							)
						}
					}
				}
			}
		}
	}

	// Return ANT to other validators (fallback case)
	if k.anteilKeeper != nil && len(auction.Reveals) > 1 {
		for _, reveal := range auction.Reveals {
			if reveal.Validator != winnerValidator {
				ctx.Logger().Info("ANT returned to non-winner validator (fallback)", "validator", reveal.Validator, "bid", reveal.BidAmount)
			}
		}
	}

	auction.Winner = winnerValidator
	auction.WinningBid = winningBid
	auction.Phase = consensusv1.AuctionPhase_AUCTION_PHASE_COMPLETE
	auction.EndTime = timestamppb.Now()

	err = k.SetBlindAuction(ctx, auction)
	if err != nil {
		return "", "", err
	}

	return winnerValidator, winningBid, nil
}

// TransitionAuctionPhase transitions auction from commit to reveal phase
func (k Keeper) TransitionAuctionPhase(ctx sdk.Context, height uint64) error {
	auction, err := k.GetBlindAuction(ctx, height)
	if err != nil {
		return err
	}

	if auction.Phase == consensusv1.AuctionPhase_AUCTION_PHASE_COMMIT {
		auction.Phase = consensusv1.AuctionPhase_AUCTION_PHASE_REVEAL
		return k.SetBlindAuction(ctx, auction)
	}

	return nil
}

// CleanupOldAuctions removes old completed auctions to prevent storage bloat
// Keeps only the last N completed auctions for history (configurable)
// According to whitepaper: auctions are per-block, so old ones can be safely removed
func (k Keeper) CleanupOldAuctions(ctx sdk.Context, currentHeight uint64) error {
	// Keep last N completed auctions for history (from params)
	params := k.GetParams(ctx)
	keepHistoryBlocks := params.AuctionHistoryBlocks
	if keepHistoryBlocks == 0 {
		keepHistoryBlocks = 100 // Default fallback
	}
	
	// Calculate the oldest height we want to keep
	// We keep auctions from (currentHeight - keepHistoryBlocks) to currentHeight
	if currentHeight <= keepHistoryBlocks {
		// Not enough blocks to clean up yet
		return nil
	}
	
	oldestHeightToKeep := currentHeight - keepHistoryBlocks
	
	// Delete auctions older than oldestHeightToKeep
	store := ctx.KVStore(k.storeKey)
	deletedCount := 0
	
	// Iterate through potential auction keys
	// We check heights from 1 to (oldestHeightToKeep - 1)
	for height := uint64(1); height < oldestHeightToKeep; height++ {
		key := types.GetBlindAuctionKey(height)
		if store.Has(key) {
			// Only delete completed auctions (to avoid deleting active ones)
			auction, err := k.GetBlindAuction(ctx, height)
			if err == nil && auction != nil {
				// Only delete if auction is complete (winner selected)
				if auction.Phase == consensusv1.AuctionPhase_AUCTION_PHASE_COMPLETE {
					store.Delete(key)
					deletedCount++
				}
			}
		}
	}
	
	if deletedCount > 0 {
		ctx.Logger().Info("cleaned up old auctions", "deleted_count", deletedCount, "oldest_kept_height", oldestHeightToKeep)
	}
	
	return nil
}

// ============================================================================
// Base Reward Distribution (Circuit 1)
// ============================================================================

const (
	// HalvingInterval is the number of blocks between halvings (210,000 blocks)
	HalvingInterval = 210_000
)

// CalculateBaseReward calculates the base block reward considering halving
// Formula: base_reward = BASE_BLOCK_REWARD / (2^halving_count)
// where halving_count = floor(block_height / HALVING_INTERVAL)
func (k Keeper) CalculateBaseReward(ctx sdk.Context, height uint64) (uint64, error) {
	params := k.GetParams(ctx)
	
	// Parse base block reward from params (e.g., "50000000uwrt")
	// Extract numeric part before "uwrt" or parse as coin
	baseRewardStr := params.BaseBlockReward
	if baseRewardStr == "" {
		// Fallback to default if not set
		baseRewardStr = "50000000uwrt"
	}
	
	// Parse the amount (remove "uwrt" suffix if present)
	baseRewardStr = strings.TrimSuffix(baseRewardStr, "uwrt")
	baseRewardStr = strings.TrimSpace(baseRewardStr)
	
	baseReward, err := strconv.ParseUint(baseRewardStr, 10, 64)
	if err != nil {
		// Fallback to default if parsing fails
		baseReward = 50_000_000 // 50 WRT in micro units
	}
	
	// Calculate halving count
	halvingCount := height / HalvingInterval

	// Calculate reward: base_reward / (2^halving_count)
	reward := baseReward
	if halvingCount > 0 {
		// Right shift by halvingCount is equivalent to dividing by 2^halvingCount
		// But we need to be careful with large halvingCount values
		for i := uint64(0); i < halvingCount && reward > 0; i++ {
			reward = reward / 2
		}
	}

	// Minimum reward is 1 micro WRT (to avoid zero rewards)
	if reward == 0 {
		reward = 1
	}

	return reward, nil
}

// GetHalvingCount returns the current halving count for a given block height
func (k Keeper) GetHalvingCount(height uint64) uint64 {
	return height / HalvingInterval
}

// extractLizenzInfo extracts validator and amount from an ActivatedLizenz object
// Uses reflection to avoid circular dependency with lizenzv1 package
func extractLizenzInfo(lizenzInterface interface{}) (validator string, amount string, err error) {
	// Use reflection to get the fields
	v := reflect.ValueOf(lizenzInterface)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Get Validator field
	validatorField := v.FieldByName("Validator")
	if !validatorField.IsValid() || validatorField.Kind() != reflect.String {
		return "", "", fmt.Errorf("invalid LZN object: Validator field not found or invalid")
	}
	validator = validatorField.String()

	// Get Amount field
	amountField := v.FieldByName("Amount")
	if !amountField.IsValid() || amountField.Kind() != reflect.String {
		return "", "", fmt.Errorf("invalid LZN object: Amount field not found or invalid")
	}
	amount = amountField.String()

	return validator, amount, nil
}

// extractAntBalance extracts ANT balance from a UserPosition object
// Uses reflection to avoid circular dependency with anteilv1 package
func extractAntBalance(positionInterface interface{}) (string, error) {
	// Use reflection to get the fields
	v := reflect.ValueOf(positionInterface)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Get AntBalance field
	antBalanceField := v.FieldByName("AntBalance")
	if !antBalanceField.IsValid() || antBalanceField.Kind() != reflect.String {
		return "0", fmt.Errorf("invalid UserPosition object: AntBalance field not found or invalid")
	}

	return antBalanceField.String(), nil
}

// CalculateMOAPenaltyMultiplier calculates the penalty multiplier based on MOA compliance
// According to whitepaper and economic-formulas.md:
// - >= threshold_high: no penalty (1.0)
// - >= threshold_warning: warning (1.0, but logged)
// - >= threshold_medium: 25% penalty (0.75)
// - >= threshold_low: 50% penalty (0.5)
// - < threshold_low: deactivation (0.0)
func (k Keeper) CalculateMOAPenaltyMultiplier(ctx sdk.Context, moaCompliance float64) float64 {
	params := k.GetParams(ctx)
	
	// Parse thresholds from params with fallback to defaults
	thresholdHigh, _ := strconv.ParseFloat(params.MoaPenaltyThresholdHigh, 64)
	if thresholdHigh == 0 {
		thresholdHigh = 1.0
	}
	thresholdWarning, _ := strconv.ParseFloat(params.MoaPenaltyThresholdWarning, 64)
	if thresholdWarning == 0 {
		thresholdWarning = 0.9
	}
	thresholdMedium, _ := strconv.ParseFloat(params.MoaPenaltyThresholdMedium, 64)
	if thresholdMedium == 0 {
		thresholdMedium = 0.7
	}
	thresholdLow, _ := strconv.ParseFloat(params.MoaPenaltyThresholdLow, 64)
	if thresholdLow == 0 {
		thresholdLow = 0.5
	}
	
	if moaCompliance >= thresholdHigh {
		return 1.0 // No penalty
	} else if moaCompliance >= thresholdWarning {
		return 1.0 // Warning, but no penalty
	} else if moaCompliance >= thresholdMedium {
		return 0.75 // 25% penalty
	} else if moaCompliance >= thresholdLow {
		return 0.5 // 50% penalty
	} else {
		return 0.0 // Deactivation, no reward
	}
}

// ValidatorRewardInfo contains information about a validator's reward
type ValidatorRewardInfo struct {
	Validator        string
	ActivatedLZN     uint64  // Amount of activated LZN in micro units
	RewardShare      float64 // Share of total reward (0.0 to 1.0)
	BaseRewardAmount uint64  // Base calculated reward amount in micro WRT (before MOA penalty)
	MOACompliance    float64 // MOA compliance ratio (0.0 to 1.0+)
	PenaltyMultiplier float64 // Penalty multiplier based on MOA (0.0 to 1.0)
	FinalRewardAmount uint64  // Final reward amount after MOA penalty in micro WRT
}

// CalculateRewardDistribution calculates how base reward should be distributed among validators
// According to whitepaper: "validator_passive_income = (activated_lzn_validator / total_activated_lzn) × current_block_reward"
// Returns list of validator rewards and total distributed amount
func (k Keeper) CalculateRewardDistribution(ctx sdk.Context, baseReward uint64, validatorLZN map[string]uint64) ([]ValidatorRewardInfo, uint64, error) {
	if len(validatorLZN) == 0 {
		return []ValidatorRewardInfo{}, 0, fmt.Errorf("no validators with activated LZN")
	}

	// Calculate total activated LZN
	totalLZN := uint64(0)
	for _, lzn := range validatorLZN {
		totalLZN += lzn
	}

	if totalLZN == 0 {
		return []ValidatorRewardInfo{}, 0, fmt.Errorf("total activated LZN is zero")
	}

	var rewards []ValidatorRewardInfo
	totalDistributed := uint64(0)

	// Calculate reward for each validator
	for validator, activatedLZN := range validatorLZN {
		if activatedLZN == 0 {
			continue // Skip validators with no activated LZN
		}

		// Calculate share: activated_lzn / total_activated_lzn
		share := float64(activatedLZN) / float64(totalLZN)

		// Calculate base reward: share × base_reward
		// Use integer arithmetic to avoid floating point precision issues
		baseRewardAmount := (baseReward * activatedLZN) / totalLZN

		// Get MOA compliance (default to 1.0 if not available)
		moaCompliance := 1.0
		if k.lizenzKeeper != nil {
			compliance, err := k.lizenzKeeper.GetMOACompliance(ctx, validator)
			if err == nil {
				moaCompliance = compliance
			}
		}

		// Calculate penalty multiplier
		penaltyMultiplier := k.CalculateMOAPenaltyMultiplier(ctx, moaCompliance)

		// Calculate final reward after penalty
		finalRewardAmount := uint64(float64(baseRewardAmount) * penaltyMultiplier)

		rewards = append(rewards, ValidatorRewardInfo{
			Validator:         validator,
			ActivatedLZN:      activatedLZN,
			RewardShare:       share,
			BaseRewardAmount:  baseRewardAmount,
			MOACompliance:     moaCompliance,
			PenaltyMultiplier: penaltyMultiplier,
			FinalRewardAmount: finalRewardAmount,
		})

		totalDistributed += finalRewardAmount
	}

	// Handle rounding: distribute any remainder to the first compliant validator
	// (only if they have full compliance)
	if totalDistributed < baseReward {
		remainder := baseReward - totalDistributed
		for i := range rewards {
			if rewards[i].MOACompliance >= 1.0 {
				rewards[i].FinalRewardAmount += remainder
				totalDistributed += remainder
				break
			}
		}
	}

	return rewards, totalDistributed, nil
}

// DistributeBaseRewards distributes base block rewards to validators based on their activated LZN
// This implements Circuit 1 of the economic model: passive income for validators
// According to whitepaper: "validator_passive_income = (activated_lzn_validator / total_activated_lzn) × current_block_reward"
func (k Keeper) DistributeBaseRewards(ctx sdk.Context, height uint64) error {
	// Calculate base reward for this block
	baseReward, err := k.CalculateBaseReward(ctx, height)
	if err != nil {
		return fmt.Errorf("failed to calculate base reward: %w", err)
	}

	// If no lizenz keeper is set, we can't distribute rewards
	// This is expected in some test scenarios
	if k.lizenzKeeper == nil {
		ctx.Logger().Info("lizenz keeper not set, skipping reward distribution", "height", height, "base_reward", baseReward)
		return nil
	}

	// Get all activated LZN
	allLizenzs, err := k.lizenzKeeper.GetAllActivatedLizenz(ctx)
	if err != nil {
		return fmt.Errorf("failed to get activated LZN: %w", err)
	}

	// Build map of validator -> activated LZN amount
	validatorLZN := make(map[string]uint64)
	for _, lizenzInterface := range allLizenzs {
		// Type assertion: we expect *lizenzv1.ActivatedLizenz
		// Use reflection or type assertion to get validator and amount
		// Since we can't import lizenzv1 here, we use a helper function
		validator, amount, err := extractLizenzInfo(lizenzInterface)
		if err != nil {
			ctx.Logger().Error("failed to extract LZN info", "error", err)
			continue
		}

		// Parse amount to uint64
		amountInt, err := strconv.ParseUint(amount, 10, 64)
		if err != nil {
			ctx.Logger().Error("failed to parse LZN amount", "error", err, "amount", amount)
			continue
		}

		validatorLZN[validator] = amountInt
	}

	// If no validators have activated LZN, skip distribution
	if len(validatorLZN) == 0 {
		ctx.Logger().Info("no validators with activated LZN, skipping reward distribution", "height", height)
		return nil
	}

	// Calculate reward distribution
	rewards, totalDistributed, err := k.CalculateRewardDistribution(ctx, baseReward, validatorLZN)
	if err != nil {
		return fmt.Errorf("failed to calculate reward distribution: %w", err)
	}

	// Log distribution with MOA compliance information
	ctx.Logger().Info("base reward distribution calculated",
		"height", height,
		"base_reward", baseReward,
		"total_distributed", totalDistributed,
		"validators_count", len(rewards))

	// Update reward statistics for each validator
	for _, reward := range rewards {
		ctx.Logger().Info("validator reward",
			"validator", reward.Validator,
			"activated_lzn", reward.ActivatedLZN,
			"share", reward.RewardShare,
			"base_reward", reward.BaseRewardAmount,
			"moa_compliance", reward.MOACompliance,
			"penalty_multiplier", reward.PenaltyMultiplier,
			"final_reward", reward.FinalRewardAmount)
		
		// Update reward stats in lizenz module
		if k.lizenzKeeper != nil {
			if err := k.lizenzKeeper.UpdateRewardStats(ctx, reward.Validator, reward.FinalRewardAmount, height, reward.MOACompliance, reward.PenaltyMultiplier, reward.BaseRewardAmount); err != nil {
				ctx.Logger().Error("failed to update reward stats", "error", err, "validator", reward.Validator)
				// Don't fail reward distribution if stats update fails
			}
		}
	}

	// Send WRT tokens to validators via bank module
	if k.bankKeeper != nil {
		for _, reward := range rewards {
			validatorAddr, err := sdk.AccAddressFromBech32(reward.Validator)
			if err != nil {
				ctx.Logger().Error("invalid validator address", "error", err, "validator", reward.Validator)
				continue
			}

			// Create coins for reward amount (in micro WRT)
			rewardCoins := sdk.NewCoins(sdk.NewCoin("uwrt", math.NewIntFromUint64(reward.FinalRewardAmount)))

			// Mint coins from consensus module account and send to validator
			// Note: In production, coins should be minted from a module account
			// For now, we use SendCoinsFromModuleToAccount which requires module account setup
			moduleName := types.ModuleName
			if err := k.bankKeeper.MintCoins(ctx, moduleName, rewardCoins); err != nil {
				ctx.Logger().Error("failed to mint reward coins", "error", err, "validator", reward.Validator, "amount", reward.FinalRewardAmount)
				continue
			}

			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, moduleName, validatorAddr, rewardCoins); err != nil {
				ctx.Logger().Error("failed to send reward coins", "error", err, "validator", reward.Validator, "amount", reward.FinalRewardAmount)
				// Note: Minted coins remain in module account if sending fails
				// In production, should implement proper error recovery (burn or retry)
				continue
			}

			ctx.Logger().Info("WRT reward sent to validator",
				"validator", reward.Validator,
				"amount", reward.FinalRewardAmount,
				"height", height)
			
			// Emit reward distribution event
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRewardDistributed,
					sdk.NewAttribute(types.AttributeKeyValidator, reward.Validator),
					sdk.NewAttribute(types.AttributeKeyRewardAmount, fmt.Sprintf("%d", reward.FinalRewardAmount)),
					sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", height)),
					sdk.NewAttribute(types.AttributeKeyRewardShare, fmt.Sprintf("%.6f", reward.RewardShare)),
					sdk.NewAttribute(types.AttributeKeyMOACompliance, fmt.Sprintf("%.2f", reward.MOACompliance)),
					sdk.NewAttribute(types.AttributeKeyPenaltyMultiplier, fmt.Sprintf("%.2f", reward.PenaltyMultiplier)),
				),
			)
		}
	} else {
		ctx.Logger().Info("bank keeper not set, rewards calculated but not sent", "height", height)
	}

	return nil
}
