package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
	"github.com/volnix-protocol/volnix-protocol/x/governance/types"
)

type QueryServer struct {
	governancev1.UnimplementedQueryServer
	k *Keeper
}

func NewQueryServer(k *Keeper) QueryServer {
	return QueryServer{k: k}
}

// Proposal queries a proposal by ID
func (qs QueryServer) Proposal(ctx context.Context, req *governancev1.QueryProposalRequest) (*governancev1.QueryProposalResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	proposal, err := qs.k.GetProposal(sdkCtx, req.ProposalId)
	if err != nil {
		return nil, err
	}

	return &governancev1.QueryProposalResponse{
		Proposal: proposal,
	}, nil
}

// Proposals queries all proposals (optionally filtered by status)
func (qs QueryServer) Proposals(ctx context.Context, req *governancev1.QueryProposalsRequest) (*governancev1.QueryProposalsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	allProposals, err := qs.k.GetAllProposals(sdkCtx)
	if err != nil {
		return nil, err
	}

	// Filter by status if specified
	var proposals []*governancev1.Proposal
	if req.Status == governancev1.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED {
		proposals = allProposals
	} else {
		for _, proposal := range allProposals {
			if proposal.Status == req.Status {
				proposals = append(proposals, proposal)
			}
		}
	}

	return &governancev1.QueryProposalsResponse{
		Proposals: proposals,
	}, nil
}

// Vote queries a vote on a proposal
func (qs QueryServer) Vote(ctx context.Context, req *governancev1.QueryVoteRequest) (*governancev1.QueryVoteResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	vote, err := qs.k.GetVote(sdkCtx, req.ProposalId, req.Voter)
	if err != nil {
		return nil, err
	}

	return &governancev1.QueryVoteResponse{
		Vote: vote,
	}, nil
}

// Votes queries all votes on a proposal
func (qs QueryServer) Votes(ctx context.Context, req *governancev1.QueryVotesRequest) (*governancev1.QueryVotesResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	votes, err := qs.k.GetVotes(sdkCtx, req.ProposalId)
	if err != nil {
		return nil, err
	}

	return &governancev1.QueryVotesResponse{
		Votes: votes,
	}, nil
}

// Params queries governance parameters
func (qs QueryServer) Params(ctx context.Context, req *governancev1.QueryParamsRequest) (*governancev1.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	params := qs.k.GetParams(sdkCtx)

	// Convert types.Params to governancev1.Params
	govParams := &governancev1.Params{
		VotingPeriod:  types.DurationToProto(params.VotingPeriod),
		TimelockPeriod: types.DurationToProto(params.TimelockPeriod),
		MinDeposit:    params.MinDeposit,
		Quorum:        params.Quorum,
		Threshold:     params.Threshold,
	}

	return &governancev1.QueryParamsResponse{
		Params: govParams,
	}, nil
}

