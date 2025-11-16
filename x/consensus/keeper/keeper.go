package keeper

import (
	"crypto/sha256"
	"encoding/hex"
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

// LizenzKeeperInterface defines the interface for interacting with lizenz module
// This allows consensus module to get information about activated LZN and MOA status
type LizenzKeeperInterface interface {
	GetAllActivatedLizenz(ctx sdk.Context) ([]interface{}, error) // Returns list of activated LZN
	GetTotalActivatedLizenz(ctx sdk.Context) (string, error)     // Returns total activated LZN
	GetMOACompliance(ctx sdk.Context, validator string) (float64, error) // Returns MOA compliance ratio (0.0 to 1.0+)
}

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeKey     storetypes.StoreKey
		paramstore   paramtypes.Subspace
		lizenzKeeper LizenzKeeperInterface // Optional: for reward distribution
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

	// Distribute base rewards to validators (Circuit 1)
	// This happens after block creation, distributing passive income based on activated LZN
	err = k.DistributeBaseRewards(ctx, currentHeight)
	if err != nil {
		ctx.Logger().Error("failed to distribute base rewards", "error", err, "height", currentHeight)
		// Don't fail the block if reward distribution fails
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

	return k.SetBlindAuction(ctx, auction)
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

	// Validate bid amount
	bidAmountInt, err := strconv.ParseUint(bidAmount, 10, 64)
	if err != nil || bidAmountInt == 0 {
		return types.ErrInvalidBidAmount
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
			auction.Winner = winner.Validator
			auction.WinningBid = winner.BidAmount
			auction.Phase = consensusv1.AuctionPhase_AUCTION_PHASE_COMPLETE
			auction.EndTime = timestamppb.Now()

			err = k.SetBlindAuction(ctx, auction)
			if err != nil {
				return "", "", err
			}

			return winner.Validator, winner.BidAmount, nil
		}
	}

	// Fallback to first reveal (should not happen)
	winner := auction.Reveals[0]
	auction.Winner = winner.Validator
	auction.WinningBid = winner.BidAmount
	auction.Phase = consensusv1.AuctionPhase_AUCTION_PHASE_COMPLETE
	auction.EndTime = timestamppb.Now()

	err = k.SetBlindAuction(ctx, auction)
	if err != nil {
		return "", "", err
	}

	return winner.Validator, winner.BidAmount, nil
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

// ============================================================================
// Base Reward Distribution (Circuit 1)
// ============================================================================

const (
	// BaseBlockReward is the base reward per block in micro WRT (50 WRT = 50,000,000 uwrt)
	BaseBlockReward = 50_000_000 // 50 WRT in micro units
	// HalvingInterval is the number of blocks between halvings (210,000 blocks)
	HalvingInterval = 210_000
)

// CalculateBaseReward calculates the base block reward considering halving
// Formula: base_reward = BASE_BLOCK_REWARD / (2^halving_count)
// where halving_count = floor(block_height / HALVING_INTERVAL)
func (k Keeper) CalculateBaseReward(ctx sdk.Context, height uint64) (uint64, error) {
	// Calculate halving count
	halvingCount := height / HalvingInterval

	// Calculate reward: base_reward / (2^halving_count)
	// Use bit shift for efficiency: 2^halving_count = 1 << halvingCount
	reward := uint64(BaseBlockReward)
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

// CalculateMOAPenaltyMultiplier calculates the penalty multiplier based on MOA compliance
// According to whitepaper and economic-formulas.md:
// - >= 1.0: no penalty (1.0)
// - 0.9-1.0: warning (1.0, but logged)
// - 0.7-0.9: 25% penalty (0.75)
// - 0.5-0.7: 50% penalty (0.5)
// - < 0.5: deactivation (0.0)
func CalculateMOAPenaltyMultiplier(moaCompliance float64) float64 {
	if moaCompliance >= 1.0 {
		return 1.0 // No penalty
	} else if moaCompliance >= 0.9 {
		return 1.0 // Warning, but no penalty
	} else if moaCompliance >= 0.7 {
		return 0.75 // 25% penalty
	} else if moaCompliance >= 0.5 {
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
		penaltyMultiplier := CalculateMOAPenaltyMultiplier(moaCompliance)

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
		// Type assertion to get the actual lizenz object
		// We need to handle this carefully since we're using interface{}
		// For now, we'll assume the interface provides the necessary methods
		// In a real implementation, we'd use a proper type
		// This is a placeholder - actual implementation would need proper type casting
		_ = lizenzInterface
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

	// Log MOA compliance details for each validator
	for _, reward := range rewards {
		if reward.MOACompliance < 1.0 {
			ctx.Logger().Info("validator MOA penalty applied",
				"validator", reward.Validator,
				"moa_compliance", reward.MOACompliance,
				"penalty_multiplier", reward.PenaltyMultiplier,
				"base_reward", reward.BaseRewardAmount,
				"final_reward", reward.FinalRewardAmount)
		}
	}

	// TODO: Actually send WRT tokens to validators via bank module
	// This requires integration with bank keeper
	// For now, we just calculate and log the distribution

	return nil
}
