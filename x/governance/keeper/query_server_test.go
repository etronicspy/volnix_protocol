package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
	"github.com/volnix-protocol/volnix-protocol/x/governance/types"
)

type QueryServerTestSuite struct {
	suite.Suite

	cdc           codec.Codec
	ctx           sdk.Context
	keeper        *Keeper
	queryServer   governancev1.QueryServer
	mockBankKeeper *MockBankKeeperForGovernance
	storeKey      storetypes.StoreKey
	paramStore    paramtypes.Subspace
}

func TestQueryServerTestSuite(t *testing.T) {
	suite.Run(t, new(QueryServerTestSuite))
}

func (suite *QueryServerTestSuite) SetupTest() {
	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)

	// Create store keys
	suite.storeKey = storetypes.NewKVStoreKey("test_governance")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")

	// Create test context
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)

	// Create params keeper and subspace
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore.WithKeyTable(types.ParamKeyTable())

	// Create keeper and query server
	suite.keeper = NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)
	suite.queryServer = NewQueryServer(suite.keeper)

	// Create and set mock bank keeper
	suite.mockBankKeeper = NewMockBankKeeperForGovernance()
	suite.keeper.SetBankKeeper(suite.mockBankKeeper)

	// Set default params
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

// Test QueryProposal
func (suite *QueryServerTestSuite) TestQueryProposal() {
	// Create a proposal
	proposal := &governancev1.Proposal{
		ProposalId:   1,
		Proposer:     sdk.AccAddress("test_proposer_12345678901234567890").String(),
		Title:        "Test Proposal",
		Description:  "This is a test proposal",
		Status:       governancev1.ProposalStatus_PROPOSAL_STATUS_SUBMITTED,
		SubmitTime:   timestamppb.Now(),
		YesVotes:     "0",
		NoVotes:      "0",
		AbstainVotes: "0",
		TotalVotes:   "0",
	}
	err := suite.keeper.SetProposal(suite.ctx, proposal)
	require.NoError(suite.T(), err)

	// Query proposal
	req := &governancev1.QueryProposalRequest{
		ProposalId: 1,
	}
	resp, err := suite.queryServer.Proposal(suite.ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotNil(suite.T(), resp.Proposal)
	require.Equal(suite.T(), uint64(1), resp.Proposal.ProposalId)
	require.Equal(suite.T(), "Test Proposal", resp.Proposal.Title)
}

func (suite *QueryServerTestSuite) TestQueryProposal_NotFound() {
	req := &governancev1.QueryProposalRequest{
		ProposalId: 999,
	}
	_, err := suite.queryServer.Proposal(suite.ctx, req)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "not found")
}

// Test QueryProposals
func (suite *QueryServerTestSuite) TestQueryProposals() {
	// Create multiple proposals
	proposal1 := &governancev1.Proposal{
		ProposalId:   1,
		Proposer:     sdk.AccAddress("test_proposer1_12345678901234567890").String(),
		Title:        "Proposal 1",
		Description:  "First proposal",
		Status:       governancev1.ProposalStatus_PROPOSAL_STATUS_SUBMITTED,
		SubmitTime:   timestamppb.Now(),
		YesVotes:     "0",
		NoVotes:      "0",
		AbstainVotes: "0",
		TotalVotes:   "0",
	}
	proposal2 := &governancev1.Proposal{
		ProposalId:   2,
		Proposer:     sdk.AccAddress("test_proposer2_12345678901234567890").String(),
		Title:        "Proposal 2",
		Description:  "Second proposal",
		Status:       governancev1.ProposalStatus_PROPOSAL_STATUS_VOTING,
		SubmitTime:   timestamppb.Now(),
		YesVotes:     "0",
		NoVotes:      "0",
		AbstainVotes: "0",
		TotalVotes:   "0",
	}
	err := suite.keeper.SetProposal(suite.ctx, proposal1)
	require.NoError(suite.T(), err)
	err = suite.keeper.SetProposal(suite.ctx, proposal2)
	require.NoError(suite.T(), err)

	// Query all proposals
	req := &governancev1.QueryProposalsRequest{
		Status: governancev1.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED,
	}
	resp, err := suite.queryServer.Proposals(suite.ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.GreaterOrEqual(suite.T(), len(resp.Proposals), 2, "Should have at least 2 proposals")
}

func (suite *QueryServerTestSuite) TestQueryProposals_FilterByStatus() {
	// Create proposals with different statuses
	proposal1 := &governancev1.Proposal{
		ProposalId:   1,
		Proposer:     sdk.AccAddress("test_proposer1_12345678901234567890").String(),
		Title:        "Proposal 1",
		Description:  "Submitted proposal",
		Status:       governancev1.ProposalStatus_PROPOSAL_STATUS_SUBMITTED,
		SubmitTime:   timestamppb.Now(),
		YesVotes:     "0",
		NoVotes:      "0",
		AbstainVotes: "0",
		TotalVotes:   "0",
	}
	proposal2 := &governancev1.Proposal{
		ProposalId:   2,
		Proposer:     sdk.AccAddress("test_proposer2_12345678901234567890").String(),
		Title:        "Proposal 2",
		Description:  "Voting proposal",
		Status:       governancev1.ProposalStatus_PROPOSAL_STATUS_VOTING,
		SubmitTime:   timestamppb.Now(),
		YesVotes:     "0",
		NoVotes:      "0",
		AbstainVotes: "0",
		TotalVotes:   "0",
	}
	err := suite.keeper.SetProposal(suite.ctx, proposal1)
	require.NoError(suite.T(), err)
	err = suite.keeper.SetProposal(suite.ctx, proposal2)
	require.NoError(suite.T(), err)

	// Query only SUBMITTED proposals
	req := &governancev1.QueryProposalsRequest{
		Status: governancev1.ProposalStatus_PROPOSAL_STATUS_SUBMITTED,
	}
	resp, err := suite.queryServer.Proposals(suite.ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	
	// All returned proposals should be SUBMITTED
	for _, proposal := range resp.Proposals {
		require.Equal(suite.T(), governancev1.ProposalStatus_PROPOSAL_STATUS_SUBMITTED, proposal.Status)
	}
}

// Test QueryVote
func (suite *QueryServerTestSuite) TestQueryVote() {
	// Create proposal
	proposal := &governancev1.Proposal{
		ProposalId:   1,
		Proposer:     sdk.AccAddress("test_proposer_12345678901234567890").String(),
		Title:        "Test Proposal",
		Description:  "This is a test proposal",
		Status:       governancev1.ProposalStatus_PROPOSAL_STATUS_VOTING,
		SubmitTime:   timestamppb.Now(),
		YesVotes:     "0",
		NoVotes:      "0",
		AbstainVotes: "0",
		TotalVotes:   "0",
	}
	err := suite.keeper.SetProposal(suite.ctx, proposal)
	require.NoError(suite.T(), err)

	// Create vote
	voter := sdk.AccAddress("test_voter_12345678901234567890").String()
	vote := &governancev1.Vote{
		ProposalId:  1,
		Voter:       voter,
		Option:      governancev1.VoteOption_VOTE_OPTION_YES,
		VotingPower: "5000000",
		VoteTime:    timestamppb.Now(),
	}
	err = suite.keeper.SetVote(suite.ctx, vote)
	require.NoError(suite.T(), err)

	// Query vote
	req := &governancev1.QueryVoteRequest{
		ProposalId: 1,
		Voter:      voter,
	}
	resp, err := suite.queryServer.Vote(suite.ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotNil(suite.T(), resp.Vote)
	require.Equal(suite.T(), uint64(1), resp.Vote.ProposalId)
	require.Equal(suite.T(), voter, resp.Vote.Voter)
	require.Equal(suite.T(), governancev1.VoteOption_VOTE_OPTION_YES, resp.Vote.Option)
}

func (suite *QueryServerTestSuite) TestQueryVote_NotFound() {
	req := &governancev1.QueryVoteRequest{
		ProposalId: 1,
		Voter:      sdk.AccAddress("test_voter_12345678901234567890").String(),
	}
	_, err := suite.queryServer.Vote(suite.ctx, req)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "not found")
}

// Test QueryVotes
func (suite *QueryServerTestSuite) TestQueryVotes() {
	// Create proposal
	proposal := &governancev1.Proposal{
		ProposalId:   1,
		Proposer:     sdk.AccAddress("test_proposer_12345678901234567890").String(),
		Title:        "Test Proposal",
		Description:  "This is a test proposal",
		Status:       governancev1.ProposalStatus_PROPOSAL_STATUS_VOTING,
		SubmitTime:   timestamppb.Now(),
		YesVotes:     "0",
		NoVotes:      "0",
		AbstainVotes: "0",
		TotalVotes:   "0",
	}
	err := suite.keeper.SetProposal(suite.ctx, proposal)
	require.NoError(suite.T(), err)

	// Create multiple votes
	voter1 := sdk.AccAddress("test_voter1_12345678901234567890").String()
	voter2 := sdk.AccAddress("test_voter2_12345678901234567890").String()
	
	vote1 := &governancev1.Vote{
		ProposalId:  1,
		Voter:       voter1,
		Option:      governancev1.VoteOption_VOTE_OPTION_YES,
		VotingPower: "5000000",
		VoteTime:    timestamppb.Now(),
	}
	vote2 := &governancev1.Vote{
		ProposalId:  1,
		Voter:       voter2,
		Option:      governancev1.VoteOption_VOTE_OPTION_NO,
		VotingPower: "3000000",
		VoteTime:    timestamppb.Now(),
	}
	err = suite.keeper.SetVote(suite.ctx, vote1)
	require.NoError(suite.T(), err)
	err = suite.keeper.SetVote(suite.ctx, vote2)
	require.NoError(suite.T(), err)

	// Query votes
	req := &governancev1.QueryVotesRequest{
		ProposalId: 1,
	}
	resp, err := suite.queryServer.Votes(suite.ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.Len(suite.T(), resp.Votes, 2, "Should have 2 votes")
}

// Test QueryParams
func (suite *QueryServerTestSuite) TestQueryParams() {
	req := &governancev1.QueryParamsRequest{}
	resp, err := suite.queryServer.Params(suite.ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotNil(suite.T(), resp.Params)
	require.NotEmpty(suite.T(), resp.Params.MinDeposit)
	require.NotEmpty(suite.T(), resp.Params.Quorum)
	require.NotEmpty(suite.T(), resp.Params.Threshold)
	require.NotNil(suite.T(), resp.Params.VotingPeriod)
	require.NotNil(suite.T(), resp.Params.TimelockPeriod)
}

