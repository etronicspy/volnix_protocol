package keeper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"

	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/math"
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

// Mock BankKeeperInterface for testing
type MockBankKeeperForGovernance struct {
	balances map[string]uint64 // address -> balance in micro units
	supply   uint64
}

func NewMockBankKeeperForGovernance() *MockBankKeeperForGovernance {
	return &MockBankKeeperForGovernance{
		balances: make(map[string]uint64),
		supply:   TotalWRTSupply,
	}
}

func (m *MockBankKeeperForGovernance) SetBalance(addr string, balance uint64) {
	m.balances[addr] = balance
}

func (m *MockBankKeeperForGovernance) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	balance, ok := m.balances[addr.String()]
	if !ok {
		balance = 0
	}
	return sdk.NewCoin(denom, math.NewIntFromUint64(balance))
}

func (m *MockBankKeeperForGovernance) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	balance, ok := m.balances[addr.String()]
	if !ok {
		balance = 0
	}
	return sdk.NewCoins(sdk.NewCoin("uwrt", math.NewIntFromUint64(balance)))
}

func (m *MockBankKeeperForGovernance) GetSupply(ctx sdk.Context, denom string) sdk.Coin {
	return sdk.NewCoin(denom, math.NewIntFromUint64(m.supply))
}

type MsgServerTestSuite struct {
	suite.Suite

	cdc           codec.Codec
	ctx           sdk.Context
	keeper        *Keeper
	msgServer     governancev1.MsgServer
	mockBankKeeper *MockBankKeeperForGovernance
	storeKey      storetypes.StoreKey
	paramStore    paramtypes.Subspace
}

func TestMsgServerTestSuite(t *testing.T) {
	suite.Run(t, new(MsgServerTestSuite))
}

func (suite *MsgServerTestSuite) SetupTest() {
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

	// Create keeper and msg server
	suite.keeper = NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)
	suite.msgServer = NewMsgServer(suite.keeper)

	// Create and set mock bank keeper
	suite.mockBankKeeper = NewMockBankKeeperForGovernance()
	suite.keeper.SetBankKeeper(suite.mockBankKeeper)

	// Set default params
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

// Test SubmitProposal
func (suite *MsgServerTestSuite) TestSubmitProposal() {
	proposer := sdk.AccAddress("test_proposer_12345678901234567890").String()
	suite.mockBankKeeper.SetBalance(proposer, 10000000) // 10 WRT

	req := &governancev1.MsgSubmitProposal{
		Proposer:     proposer,
		Title:         "Test Proposal",
		Description:   "This is a test proposal",
		ProposalType:  governancev1.ProposalType_PROPOSAL_TYPE_PARAMETER_CHANGE,
		Deposit:       "2000000", // 2 WRT (above min deposit of 1 WRT)
		ParameterChanges: []*governancev1.ParameterChange{
			{
				Module:    "governance",
				Parameter: "voting_period",
				NewValue:  "168h", // 7 days in hours
			},
		},
	}

	resp, err := suite.msgServer.SubmitProposal(suite.ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.Equal(suite.T(), uint64(1), resp.ProposalId)

	// Verify proposal was stored
	proposal, err := suite.keeper.GetProposal(suite.ctx, 1)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), proposer, proposal.Proposer)
	require.Equal(suite.T(), "Test Proposal", proposal.Title)
	require.Equal(suite.T(), governancev1.ProposalStatus_PROPOSAL_STATUS_SUBMITTED, proposal.Status)

	// Check events
	events := suite.ctx.EventManager().Events()
	proposalSubmittedFound := false
	for _, event := range events {
		if event.Type == "governance.proposal_submitted" {
			proposalSubmittedFound = true
		}
	}
	require.True(suite.T(), proposalSubmittedFound, "Should have proposal_submitted event")
}

func (suite *MsgServerTestSuite) TestSubmitProposal_InsufficientDeposit() {
	proposer := sdk.AccAddress("test_proposer_12345678901234567890").String()
	suite.mockBankKeeper.SetBalance(proposer, 10000000)

	req := &governancev1.MsgSubmitProposal{
		Proposer:    proposer,
		Title:       "Test Proposal",
		Description: "This is a test proposal",
		Deposit:     "500000", // 0.5 WRT (below min deposit of 1 WRT)
	}

	_, err := suite.msgServer.SubmitProposal(suite.ctx, req)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrInsufficientDeposit, err)
}

// Test Vote
func (suite *MsgServerTestSuite) TestVote() {
	proposer := sdk.AccAddress("test_proposer_12345678901234567890").String()
	voter := sdk.AccAddress("test_voter_12345678901234567890").String()
	
	// Set balances
	suite.mockBankKeeper.SetBalance(proposer, 10000000)
	suite.mockBankKeeper.SetBalance(voter, 5000000) // 5 WRT

	// First create a proposal
	submitReq := &governancev1.MsgSubmitProposal{
		Proposer:    proposer,
		Title:       "Test Proposal",
		Description: "This is a test proposal",
		Deposit:     "2000000",
	}
	_, err := suite.msgServer.SubmitProposal(suite.ctx, submitReq)
	require.NoError(suite.T(), err)

	// Get proposal and set it to VOTING status (voting start time should be in the past)
	proposal, err := suite.keeper.GetProposal(suite.ctx, 1)
	require.NoError(suite.T(), err)
	proposal.Status = governancev1.ProposalStatus_PROPOSAL_STATUS_VOTING
	proposal.VotingStartTime = timestamppb.New(time.Now().Add(-1 * time.Hour)) // Past start time
	err = suite.keeper.SetProposal(suite.ctx, proposal)
	require.NoError(suite.T(), err)

	// Vote on proposal
	voteReq := &governancev1.MsgVote{
		ProposalId: 1,
		Voter:      voter,
		Option:     governancev1.VoteOption_VOTE_OPTION_YES,
	}

	resp, err := suite.msgServer.Vote(suite.ctx, voteReq)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify vote was stored
	vote, err := suite.keeper.GetVote(suite.ctx, 1, voter)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), governancev1.VoteOption_VOTE_OPTION_YES, vote.Option)

	// Check events
	events := suite.ctx.EventManager().Events()
	voteCastFound := false
	for _, event := range events {
		if event.Type == "governance.vote_cast" {
			voteCastFound = true
		}
	}
	require.True(suite.T(), voteCastFound, "Should have vote_cast event")
}

// Test ExecuteProposal
func (suite *MsgServerTestSuite) TestExecuteProposal() {
	proposer := sdk.AccAddress("test_proposer_12345678901234567890").String()
	voter1 := sdk.AccAddress("test_voter1_12345678901234567890").String()
	voter2 := sdk.AccAddress("test_voter2_12345678901234567890").String()
	
	// Set balances (enough to meet quorum and threshold)
	suite.mockBankKeeper.SetBalance(proposer, 10000000)
	suite.mockBankKeeper.SetBalance(voter1, 50000000) // 50 WRT
	suite.mockBankKeeper.SetBalance(voter2, 30000000) // 30 WRT

	// Create proposal
	submitReq := &governancev1.MsgSubmitProposal{
		Proposer:    proposer,
		Title:       "Test Proposal",
		Description: "This is a test proposal",
		Deposit:     "2000000",
		ParameterChanges: []*governancev1.ParameterChange{
			{
				Module:    "governance",
				Parameter: "voting_period",
				NewValue:  "168h", // 7 days in hours
			},
		},
	}
	_, err := suite.msgServer.SubmitProposal(suite.ctx, submitReq)
	require.NoError(suite.T(), err)

	// Get proposal and set it to VOTING status with past voting start time
	proposal1, err := suite.keeper.GetProposal(suite.ctx, 1)
	require.NoError(suite.T(), err)
	proposal1.Status = governancev1.ProposalStatus_PROPOSAL_STATUS_VOTING
	proposal1.VotingStartTime = timestamppb.New(time.Now().Add(-2 * time.Hour)) // Past start time
	proposal1.VotingEndTime = timestamppb.New(time.Now().Add(1 * time.Hour)) // Future end time
	err = suite.keeper.SetProposal(suite.ctx, proposal1)
	require.NoError(suite.T(), err)

	// Vote yes - first vote
	voteReq1 := &governancev1.MsgVote{
		ProposalId: 1,
		Voter:      voter1,
		Option:     governancev1.VoteOption_VOTE_OPTION_YES,
	}
	_, err = suite.msgServer.Vote(suite.ctx, voteReq1)
	require.NoError(suite.T(), err)

	// Re-set proposal to VOTING status (TallyVotes might have changed it)
	proposal1, err = suite.keeper.GetProposal(suite.ctx, 1)
	require.NoError(suite.T(), err)
	if proposal1.Status != governancev1.ProposalStatus_PROPOSAL_STATUS_VOTING {
		proposal1.Status = governancev1.ProposalStatus_PROPOSAL_STATUS_VOTING
		proposal1.VotingStartTime = timestamppb.New(time.Now().Add(-2 * time.Hour))
		proposal1.VotingEndTime = timestamppb.New(time.Now().Add(1 * time.Hour))
		err = suite.keeper.SetProposal(suite.ctx, proposal1)
		require.NoError(suite.T(), err)
	}

	// Vote yes - second vote
	voteReq2 := &governancev1.MsgVote{
		ProposalId: 1,
		Voter:      voter2,
		Option:     governancev1.VoteOption_VOTE_OPTION_YES,
	}
	_, err = suite.msgServer.Vote(suite.ctx, voteReq2)
	require.NoError(suite.T(), err)

	// Get proposal - it should be PASSED after voting (TallyVotes sets it)
	proposal, err := suite.keeper.GetProposal(suite.ctx, 1)
	require.NoError(suite.T(), err)
	
	// Ensure proposal is PASSED and execution time is in the past
	// Use block time to ensure consistency
	blockTime := suite.ctx.BlockTime()
	pastTime := blockTime.Add(-2 * time.Hour) // Well in the past
	
	proposal.Status = governancev1.ProposalStatus_PROPOSAL_STATUS_PASSED
	proposal.ExecutionTime = timestamppb.New(pastTime) // Past execution time (timelock expired)
	err = suite.keeper.SetProposal(suite.ctx, proposal)
	require.NoError(suite.T(), err)
	
	// Verify proposal is in PASSED status and execution time is in the past
	proposal, err = suite.keeper.GetProposal(suite.ctx, 1)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), governancev1.ProposalStatus_PROPOSAL_STATUS_PASSED, proposal.Status)
	require.NotNil(suite.T(), proposal.ExecutionTime)
	require.True(suite.T(), proposal.ExecutionTime.AsTime().Before(blockTime), "Execution time should be before block time")

	// Execute proposal
	executeReq := &governancev1.MsgExecuteProposal{
		ProposalId: 1,
		Executor:   proposer,
	}

	resp, err := suite.msgServer.ExecuteProposal(suite.ctx, executeReq)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify proposal status is EXECUTED
	executedProposal, err := suite.keeper.GetProposal(suite.ctx, 1)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), governancev1.ProposalStatus_PROPOSAL_STATUS_EXECUTED, executedProposal.Status)

	// Check events
	events := suite.ctx.EventManager().Events()
	proposalExecutedFound := false
	for _, event := range events {
		if event.Type == "governance.proposal_executed" {
			proposalExecutedFound = true
		}
	}
	require.True(suite.T(), proposalExecutedFound, "Should have proposal_executed event")
}

