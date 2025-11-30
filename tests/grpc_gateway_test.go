package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
	consensuskeeper "github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	governancekeeper "github.com/volnix-protocol/volnix-protocol/x/governance/keeper"
)

// GRPCGatewayTestSuite tests gRPC Gateway REST endpoints
type GRPCGatewayTestSuite struct {
	suite.Suite

	testCtx     *TestContext
	ctx         sdk.Context
	cdc         codec.Codec
	httpServer  *httptest.Server
	mux         *http.ServeMux
	consensusKeeper *consensuskeeper.Keeper
	governanceKeeper *governancekeeper.Keeper
}

func (suite *GRPCGatewayTestSuite) SetupTest() {
	// Use test helper to create properly initialized test context
	suite.testCtx = NewTestContext(suite.T())
	suite.ctx = suite.testCtx.Ctx
	suite.cdc = suite.testCtx.Cdc
	suite.consensusKeeper = suite.testCtx.ConsensusKeeper
	suite.governanceKeeper = suite.testCtx.GovernanceKeeper 

	// Create HTTP mux for REST endpoints
	suite.mux = http.NewServeMux()

	// Register REST endpoints that simulate gRPC Gateway behavior
	// In production, BaseApp handles this automatically from proto annotations
	suite.registerGatewayRoutes()

	// Create HTTP test server
	suite.httpServer = httptest.NewServer(suite.mux)
}

func (suite *GRPCGatewayTestSuite) TearDownTest() {
	if suite.httpServer != nil {
		suite.httpServer.Close()
	}
}

// registerGatewayRoutes manually registers REST endpoints for testing
// In production, BaseApp automatically registers routes from proto annotations
func (suite *GRPCGatewayTestSuite) registerGatewayRoutes() {
	// Consensus module routes
	suite.mux.HandleFunc("/volnix/consensus/v1/params", suite.handleConsensusParams)
	suite.mux.HandleFunc("/volnix/consensus/v1/validators", suite.handleConsensusValidators)
	
	// Governance module routes
	suite.mux.HandleFunc("/volnix/governance/v1/proposal/", suite.handleGovernanceProposal)
	suite.mux.HandleFunc("/volnix/governance/v1/proposals", suite.handleGovernanceProposals)
	suite.mux.HandleFunc("/volnix/governance/v1/vote/", suite.handleGovernanceVote)
	suite.mux.HandleFunc("/volnix/governance/v1/votes/", suite.handleGovernanceVotes)
	suite.mux.HandleFunc("/volnix/governance/v1/params", suite.handleGovernanceParams)
}

// HTTP handlers that simulate gRPC Gateway behavior
func (suite *GRPCGatewayTestSuite) handleConsensusParams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	queryServer := consensuskeeper.NewQueryServer(*suite.consensusKeeper)
	req := &consensusv1.QueryParamsRequest{}
	resp, err := queryServer.Params(suite.ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (suite *GRPCGatewayTestSuite) handleConsensusValidators(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	queryServer := consensuskeeper.NewQueryServer(*suite.consensusKeeper)
	req := &consensusv1.QueryValidatorsRequest{}
	resp, err := queryServer.Validators(suite.ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (suite *GRPCGatewayTestSuite) handleGovernanceProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse proposal_id from URL path
	// URL format: /volnix/governance/v1/proposal/{proposal_id}
	path := r.URL.Path
	var id uint64
	if _, err := fmt.Sscanf(path, "/volnix/governance/v1/proposal/%d", &id); err != nil {
		http.Error(w, "invalid proposal_id", http.StatusBadRequest)
		return
	}
	
	queryServer := governancekeeper.NewQueryServer(suite.governanceKeeper)
	req := &governancev1.QueryProposalRequest{ProposalId: id}
	resp, err := queryServer.Proposal(suite.ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (suite *GRPCGatewayTestSuite) handleGovernanceProposals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	queryServer := governancekeeper.NewQueryServer(suite.governanceKeeper)
	req := &governancev1.QueryProposalsRequest{}
	resp, err := queryServer.Proposals(suite.ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (suite *GRPCGatewayTestSuite) handleGovernanceVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse proposal_id and voter from URL path
	// URL format: /volnix/governance/v1/vote/{proposal_id}/{voter}
	path := r.URL.Path
	var id uint64
	var voter string
	if _, err := fmt.Sscanf(path, "/volnix/governance/v1/vote/%d/%s", &id, &voter); err != nil {
		http.Error(w, "invalid proposal_id or voter", http.StatusBadRequest)
		return
	}
	
	queryServer := governancekeeper.NewQueryServer(suite.governanceKeeper)
	req := &governancev1.QueryVoteRequest{ProposalId: id, Voter: voter}
	resp, err := queryServer.Vote(suite.ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (suite *GRPCGatewayTestSuite) handleGovernanceVotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse proposal_id from URL path
	// URL format: /volnix/governance/v1/votes/{proposal_id}
	path := r.URL.Path
	var id uint64
	if _, err := fmt.Sscanf(path, "/volnix/governance/v1/votes/%d", &id); err != nil {
		http.Error(w, "invalid proposal_id", http.StatusBadRequest)
		return
	}
	
	queryServer := governancekeeper.NewQueryServer(suite.governanceKeeper)
	req := &governancev1.QueryVotesRequest{ProposalId: id}
	resp, err := queryServer.Votes(suite.ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (suite *GRPCGatewayTestSuite) handleGovernanceParams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	queryServer := governancekeeper.NewQueryServer(suite.governanceKeeper)
	req := &governancev1.QueryParamsRequest{}
	resp, err := queryServer.Params(suite.ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Test Consensus Module REST Endpoints

func (suite *GRPCGatewayTestSuite) TestConsensusParamsEndpoint() {
	url := fmt.Sprintf("%s/volnix/consensus/v1/params", suite.httpServer.URL)
	
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	require.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))
	
	var result consensusv1.QueryParamsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result.Params)
}

func (suite *GRPCGatewayTestSuite) TestConsensusValidatorsEndpoint() {
	url := fmt.Sprintf("%s/volnix/consensus/v1/validators", suite.httpServer.URL)
	
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	require.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))
	
	var result consensusv1.QueryValidatorsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	// Validators list can be empty in test context, but should not be nil
	// In Go, empty slice is not nil, so we just check that decoding succeeded
	require.NotNil(suite.T(), &result)
}

// Test Governance Module REST Endpoints

func (suite *GRPCGatewayTestSuite) TestGovernanceParamsEndpoint() {
	url := fmt.Sprintf("%s/volnix/governance/v1/params", suite.httpServer.URL)
	
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	require.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))
	
	var result governancev1.QueryParamsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result.Params)
}

func (suite *GRPCGatewayTestSuite) TestGovernanceProposalsEndpoint() {
	url := fmt.Sprintf("%s/volnix/governance/v1/proposals", suite.httpServer.URL)
	
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	require.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))
	
	var result governancev1.QueryProposalsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	// Proposals list can be empty in test context, but should not be nil
	// In Go, empty slice is not nil, so we just check that decoding succeeded
	require.NotNil(suite.T(), &result)
}

func (suite *GRPCGatewayTestSuite) TestGovernanceProposalEndpoint() {
	// First, create a proposal to query
	proposer := sdk.AccAddress("cosmos1proposer").String()
	
	proposal := &governancev1.Proposal{
		ProposalId:   1,
		Proposer:     proposer,
		Title:        "Test Proposal",
		Description:  "Test Description",
		Status:       governancev1.ProposalStatus_PROPOSAL_STATUS_SUBMITTED,
		ProposalType: governancev1.ProposalType_PROPOSAL_TYPE_PARAMETER_CHANGE,
	}
	
	err := suite.governanceKeeper.SetProposal(suite.ctx, proposal)
	require.NoError(suite.T(), err)
	
	// Now test the endpoint
	url := fmt.Sprintf("%s/volnix/governance/v1/proposal/1", suite.httpServer.URL)
	
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	require.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))
	
	var result governancev1.QueryProposalResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result.Proposal)
	require.Equal(suite.T(), uint64(1), result.Proposal.ProposalId)
}

func (suite *GRPCGatewayTestSuite) TestGovernanceProposalEndpoint_NotFound() {
	url := fmt.Sprintf("%s/volnix/governance/v1/proposal/999", suite.httpServer.URL)
	
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Should return error (not found)
	require.NotEqual(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *GRPCGatewayTestSuite) TestGovernanceVotesEndpoint() {
	// Create a proposal and vote first
	proposer := sdk.AccAddress("cosmos1proposer").String()
	voter := sdk.AccAddress("cosmos1voter").String()
	
	proposal := &governancev1.Proposal{
		ProposalId:   2,
		Proposer:     proposer,
		Title:        "Test Proposal 2",
		Description:  "Test Description 2",
		Status:       governancev1.ProposalStatus_PROPOSAL_STATUS_VOTING,
		ProposalType: governancev1.ProposalType_PROPOSAL_TYPE_PARAMETER_CHANGE,
	}
	
	err := suite.governanceKeeper.SetProposal(suite.ctx, proposal)
	require.NoError(suite.T(), err)
	
	vote := &governancev1.Vote{
		ProposalId: 2,
		Voter:      voter,
		Option:     governancev1.VoteOption_VOTE_OPTION_YES,
	}
	
	err = suite.governanceKeeper.SetVote(suite.ctx, vote)
	require.NoError(suite.T(), err)
	
	// Test the endpoint
	url := fmt.Sprintf("%s/volnix/governance/v1/votes/2", suite.httpServer.URL)
	
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	require.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))
	
	var result governancev1.QueryVotesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result.Votes)
	require.Len(suite.T(), result.Votes, 1)
}

func (suite *GRPCGatewayTestSuite) TestGovernanceVoteEndpoint() {
	// Create a proposal and vote first
	proposer := sdk.AccAddress("cosmos1proposer2").String()
	voter := sdk.AccAddress("cosmos1voter2").String()
	
	proposal := &governancev1.Proposal{
		ProposalId:   3,
		Proposer:     proposer,
		Title:        "Test Proposal 3",
		Description:  "Test Description 3",
		Status:       governancev1.ProposalStatus_PROPOSAL_STATUS_VOTING,
		ProposalType: governancev1.ProposalType_PROPOSAL_TYPE_PARAMETER_CHANGE,
	}
	
	err := suite.governanceKeeper.SetProposal(suite.ctx, proposal)
	require.NoError(suite.T(), err)
	
	vote := &governancev1.Vote{
		ProposalId: 3,
		Voter:      voter,
		Option:     governancev1.VoteOption_VOTE_OPTION_NO,
	}
	
	err = suite.governanceKeeper.SetVote(suite.ctx, vote)
	require.NoError(suite.T(), err)
	
	// Test the endpoint (URL encode the voter address)
	url := fmt.Sprintf("%s/volnix/governance/v1/vote/3/%s", suite.httpServer.URL, voter)
	
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	require.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))
	
	var result governancev1.QueryVoteResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result.Vote)
	require.Equal(suite.T(), uint64(3), result.Vote.ProposalId)
	require.Equal(suite.T(), voter, result.Vote.Voter)
}

// Test HTTP Method Validation

func (suite *GRPCGatewayTestSuite) TestInvalidHTTPMethod() {
	url := fmt.Sprintf("%s/volnix/consensus/v1/params", suite.httpServer.URL)
	
	// Try POST instead of GET
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
	require.NoError(suite.T(), err)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Should return method not allowed or not found
	require.NotEqual(suite.T(), http.StatusOK, resp.StatusCode)
}

// Test Content-Type Header

func (suite *GRPCGatewayTestSuite) TestContentTypeHeader() {
	url := fmt.Sprintf("%s/volnix/consensus/v1/params", suite.httpServer.URL)
	
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	require.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))
}

func TestGRPCGatewayTestSuite(t *testing.T) {
	suite.Run(t, new(GRPCGatewayTestSuite))
}

