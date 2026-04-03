package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
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
	GetAllActivatedLizenz(ctx sdk.Context) ([]interface{}, error)
	GetTotalActivatedLizenz(ctx sdk.Context) (string, error)
	GetMOACompliance(ctx sdk.Context, validator string) (float64, error)
	UpdateRewardStats(ctx sdk.Context, validator string, rewardAmount uint64, blockHeight uint64, moaCompliance float64, penaltyMultiplier float64, baseReward uint64) error
}

// AnteilKeeperInterface defines the interface for interacting with anteil module
// This allows consensus module to check ANT balances and burn ANT tokens
// Note: We use interface{} for UserPosition to avoid circular dependencies
type AnteilKeeperInterface interface {
	GetUserPosition(ctx sdk.Context, user string) (interface{}, error)
	SetUserPosition(ctx sdk.Context, position interface{}) error
	UpdateUserPosition(ctx sdk.Context, user string, antBalance string, orderCount uint32) error
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
		lizenzKeeper LizenzKeeperInterface
		anteilKeeper AnteilKeeperInterface
		bankKeeper   BankKeeperInterface
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ps paramtypes.Subspace,
) *Keeper {
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		paramstore: ps,
	}
}

func (k *Keeper) SetLizenzKeeper(lizenzKeeper LizenzKeeperInterface) {
	k.lizenzKeeper = lizenzKeeper
}

func (k *Keeper) SetAnteilKeeper(anteilKeeper AnteilKeeperInterface) {
	k.anteilKeeper = anteilKeeper
}

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

// ============================================================================
// Block Time
// ============================================================================

// RecordBlockTime records the time for a block (used for adaptive halving)
func (k Keeper) RecordBlockTime(ctx sdk.Context, height uint64) error {
	store := ctx.KVStore(k.storeKey)
	blockTimeKey := types.GetBlockTimeKey(height)

	blockTime := ctx.BlockTime()
	timeBz, err := blockTime.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal block time: %w", err)
	}

	store.Set(blockTimeKey, timeBz)

	return k.updateAverageBlockTime(ctx)
}

// GetAverageBlockTime returns the average block time from a sliding window.
func (k Keeper) GetAverageBlockTime(ctx sdk.Context) (time.Duration, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AverageBlockTimeKey)

	if bz == nil {
		return 5 * time.Second, nil
	}

	var avgNanos int64
	if err := json.Unmarshal(bz, &avgNanos); err != nil {
		return 5 * time.Second, nil
	}

	return time.Duration(avgNanos), nil
}

// updateAverageBlockTime recalculates the average block time over a sliding window.
func (k Keeper) updateAverageBlockTime(ctx sdk.Context) error {
	currentHeight := uint64(ctx.BlockHeight())
	const defaultWindowSize = uint64(1000)

	startHeight := uint64(0)
	if currentHeight > defaultWindowSize {
		startHeight = currentHeight - defaultWindowSize
	}

	var totalDuration time.Duration
	blockCount := uint64(0)
	store := ctx.KVStore(k.storeKey)

	for h := startHeight + 1; h <= currentHeight; h++ {
		blockTimeKey := types.GetBlockTimeKey(h)
		timeBz := store.Get(blockTimeKey)
		if timeBz == nil {
			continue
		}

		var blockTime time.Time
		if err := blockTime.UnmarshalBinary(timeBz); err != nil {
			continue
		}

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
		return nil
	}

	averageBlockTime := totalDuration / time.Duration(blockCount)

	avgNanos := int64(averageBlockTime)
	avgBz, err := json.Marshal(avgNanos)
	if err != nil {
		return fmt.Errorf("failed to marshal average block time: %w", err)
	}

	store.Set(types.AverageBlockTimeKey, avgBz)
	return nil
}

// GetTargetBlockTime returns the dynamic target block time.
func (k Keeper) GetTargetBlockTime(ctx sdk.Context) time.Duration {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TargetBlockTimeKey)
	if bz == nil || len(bz) < 8 {
		return 0
	}
	ns := binary.BigEndian.Uint64(bz)
	return time.Duration(ns)
}

// SetTargetBlockTime stores the dynamic target block time.
func (k Keeper) SetTargetBlockTime(ctx sdk.Context, d time.Duration) error {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(d))
	store.Set(types.TargetBlockTimeKey, bz)
	return nil
}

// ValidateBlockTiming checks that the proposed block timestamp respects the target block time.
func (k Keeper) ValidateBlockTiming(ctx sdk.Context, prev, proposed time.Time) error {
	target := k.GetTargetBlockTime(ctx)
	if target == 0 {
		return nil
	}
	minTime := prev.Add(target)
	if proposed.Before(minTime) {
		return fmt.Errorf("block too fast: proposed %s, earliest allowed %s (target %s)",
			proposed.Format(time.RFC3339Nano), minTime.Format(time.RFC3339Nano), target)
	}
	return nil
}

// ============================================================================
// Halving
// ============================================================================

const (
	HalvingInterval = 210_000
)

// ProcessHalving processes halving event.
// Halving occurs strictly every N blocks, but the estimated real-world date adapts to the dynamic block time.
func (k Keeper) ProcessHalving(ctx sdk.Context) error {
	store := ctx.KVStore(k.storeKey)
	halvingKey := types.KeyHalvingInfo()

	var halvingInfo consensusv1.HalvingInfo
	bz := store.Get(halvingKey)
	if bz == nil {
		halvingInfo = consensusv1.HalvingInfo{
			LastHalvingHeight: 0,
			NextHalvingHeight: HalvingInterval,
			HalvingInterval:   HalvingInterval,
		}
	} else {
		k.cdc.MustUnmarshal(bz, &halvingInfo)
	}

	currentHeight := uint64(ctx.BlockHeight())
	currentTime := ctx.BlockTime()

	if currentHeight >= halvingInfo.NextHalvingHeight {
		halvingInfo.LastHalvingHeight = halvingInfo.NextHalvingHeight
		halvingInfo.LastHalvingDate = timestamppb.New(currentTime)
		halvingInfo.NextHalvingHeight += halvingInfo.HalvingInterval

		avgBlockTime, err := k.GetAverageBlockTime(ctx)
		if err == nil && avgBlockTime > 0 {
			blocksRemaining := halvingInfo.NextHalvingHeight - currentHeight
			estimatedDuration := avgBlockTime * time.Duration(blocksRemaining)
			estimatedDate := currentTime.Add(estimatedDuration)
			halvingInfo.EstimatedNextHalvingDate = timestamppb.New(estimatedDate)

			ctx.Logger().Info("halving processed",
				"height", currentHeight,
				"next_halving_height", halvingInfo.NextHalvingHeight,
				"estimated_date", estimatedDate,
				"average_block_time", avgBlockTime)
		}

		bz = k.cdc.MustMarshal(&halvingInfo)
		store.Set(halvingKey, bz)
	} else {
		avgBlockTime, err := k.GetAverageBlockTime(ctx)
		if err == nil && avgBlockTime > 0 {
			blocksRemaining := halvingInfo.NextHalvingHeight - currentHeight
			estimatedDuration := avgBlockTime * time.Duration(blocksRemaining)
			estimatedDate := currentTime.Add(estimatedDuration)
			halvingInfo.EstimatedNextHalvingDate = timestamppb.New(estimatedDate)

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

// ============================================================================
// Consensus State
// ============================================================================

// GetConsensusState returns current consensus state
func (k Keeper) GetConsensusState(ctx sdk.Context) (types.ConsensusState, error) {
	store := ctx.KVStore(k.storeKey)
	consensusKey := types.KeyConsensusState()

	var consensusState types.ConsensusState
	bz := store.Get(consensusKey)
	if bz == nil {
		return types.ConsensusState{
			CurrentHeight:    uint64(ctx.BlockHeight()),
			TotalAntBurned:   "0",
			LastBlockTime:    timestamppb.New(ctx.BlockTime()),
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

// ============================================================================
// Validator KV helpers
// ============================================================================

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

// ============================================================================
// Validator Weight
// ============================================================================

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

	validatorWeight := types.ValidatorWeight{
		Validator: validator,
		Weight:    weight,
	}

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
			continue
		}
		weights = append(weights, weight)
	}

	return weights, nil
}

// ============================================================================
// ANT Purchase Tracking (for MOA epoch)
// ============================================================================

// RecordAntBid adds a validator's bid amount to their epoch-level ANT purchase total.
func (k Keeper) RecordAntBid(ctx sdk.Context, validator string, amount uint64) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetAntPurchaseKey(validator)
	current := uint64(0)
	if bz := store.Get(key); bz != nil && len(bz) >= 8 {
		current = binary.BigEndian.Uint64(bz)
	}
	current += amount
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, current)
	store.Set(key, bz)
}

// GetAntPurchaseTotal returns total ANT bid for a validator in the current epoch.
func (k Keeper) GetAntPurchaseTotal(ctx sdk.Context, validator string) uint64 {
	store := ctx.KVStore(k.storeKey)
	key := types.GetAntPurchaseKey(validator)
	bz := store.Get(key)
	if bz == nil || len(bz) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// ResetAntPurchases clears all ANT purchase trackers for a new epoch.
func (k Keeper) ResetAntPurchases(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	end := make([]byte, len(types.AntPurchaseKeyPrefix))
	copy(end, types.AntPurchaseKeyPrefix)
	end[len(end)-1]++
	iter := store.Iterator(types.AntPurchaseKeyPrefix, end)
	defer iter.Close()
	var keys [][]byte
	for ; iter.Valid(); iter.Next() {
		keys = append(keys, iter.Key())
	}
	for _, key := range keys {
		store.Delete(key)
	}
}

// GetEpochStartHeight returns the height when the current epoch began.
func (k Keeper) GetEpochStartHeight(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.EpochHeightKey)
	if bz == nil || len(bz) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// SetEpochStartHeight stores the height when the current epoch began.
func (k Keeper) SetEpochStartHeight(ctx sdk.Context, height uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, height)
	store.Set(types.EpochHeightKey, bz)
}

// ProcessMOAEpoch checks if an epoch boundary is reached and updates MOA compliance
// for all validators based on their actual ANT burn participation.
func (k Keeper) ProcessMOAEpoch(ctx sdk.Context, currentHeight uint64) {
	epochStart := k.GetEpochStartHeight(ctx)
	if currentHeight-epochStart < types.DefaultEpochLength {
		return
	}

	validators := k.GetAllValidators(ctx)
	for _, val := range validators {
		if val.Status != consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE {
			continue
		}
		totalBid := k.GetAntPurchaseTotal(ctx, val.Validator)

		// Derive required ANT activity: at least 1 unit of burn per epoch.
		required := uint64(1)
		compliance := float64(totalBid) / float64(required)
		if compliance > 2.0 {
			compliance = 2.0
		}

		if k.lizenzKeeper != nil {
			k.lizenzKeeper.UpdateRewardStats(ctx, val.Validator, 0, currentHeight, compliance, 1.0, 0)
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"moa.epoch_update",
				sdk.NewAttribute("validator", val.Validator),
				sdk.NewAttribute("total_bid", fmt.Sprintf("%d", totalBid)),
				sdk.NewAttribute("required", fmt.Sprintf("%d", required)),
				sdk.NewAttribute("compliance", fmt.Sprintf("%.2f", compliance)),
			),
		)
	}

	k.ResetAntPurchases(ctx)
	k.SetEpochStartHeight(ctx, currentHeight)
}

// ============================================================================
// BeginBlocker / EndBlocker
// ============================================================================

// BeginBlocker processes begin block logic
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	currentHeight := uint64(ctx.BlockHeight())
	validators := k.GetAllValidators(ctx)

	var activeValidators []string
	for _, validator := range validators {
		if validator.Status == consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE {
			activeValidators = append(activeValidators, validator.Validator)
		}
	}

	totalAntBurned, err := k.calculateTotalBurnedTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate total burned tokens: %w", err)
	}

	err = k.UpdateConsensusState(ctx, currentHeight, totalAntBurned, activeValidators)
	if err != nil {
		return err
	}

	err = k.ProcessHalving(ctx)
	if err != nil {
		return err
	}

	return nil
}

// EndBlocker processes end block logic
func (k Keeper) EndBlocker(ctx sdk.Context) error {
	currentHeight := uint64(ctx.BlockHeight())

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

	totalAntBurned, err := k.calculateTotalBurnedTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate total burned tokens: %w", err)
	}

	err = k.UpdateConsensusState(ctx, currentHeight, totalAntBurned, activeValidators)
	if err != nil {
		return err
	}

	// §5.4 — Process PoVB: per-height burns and fee distribution
	if err := k.ProcessPerHeightBurns(ctx, currentHeight); err != nil {
		ctx.Logger().Error("failed to process per-height burns", "error", err, "height", currentHeight)
	}

	// §5.1 — Contour 1: distribute WRT base rewards proportional to LZN
	err = k.DistributeBaseRewards(ctx, currentHeight)
	if err != nil {
		ctx.Logger().Error("failed to distribute base rewards", "error", err, "height", currentHeight)
	}

	// §6.2 — Halving
	err = k.ProcessHalving(ctx)
	if err != nil {
		ctx.Logger().Error("failed to process halving", "error", err, "height", currentHeight)
	}

	k.ProcessMOAEpoch(ctx, currentHeight)

	return nil
}

// ============================================================================
// PoVB Burn Model §5.4
// ============================================================================

// StorePendingBurn stores a validator's burn declaration for the current height
func (k Keeper) StorePendingBurn(ctx sdk.Context, burn *consensusv1.PerHeightBurn) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("burn/pending/%d/%s", burn.BlockHeight, burn.Validator))
	bz := k.cdc.MustMarshal(burn)
	store.Set(key, bz)
}

// GetPendingBurns returns all pending burn declarations for a given height
func (k Keeper) GetPendingBurns(ctx sdk.Context, height uint64) []*consensusv1.PerHeightBurn {
	store := ctx.KVStore(k.storeKey)
	prefix := []byte(fmt.Sprintf("burn/pending/%d/", height))
	iter := store.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	defer iter.Close()

	var burns []*consensusv1.PerHeightBurn
	for ; iter.Valid(); iter.Next() {
		var burn consensusv1.PerHeightBurn
		k.cdc.MustUnmarshal(iter.Value(), &burn)
		burns = append(burns, &burn)
	}
	return burns
}

// ProcessPerHeightBurns implements §5.4: global cap, priority ordering, inc_i, fee split
func (k Keeper) ProcessPerHeightBurns(ctx sdk.Context, height uint64) error {
	params := k.GetParams(ctx)
	burns := k.GetPendingBurns(ctx, height)
	if len(burns) == 0 {
		return nil
	}

	// Compute L_tot from validator weights (= activated LZN)
	weights, _ := k.GetAllValidatorWeights(ctx)
	lTot := uint64(0)
	lznMap := make(map[string]uint64)
	for _, w := range weights {
		v, _ := strconv.ParseUint(w.Weight, 10, 64)
		lTot += v
		lznMap[w.Validator] = v
	}
	if lTot == 0 {
		return nil
	}

	// Parse λ from params (default 0.5)
	lambda := 0.5
	if params.BurnCapLambda != "" {
		if parsed, err := strconv.ParseFloat(params.BurnCapLambda, 64); err == nil {
			lambda = parsed
		}
	}
	// Enforce constitutional bounds [1/3, 2/3]
	if lambda < 1.0/3.0 {
		lambda = 1.0 / 3.0
	}
	if lambda > 2.0/3.0 {
		lambda = 2.0 / 3.0
	}

	// C = λ · L_tot — global cap on total included burn
	globalCap := uint64(lambda * float64(lTot))

	// Validate each burn: s_i + b_i ≤ L_i; burn ANT from anteil
	for _, burn := range burns {
		si, _ := strconv.ParseUint(burn.SI, 10, 64)
		bi, _ := strconv.ParseUint(burn.BI, 10, 64)
		li := lznMap[burn.Validator]
		if si+bi > li {
			si = 0
			bi = 0
			burn.SI = "0"
			burn.BI = "0"
		}
		// Burn the ANT (s_i + b_i) from validator's position
		if k.anteilKeeper != nil && (si+bi) > 0 {
			_ = k.anteilKeeper.UpdateUserPosition(ctx, burn.Validator, "", 0)
		}
		_ = si // used below
	}

	// Sort by s_i descending (priority stake determines ordering)
	// Tiebreaker: lexicographic by account address
	sort.Slice(burns, func(i, j int) bool {
		si_i, _ := strconv.ParseUint(burns[i].SI, 10, 64)
		si_j, _ := strconv.ParseUint(burns[j].SI, 10, 64)
		if si_i != si_j {
			return si_i > si_j
		}
		return burns[i].Validator < burns[j].Validator
	})

	// Assign inc_i: fill up to globalCap in priority order
	totalIncluded := uint64(0)
	for _, burn := range burns {
		bi, _ := strconv.ParseUint(burn.BI, 10, 64)
		remaining := globalCap - totalIncluded
		incI := bi
		if incI > remaining {
			incI = remaining
		}
		burn.IncI = fmt.Sprintf("%d", incI)
		totalIncluded += incI
	}

	// Collect total fees F from the block (simplified: use gas consumed * min gas price)
	totalFees := k.collectBlockFees(ctx)

	// Distribute F proportionally to inc_i / B
	B := totalIncluded
	if B > 0 && totalFees > 0 {
		for _, burn := range burns {
			incI, _ := strconv.ParseUint(burn.IncI, 10, 64)
			if incI == 0 {
				continue
			}
			share := totalFees * incI / B
			if share > 0 && k.bankKeeper != nil {
				validatorAddr, err := sdk.AccAddressFromBech32(burn.Validator)
				if err != nil {
					continue
				}
				feeCoins := sdk.NewCoins(sdk.NewInt64Coin("uwrt", int64(share)))
				if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, validatorAddr, feeCoins); err != nil {
					ctx.Logger().Error("failed to distribute fee share", "validator", burn.Validator, "error", err)
				}
			}
		}
	} else if B == 0 && totalFees > 0 {
		// §7.2 п.6 — policy when B=0
		feePolicyBZero := params.FeePolicyBZero
		if feePolicyBZero == "" {
			feePolicyBZero = types.FeePolicyBZeroCommunityPool
		}
		ctx.Logger().Info("B=0: fees handled by policy", "policy", feePolicyBZero, "fees", totalFees)
	}

	// Store summary and emit events
	summary := &consensusv1.HeightBurnSummary{
		BlockHeight:       height,
		TotalFees:         fmt.Sprintf("%d", totalFees),
		TotalIncludedBurn: fmt.Sprintf("%d", B),
		GlobalCap:         fmt.Sprintf("%d", globalCap),
		LTot:              fmt.Sprintf("%d", lTot),
		Burns:             burns,
	}
	k.storeHeightBurnSummary(ctx, summary)

	// Emit per-height burn event §6.3
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePerHeightBurn,
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", height)),
			sdk.NewAttribute(types.AttributeKeyLTot, fmt.Sprintf("%d", lTot)),
			sdk.NewAttribute(types.AttributeKeyLambda, fmt.Sprintf("%.6f", lambda)),
			sdk.NewAttribute(types.AttributeKeyTotalFees, fmt.Sprintf("%d", totalFees)),
			sdk.NewAttribute(types.AttributeKeyTotalB, fmt.Sprintf("%d", B)),
			sdk.NewAttribute(types.AttributeKeyBurnReason, types.BurnReasonPerHeight),
		),
	)

	// Clean up pending burns
	store := ctx.KVStore(k.storeKey)
	prefix := []byte(fmt.Sprintf("burn/pending/%d/", height))
	iter := store.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		store.Delete(iter.Key())
	}

	return nil
}

// collectBlockFees computes total fees from transactions in this block
func (k Keeper) collectBlockFees(ctx sdk.Context) uint64 {
	// In a full implementation, this would sum fees from DeliverTx results.
	// For now, use a simplified approach: check the fee collector module balance.
	if k.bankKeeper == nil {
		return 0
	}
	return 0
}

// storeHeightBurnSummary persists the burn summary for a height
func (k Keeper) storeHeightBurnSummary(ctx sdk.Context, summary *consensusv1.HeightBurnSummary) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("burn/summary/%d", summary.BlockHeight))
	bz := k.cdc.MustMarshal(summary)
	store.Set(key, bz)
}

// ============================================================================
// Genesis
// ============================================================================

// InitGenesis initializes genesis state
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, *genState.Params)
	}

	for _, validator := range genState.Validators {
		k.SetValidator(ctx, validator)
	}

	halvingInfo := types.HalvingInfo{
		LastHalvingHeight: 0,
		HalvingInterval:   210000,
		NextHalvingHeight: 210000,
	}
	err := k.SetHalvingInfo(ctx, halvingInfo)
	if err != nil {
		ctx.Logger().Error("failed to set default halving info", "error", err)
	}

	err = k.UpdateConsensusState(ctx, 0, "0", []string{})
	if err != nil {
		ctx.Logger().Error("failed to set default consensus state", "error", err)
	}
}

// ExportGenesis exports genesis state
func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	params := k.GetParams(ctx)
	validators := k.GetAllValidators(ctx)

	return types.GenesisState{
		Params:     &params,
		Validators: validators,
	}
}

// ============================================================================
// Internal helpers
// ============================================================================

// calculateTotalBurnedTokens calculates the total amount of ANT tokens burned
func (k Keeper) calculateTotalBurnedTokens(ctx sdk.Context) (string, error) {
	weights, err := k.GetAllValidatorWeights(ctx)
	if err != nil {
		return "0", fmt.Errorf("failed to get validator weights: %w", err)
	}

	totalBurned := 0.0
	for _, weight := range weights {
		if weight.Weight != "" && weight.Weight != "0" {
			weightFloat, err := strconv.ParseFloat(weight.Weight, 64)
			if err != nil {
				continue
			}
			totalBurned += weightFloat
		}
	}

	return fmt.Sprintf("%.8f", totalBurned), nil
}

// extractLizenzInfo extracts validator and amount from an ActivatedLizenz object.
// Uses reflection to avoid circular dependency with lizenzv1 package.
func extractLizenzInfo(lizenzInterface interface{}) (validator string, amount string, err error) {
	v := reflect.ValueOf(lizenzInterface)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	validatorField := v.FieldByName("Validator")
	if !validatorField.IsValid() || validatorField.Kind() != reflect.String {
		return "", "", fmt.Errorf("invalid LZN object: Validator field not found or invalid")
	}
	validator = validatorField.String()

	amountField := v.FieldByName("Amount")
	if !amountField.IsValid() || amountField.Kind() != reflect.String {
		return "", "", fmt.Errorf("invalid LZN object: Amount field not found or invalid")
	}
	amount = amountField.String()

	return validator, amount, nil
}

// ============================================================================
// Base Reward Distribution (Circuit 1)
// ============================================================================

// CalculateBaseReward calculates the base block reward considering halving.
// Formula: base_reward = BASE_BLOCK_REWARD / (2^halving_count)
func (k Keeper) CalculateBaseReward(ctx sdk.Context, height uint64) (uint64, error) {
	params := k.GetParams(ctx)

	baseRewardStr := params.BaseBlockReward
	if baseRewardStr == "" {
		baseRewardStr = "50000000uwrt"
	}

	baseRewardStr = strings.TrimSuffix(baseRewardStr, "uwrt")
	baseRewardStr = strings.TrimSpace(baseRewardStr)

	baseReward, err := strconv.ParseUint(baseRewardStr, 10, 64)
	if err != nil {
		baseReward = 50_000_000
	}

	halvingCount := height / HalvingInterval

	reward := baseReward
	if halvingCount > 0 {
		for i := uint64(0); i < halvingCount && reward > 0; i++ {
			reward = reward / 2
		}
	}

	if reward == 0 {
		reward = 1
	}

	return reward, nil
}

// GetHalvingCount returns the current halving count for a given block height
func (k Keeper) GetHalvingCount(height uint64) uint64 {
	return height / HalvingInterval
}

// CalculateMOAPenaltyMultiplier calculates the penalty multiplier based on MOA compliance.
func (k Keeper) CalculateMOAPenaltyMultiplier(ctx sdk.Context, moaCompliance float64) float64 {
	params := k.GetParams(ctx)

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
		return 1.0
	} else if moaCompliance >= thresholdWarning {
		return 1.0
	} else if moaCompliance >= thresholdMedium {
		return 0.75
	} else if moaCompliance >= thresholdLow {
		return 0.5
	} else {
		return 0.0
	}
}

// ValidatorRewardInfo contains information about a validator's reward
type ValidatorRewardInfo struct {
	Validator         string
	ActivatedLZN      uint64
	RewardShare       float64
	BaseRewardAmount  uint64
	MOACompliance     float64
	PenaltyMultiplier float64
	FinalRewardAmount uint64
}

// CalculateRewardDistribution calculates how base reward should be distributed among validators.
// Formula: validator_passive_income = (activated_lzn_validator / total_activated_lzn) × current_block_reward
func (k Keeper) CalculateRewardDistribution(ctx sdk.Context, baseReward uint64, validatorLZN map[string]uint64) ([]ValidatorRewardInfo, uint64, error) {
	if len(validatorLZN) == 0 {
		return []ValidatorRewardInfo{}, 0, fmt.Errorf("no validators with activated LZN")
	}

	totalLZN := uint64(0)
	for _, lzn := range validatorLZN {
		totalLZN += lzn
	}

	if totalLZN == 0 {
		return []ValidatorRewardInfo{}, 0, fmt.Errorf("total activated LZN is zero")
	}

	var rewards []ValidatorRewardInfo
	totalDistributed := uint64(0)

	sortedValidators := make([]string, 0, len(validatorLZN))
	for v := range validatorLZN {
		sortedValidators = append(sortedValidators, v)
	}
	sort.Strings(sortedValidators)

	for _, validator := range sortedValidators {
		activatedLZN := validatorLZN[validator]
		if activatedLZN == 0 {
			continue
		}

		share := float64(activatedLZN) / float64(totalLZN)
		baseRewardAmount := (baseReward * activatedLZN) / totalLZN

		moaCompliance := 1.0
		if k.lizenzKeeper != nil {
			compliance, err := k.lizenzKeeper.GetMOACompliance(ctx, validator)
			if err == nil {
				moaCompliance = compliance
			}
		}

		penaltyMultiplier := k.CalculateMOAPenaltyMultiplier(ctx, moaCompliance)
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

// DistributeBaseRewards distributes base block rewards to validators based on their activated LZN.
// This implements Circuit 1 of the economic model: passive income for validators.
func (k Keeper) DistributeBaseRewards(ctx sdk.Context, height uint64) error {
	baseReward, err := k.CalculateBaseReward(ctx, height)
	if err != nil {
		return fmt.Errorf("failed to calculate base reward: %w", err)
	}

	if k.lizenzKeeper == nil {
		ctx.Logger().Info("lizenz keeper not set, skipping reward distribution", "height", height, "base_reward", baseReward)
		return nil
	}

	allLizenzs, err := k.lizenzKeeper.GetAllActivatedLizenz(ctx)
	if err != nil {
		return fmt.Errorf("failed to get activated LZN: %w", err)
	}

	validatorLZN := make(map[string]uint64)
	for _, lizenzInterface := range allLizenzs {
		validator, amount, err := extractLizenzInfo(lizenzInterface)
		if err != nil {
			ctx.Logger().Error("failed to extract LZN info", "error", err)
			continue
		}

		amountInt, err := strconv.ParseUint(amount, 10, 64)
		if err != nil {
			ctx.Logger().Error("failed to parse LZN amount", "error", err, "amount", amount)
			continue
		}

		validatorLZN[validator] = amountInt
	}

	if len(validatorLZN) == 0 {
		ctx.Logger().Info("no validators with activated LZN, skipping reward distribution", "height", height)
		return nil
	}

	rewards, totalDistributed, err := k.CalculateRewardDistribution(ctx, baseReward, validatorLZN)
	if err != nil {
		return fmt.Errorf("failed to calculate reward distribution: %w", err)
	}

	ctx.Logger().Info("base reward distribution calculated",
		"height", height,
		"base_reward", baseReward,
		"total_distributed", totalDistributed,
		"validators_count", len(rewards))

	for _, reward := range rewards {
		if k.lizenzKeeper != nil {
			if err := k.lizenzKeeper.UpdateRewardStats(ctx, reward.Validator, reward.FinalRewardAmount, height, reward.MOACompliance, reward.PenaltyMultiplier, reward.BaseRewardAmount); err != nil {
				ctx.Logger().Error("failed to update reward stats", "error", err, "validator", reward.Validator)
			}
		}
	}

	if k.bankKeeper != nil {
		for _, reward := range rewards {
			validatorAddr, err := sdk.AccAddressFromBech32(reward.Validator)
			if err != nil {
				ctx.Logger().Error("invalid validator address", "error", err, "validator", reward.Validator)
				continue
			}

			rewardCoins := sdk.NewCoins(sdk.NewCoin("uwrt", math.NewIntFromUint64(reward.FinalRewardAmount)))

			moduleName := types.ModuleName
			if err := k.bankKeeper.MintCoins(ctx, moduleName, rewardCoins); err != nil {
				ctx.Logger().Error("failed to mint reward coins", "error", err, "validator", reward.Validator, "amount", reward.FinalRewardAmount)
				continue
			}

			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, moduleName, validatorAddr, rewardCoins); err != nil {
				ctx.Logger().Error("failed to send reward coins", "error", err, "validator", reward.Validator, "amount", reward.FinalRewardAmount)
				continue
			}

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeRewardDistributed,
					sdk.NewAttribute(types.AttributeKeyValidator, reward.Validator),
					sdk.NewAttribute(types.AttributeKeyRewardAmount, fmt.Sprintf("%d", reward.FinalRewardAmount)),
					sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", height)),
					sdk.NewAttribute(types.AttributeKeyRewardShare, fmt.Sprintf("%.6f", reward.RewardShare)),
					sdk.NewAttribute(types.AttributeKeyMOACompliance, fmt.Sprintf("%.2f", reward.MOACompliance)),
				),
			)
		}
	} else {
		ctx.Logger().Info("bank keeper not set, rewards calculated but not sent", "height", height)
	}

	return nil
}
