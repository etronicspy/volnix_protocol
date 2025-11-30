package keeper

import (
	"fmt"
	"strconv"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
	"github.com/volnix-protocol/volnix-protocol/x/governance/types"
)

// Type aliases for convenience
type Proposal = governancev1.Proposal
type Vote = governancev1.Vote
type ParameterChange = governancev1.ParameterChange

// WRT tokenomics constants (from whitepaper)
const (
	// TotalWRTSupply is the total supply of WRT tokens
	// According to whitepaper: 21,000,000 WRT (similar to Bitcoin)
	TotalWRTSupply = uint64(21000000000000) // 21M WRT in micro units (uwrt)
)

// BankKeeperInterface defines the interface for interacting with bank module
// This allows governance module to get WRT balances for voting power
// According to whitepaper: "Право голоса в этом DAO принадлежит исключительно держателям WRT"
type BankKeeperInterface interface {
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	// GetSupply returns the total supply of a denomination
	// Optional: if not available, we'll use a constant from whitepaper
	GetSupply(ctx sdk.Context, denom string) sdk.Coin
}

type (
	Keeper struct {
		cdc            codec.BinaryCodec
		storeKey       storetypes.StoreKey
		paramstore     paramtypes.Subspace
		bankKeeper     BankKeeperInterface // For getting WRT balances
		lizenzKeeper  LizenzKeeperForGovernance // Optional: for lizenz parameter updates
		anteilKeeper   AnteilKeeperForGovernance // Optional: for anteil parameter updates
		consensusKeeper ConsensusKeeperForGovernance // Optional: for consensus parameter updates
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

// SetBankKeeper sets the bank keeper interface for WRT balance queries
func (k *Keeper) SetBankKeeper(bankKeeper BankKeeperInterface) {
	k.bankKeeper = bankKeeper
}

// SetLizenzKeeper sets the lizenz keeper interface for parameter updates
func (k *Keeper) SetLizenzKeeper(lizenzKeeper LizenzKeeperForGovernance) {
	k.lizenzKeeper = lizenzKeeper
}

// SetAnteilKeeper sets the anteil keeper interface for parameter updates
func (k *Keeper) SetAnteilKeeper(anteilKeeper AnteilKeeperForGovernance) {
	k.anteilKeeper = anteilKeeper
}

// SetConsensusKeeper sets the consensus keeper interface for parameter updates
func (k *Keeper) SetConsensusKeeper(consensusKeeper ConsensusKeeperForGovernance) {
	k.consensusKeeper = consensusKeeper
}

// GetParams returns the current parameters for the governance module
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var params types.Params
	k.paramstore.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the parameters for the governance module
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// GetNextProposalID returns the next proposal ID
func (k Keeper) GetNextProposalID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ProposalIDKey)
	if bz == nil {
		return 1
	}
	return sdk.BigEndianToUint64(bz)
}

// SetNextProposalID sets the next proposal ID
func (k Keeper) SetNextProposalID(ctx sdk.Context, id uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ProposalIDKey, sdk.Uint64ToBigEndian(id))
}

// SetProposal stores a proposal in the store
func (k Keeper) SetProposal(ctx sdk.Context, proposal *Proposal) error {
	store := ctx.KVStore(k.storeKey)
	proposalKey := types.GetProposalKey(proposal.ProposalId)

	// Use proto encoding
	proposalBz, err := k.cdc.Marshal(proposal)
	if err != nil {
		return fmt.Errorf("failed to marshal proposal: %w", err)
	}

	store.Set(proposalKey, proposalBz)
	return nil
}

// GetProposal retrieves a proposal by ID
func (k Keeper) GetProposal(ctx sdk.Context, proposalID uint64) (*Proposal, error) {
	store := ctx.KVStore(k.storeKey)
	proposalKey := types.GetProposalKey(proposalID)

	bz := store.Get(proposalKey)
	if bz == nil {
		return nil, types.ErrProposalNotFound
	}

	var proposal Proposal
	if err := k.cdc.Unmarshal(bz, &proposal); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proposal: %w", err)
	}

	return &proposal, nil
}

// GetAllProposals retrieves all proposals
func (k Keeper) GetAllProposals(ctx sdk.Context) ([]*Proposal, error) {
	store := ctx.KVStore(k.storeKey)
	proposalStore := prefix.NewStore(store, types.ProposalKeyPrefix)

	var proposals []*Proposal
	iterator := proposalStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var proposal Proposal
		if err := k.cdc.Unmarshal(iterator.Value(), &proposal); err != nil {
			continue // Skip invalid proposals
		}
		proposals = append(proposals, &proposal)
	}

	return proposals, nil
}

// SetVote stores a vote in the store
func (k Keeper) SetVote(ctx sdk.Context, vote *Vote) error {
	store := ctx.KVStore(k.storeKey)
	voteKey := types.GetVoteKey(vote.ProposalId, vote.Voter)

	// Use proto encoding
	voteBz, err := k.cdc.Marshal(vote)
	if err != nil {
		return fmt.Errorf("failed to marshal vote: %w", err)
	}

	store.Set(voteKey, voteBz)
	return nil
}

// GetVote retrieves a vote by proposal ID and voter
func (k Keeper) GetVote(ctx sdk.Context, proposalID uint64, voter string) (*Vote, error) {
	store := ctx.KVStore(k.storeKey)
	voteKey := types.GetVoteKey(proposalID, voter)

	bz := store.Get(voteKey)
	if bz == nil {
		return nil, types.ErrVoteNotFound
	}

	var vote Vote
	if err := k.cdc.Unmarshal(bz, &vote); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vote: %w", err)
	}

	return &vote, nil
}

// GetVotes retrieves all votes for a proposal
func (k Keeper) GetVotes(ctx sdk.Context, proposalID uint64) ([]*Vote, error) {
	store := ctx.KVStore(k.storeKey)
	voteStore := prefix.NewStore(store, types.VoteKeyPrefix)

	var votes []*Vote
	iterator := voteStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var vote Vote
		if err := k.cdc.Unmarshal(iterator.Value(), &vote); err != nil {
			continue // Skip invalid votes
		}
		if vote.ProposalId == proposalID {
			votes = append(votes, &vote)
		}
	}

	return votes, nil
}

// GetWRTBalance returns the WRT balance for an address
// According to whitepaper: voting power is based on WRT holdings
func (k Keeper) GetWRTBalance(ctx sdk.Context, addr sdk.AccAddress) uint64 {
	if k.bankKeeper == nil {
		return 0
	}

	// Get WRT balance (denom: "uwrt")
	balance := k.bankKeeper.GetBalance(ctx, addr, "uwrt")
	return balance.Amount.Uint64()
}

// GetTotalWRTSupply returns the total WRT supply for quorum calculations
// According to whitepaper: 21,000,000 WRT total supply
// Tries to get actual supply from bank keeper, falls back to constant if not available
func (k Keeper) GetTotalWRTSupply(ctx sdk.Context) uint64 {
	// Default: use constant from whitepaper
	totalWRT := TotalWRTSupply
	
	// Try to get actual supply from bank keeper if available
	if k.bankKeeper != nil {
		// Check if bank keeper implements GetSupply method
		// This is optional - if not available, use constant
		if supplyKeeper, ok := k.bankKeeper.(interface {
			GetSupply(ctx sdk.Context, denom string) sdk.Coin
		}); ok {
			supply := supplyKeeper.GetSupply(ctx, "uwrt")
			if !supply.Amount.IsZero() {
				totalWRT = supply.Amount.Uint64()
			}
		}
	}
	
	return totalWRT
}

// CalculateVotingPower calculates voting power based on WRT balance
func (k Keeper) CalculateVotingPower(ctx sdk.Context, voter string) (string, error) {
	addr, err := sdk.AccAddressFromBech32(voter)
	if err != nil {
		return "0", fmt.Errorf("invalid voter address: %w", err)
	}

	balance := k.GetWRTBalance(ctx, addr)
	return fmt.Sprintf("%d", balance), nil
}

// TallyVotes tallies votes for a proposal and updates proposal status
func (k Keeper) TallyVotes(ctx sdk.Context, proposalID uint64) error {
	proposal, err := k.GetProposal(ctx, proposalID)
	if err != nil {
		return err
	}

	// Get all votes
	votes, err := k.GetVotes(ctx, proposalID)
	if err != nil {
		return err
	}

	// Calculate totals
	var yesVotes, noVotes, abstainVotes uint64
	for _, vote := range votes {
		votingPower, err := strconv.ParseUint(vote.VotingPower, 10, 64)
		if err != nil {
			continue // Skip invalid votes
		}

		switch vote.Option {
		case governancev1.VoteOption_VOTE_OPTION_YES:
			yesVotes += votingPower
		case governancev1.VoteOption_VOTE_OPTION_NO:
			noVotes += votingPower
		case governancev1.VoteOption_VOTE_OPTION_ABSTAIN:
			abstainVotes += votingPower
		}
	}

	totalVotes := yesVotes + noVotes + abstainVotes

	// Update proposal with vote totals
	proposal.YesVotes = fmt.Sprintf("%d", yesVotes)
	proposal.NoVotes = fmt.Sprintf("%d", noVotes)
	proposal.AbstainVotes = fmt.Sprintf("%d", abstainVotes)
	proposal.TotalVotes = fmt.Sprintf("%d", totalVotes)

	// Check if proposal passed
	params := k.GetParams(ctx)
	if k.isProposalPassed(ctx, proposal, params) {
		proposal.Status = governancev1.ProposalStatus_PROPOSAL_STATUS_PASSED
		// Set execution time (after timelock)
		executionTime := ctx.BlockTime().Add(params.TimelockPeriod)
		proposal.ExecutionTime = timestamppb.New(executionTime)
	} else {
		proposal.Status = governancev1.ProposalStatus_PROPOSAL_STATUS_REJECTED
	}

	return k.SetProposal(ctx, proposal)
}

// isProposalPassed checks if a proposal has passed based on quorum and threshold
func (k Keeper) isProposalPassed(ctx sdk.Context, proposal *Proposal, params types.Params) bool {
	// Parse vote totals
	yesVotes, _ := strconv.ParseUint(proposal.YesVotes, 10, 64)
	noVotes, _ := strconv.ParseUint(proposal.NoVotes, 10, 64)
	totalVotes, _ := strconv.ParseUint(proposal.TotalVotes, 10, 64)

	// Get total WRT supply (for quorum calculation)
	totalWRT := k.GetTotalWRTSupply(ctx)

	// Check quorum
	quorum, _ := strconv.ParseFloat(params.Quorum, 64)
	quorumThreshold := uint64(float64(totalWRT) * quorum)
	if totalVotes < quorumThreshold {
		return false // Quorum not met
	}

	// Check threshold (yes votes must be > threshold of total votes)
	threshold, _ := strconv.ParseFloat(params.Threshold, 64)
	thresholdVotes := uint64(float64(totalVotes) * threshold)
	if yesVotes <= thresholdVotes {
		return false // Threshold not met
	}

	// Proposal passed if yes votes > no votes
	return yesVotes > noVotes
}

// CanExecuteProposal checks if a proposal can be executed (timelock expired)
func (k Keeper) CanExecuteProposal(ctx sdk.Context, proposalID uint64) (bool, error) {
	proposal, err := k.GetProposal(ctx, proposalID)
	if err != nil {
		return false, err
	}

	if proposal.Status != governancev1.ProposalStatus_PROPOSAL_STATUS_PASSED {
		return false, types.ErrProposalNotPassed
	}

	if proposal.ExecutionTime == nil {
		return false, fmt.Errorf("execution time not set")
	}

	// Check if timelock has expired
	if ctx.BlockTime().Before(proposal.ExecutionTime.AsTime()) {
		return false, types.ErrProposalNotExecutable
	}

	return true, nil
}

