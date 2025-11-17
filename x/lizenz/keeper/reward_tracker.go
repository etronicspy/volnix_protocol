package keeper

import (
	"encoding/json"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

// RewardRecord represents a single reward distribution record
type RewardRecord struct {
	BlockHeight    uint64  `json:"block_height"`
	RewardAmount   string  `json:"reward_amount"` // in micro WRT
	Timestamp      int64   `json:"timestamp"`
	MOACompliance  float64 `json:"moa_compliance"`
	PenaltyApplied float64 `json:"penalty_applied"`
	BaseReward     string  `json:"base_reward"` // Base reward before penalty
}

// UpdateRewardStats updates the reward statistics for a validator
// This is called after reward distribution in consensus module
func (k Keeper) UpdateRewardStats(ctx sdk.Context, validator string, rewardAmount uint64, blockHeight uint64, moaCompliance float64, penaltyMultiplier float64, baseReward uint64) error {
	// Get current activated LZN
	activatedLizenz, err := k.GetActivatedLizenz(ctx, validator)
	if err != nil {
		return fmt.Errorf("validator has no activated LZN: %w", err)
	}

	// Update total rewards earned
	var currentTotal uint64 = 0
	if activatedLizenz.TotalRewardsEarned != "" {
		var err error
		currentTotal, err = strconv.ParseUint(activatedLizenz.TotalRewardsEarned, 10, 64)
		if err != nil {
			currentTotal = 0
		}
	}
	
	newTotal := currentTotal + rewardAmount
	activatedLizenz.TotalRewardsEarned = fmt.Sprintf("%d", newTotal)
	activatedLizenz.LastRewardBlock = fmt.Sprintf("%d", blockHeight)
	activatedLizenz.LastRewardTime = timestamppb.Now()

	// Store updated activated LZN (use UpdateActivatedLizenz since it already exists)
	if err := k.UpdateActivatedLizenz(ctx, activatedLizenz); err != nil {
		return fmt.Errorf("failed to update activated LZN: %w", err)
	}

	// Record reward in history
	rewardRecord := RewardRecord{
		BlockHeight:    blockHeight,
		RewardAmount:   fmt.Sprintf("%d", rewardAmount),
		Timestamp:      ctx.BlockTime().Unix(),
		MOACompliance:  moaCompliance,
		PenaltyApplied: penaltyMultiplier,
		BaseReward:     fmt.Sprintf("%d", baseReward),
	}

	if err := k.RecordRewardHistory(ctx, validator, rewardRecord); err != nil {
		ctx.Logger().Error("failed to record reward history", "error", err, "validator", validator)
		// Don't fail if history recording fails
	}

	ctx.Logger().Info("reward stats updated",
		"validator", validator,
		"reward_amount", rewardAmount,
		"total_rewards", newTotal,
		"block_height", blockHeight)

	return nil
}

// RecordRewardHistory records a reward in the validator's reward history
func (k Keeper) RecordRewardHistory(ctx sdk.Context, validator string, record RewardRecord) error {
	store := ctx.KVStore(k.storeKey)
	historyKey := types.GetRewardHistoryKey(validator)

	// Get existing history
	var history []RewardRecord
	bz := store.Get(historyKey)
	if bz != nil {
		if err := json.Unmarshal(bz, &history); err != nil {
			// If unmarshal fails, start with empty history
			history = []RewardRecord{}
		}
	}

	// Add new record
	history = append(history, record)

	// Keep only last 1000 records (to prevent unbounded growth)
	if len(history) > 1000 {
		history = history[len(history)-1000:]
	}

	// Store updated history
	historyBz, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal reward history: %w", err)
	}

	store.Set(historyKey, historyBz)
	return nil
}

// GetRewardHistory retrieves reward history for a validator
func (k Keeper) GetRewardHistory(ctx sdk.Context, validator string) ([]RewardRecord, error) {
	store := ctx.KVStore(k.storeKey)
	historyKey := types.GetRewardHistoryKey(validator)

	bz := store.Get(historyKey)
	if bz == nil {
		return []RewardRecord{}, nil // Return empty history if not found
	}

	var history []RewardRecord
	if err := json.Unmarshal(bz, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reward history: %w", err)
	}

	return history, nil
}

// GetTotalRewardsEarned returns the total rewards earned by a validator
func (k Keeper) GetTotalRewardsEarned(ctx sdk.Context, validator string) (string, error) {
	activatedLizenz, err := k.GetActivatedLizenz(ctx, validator)
	if err != nil {
		return "0", fmt.Errorf("validator has no activated LZN: %w", err)
	}

	return activatedLizenz.TotalRewardsEarned, nil
}

// GetRewardStats returns comprehensive reward statistics for a validator
type RewardStats struct {
	TotalRewardsEarned string         `json:"total_rewards_earned"`
	LastRewardBlock    string         `json:"last_reward_block"`
	LastRewardTime     *timestamppb.Timestamp `json:"last_reward_time"`
	RewardHistory      []RewardRecord `json:"reward_history"`
	TotalRewardsCount  int            `json:"total_rewards_count"`
}

func (k Keeper) GetRewardStats(ctx sdk.Context, validator string) (*RewardStats, error) {
	activatedLizenz, err := k.GetActivatedLizenz(ctx, validator)
	if err != nil {
		return nil, fmt.Errorf("validator has no activated LZN: %w", err)
	}

	history, err := k.GetRewardHistory(ctx, validator)
	if err != nil {
		// If history retrieval fails, continue with empty history
		history = []RewardRecord{}
	}

	return &RewardStats{
		TotalRewardsEarned: activatedLizenz.TotalRewardsEarned,
		LastRewardBlock:    activatedLizenz.LastRewardBlock,
		LastRewardTime:     activatedLizenz.LastRewardTime,
		RewardHistory:      history,
		TotalRewardsCount:  len(history),
	}, nil
}

