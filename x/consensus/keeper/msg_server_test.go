package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

type MsgServerTestSuite struct {
	suite.Suite

	cdc        codec.Codec
	ctx        sdk.Context
	keeper     *Keeper
	msgServer  consensusv1.MsgServer
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
}

func (suite *MsgServerTestSuite) SetupTest() {
	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)

	// Create store keys
	suite.storeKey = storetypes.NewKVStoreKey("test_consensus")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")

	// Create test context
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)

	// Create params keeper and subspace
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore.WithKeyTable(types.ParamKeyTable())

	// Create keeper and msg server
	suite.keeper = NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)
	suite.msgServer = NewMsgServer(*suite.keeper)

	// Set default params
	suite.keeper.SetParams(suite.ctx, *types.DefaultParams())
}

func (suite *MsgServerTestSuite) TestUpdateConsensusState() {
	// Test valid consensus state update
	msg := &consensusv1.MsgUpdateConsensusState{
		Authority:        "cosmos1test",
		CurrentHeight:    1000,
		TotalAntBurned:   "1000000",
		ActiveValidators: []string{"cosmos1test1", "cosmos2test2"},
	}

	resp, err := suite.msgServer.UpdateConsensusState(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify consensus state was updated
	state, err := suite.keeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(1000), state.CurrentHeight)
	require.Equal(suite.T(), "1000000", state.TotalAntBurned)
	require.Len(suite.T(), state.ActiveValidators, 2)

	// Test invalid authority
	invalidMsg := &consensusv1.MsgUpdateConsensusState{
		Authority:        "cosmos2test",
		CurrentHeight:    1001,
		TotalAntBurned:   "1000001",
		ActiveValidators: []string{"cosmos1test1"},
	}

	_, err = suite.msgServer.UpdateConsensusState(suite.ctx, invalidMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrUnauthorized, err)
}

func (suite *MsgServerTestSuite) TestSetValidatorWeight() {
	// Test valid validator weight setting
	msg := &consensusv1.MsgSetValidatorWeight{
		Authority: "cosmos1test",
		Validator: "cosmos1validator",
		Weight:    "1000000",
	}

	resp, err := suite.msgServer.SetValidatorWeight(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify validator weight was set
	weight, err := suite.keeper.GetValidatorWeight(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "1000000", weight)

	// Test invalid authority
	invalidMsg := &consensusv1.MsgSetValidatorWeight{
		Authority: "cosmos2test",
		Validator: "cosmos2validator",
		Weight:    "1000000",
	}

	_, err = suite.msgServer.SetValidatorWeight(suite.ctx, invalidMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrUnauthorized, err)

	// Test empty validator address
	emptyValidatorMsg := &consensusv1.MsgSetValidatorWeight{
		Authority: "cosmos1test",
		Validator: "",
		Weight:    "1000000",
	}

	_, err = suite.msgServer.SetValidatorWeight(suite.ctx, emptyValidatorMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidatorAddress, err)
}

func (suite *MsgServerTestSuite) TestProcessHalving() {
	// Test processing halving
	msg := &consensusv1.MsgProcessHalving{
		Authority: "cosmos1test",
	}

	resp, err := suite.msgServer.ProcessHalving(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)

	// Verify halving was processed
	halvingInfo, err := suite.keeper.GetHalvingInfo(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), halvingInfo)

	// Test invalid authority
	invalidMsg := &consensusv1.MsgProcessHalving{
		Authority: "cosmos2test",
	}

	_, err = suite.msgServer.ProcessHalving(suite.ctx, invalidMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrUnauthorized, err)
}

func (suite *MsgServerTestSuite) TestSelectBlockProducer() {
	// Test selecting block producer
	msg := &consensusv1.MsgSelectBlockProducer{
		Authority:  "cosmos1test",
		Validators: []string{"cosmos1test1", "cosmos2test2", "cosmos3test3"},
	}

	resp, err := suite.msgServer.SelectBlockProducer(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotEmpty(suite.T(), resp.Producer)
	require.Contains(suite.T(), msg.Validators, resp.Producer)

	// Test invalid authority
	invalidMsg := &consensusv1.MsgSelectBlockProducer{
		Authority:  "cosmos2test",
		Validators: []string{"cosmos1test1", "cosmos2test2"},
	}

	_, err = suite.msgServer.SelectBlockProducer(suite.ctx, invalidMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrUnauthorized, err)

	// Test empty validators list
	emptyValidatorsMsg := &consensusv1.MsgSelectBlockProducer{
		Authority:  "cosmos1test",
		Validators: []string{},
	}

	_, err = suite.msgServer.SelectBlockProducer(suite.ctx, emptyValidatorsMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrNoValidators, err)
}

func (suite *MsgServerTestSuite) TestCalculateBlockTime() {
	// Test calculating block time
	msg := &consensusv1.MsgCalculateBlockTime{
		Authority: "cosmos1test",
		AntAmount: "1000000",
	}

	resp, err := suite.msgServer.CalculateBlockTime(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotZero(suite.T(), resp.BlockTime)

	// Test invalid authority
	invalidMsg := &consensusv1.MsgCalculateBlockTime{
		Authority: "cosmos2test",
		AntAmount: "1000000",
	}

	_, err = suite.msgServer.CalculateBlockTime(suite.ctx, invalidMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrUnauthorized, err)

	// Test zero ant amount
	zeroAntMsg := &consensusv1.MsgCalculateBlockTime{
		Authority: "cosmos1test",
		AntAmount: "0",
	}

	_, err = suite.msgServer.CalculateBlockTime(suite.ctx, zeroAntMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrInvalidAntAmount, err)
}

func (suite *MsgServerTestSuite) TestSelectBlockCreator() {
	// Add a validator first
	validator := &consensusv1.Validator{
		Validator:     "cosmos1validator",
		AntBalance:    "1000000",
		ActivityScore: "500000",
	}
	suite.keeper.SetValidator(suite.ctx, validator)

	// Test selecting block creator
	msg := &consensusv1.MsgSelectBlockCreator{}

	resp, err := suite.msgServer.SelectBlockCreator(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotEmpty(suite.T(), resp.SelectedValidator)
	require.Equal(suite.T(), "cosmos1validator", resp.SelectedValidator)

	// Verify block creator was set (use next height which is current + 1)
	nextHeight := uint64(suite.ctx.BlockHeight() + 1)
	blockCreator, err := suite.keeper.GetBlockCreator(suite.ctx, nextHeight)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), blockCreator)
	require.Equal(suite.T(), "cosmos1validator", blockCreator.Validator)
}

func (suite *MsgServerTestSuite) TestCommitBid() {
	validator := "cosmos1validator"
	nonce := "test_nonce_12345"
	bidAmount := "1000000"
	height := uint64(1000)

	// Create commit hash
	commitHash := HashCommit(nonce, bidAmount)
	require.NotEmpty(suite.T(), commitHash)
	require.Len(suite.T(), commitHash, 64) // SHA256 produces 64 hex chars

	// Test valid commit bid
	msg := &consensusv1.MsgCommitBid{
		Validator:   validator,
		CommitHash:  commitHash,
		BlockHeight: height,
	}

	resp, err := suite.msgServer.CommitBid(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.True(suite.T(), resp.Success)

	// Verify auction was created and commit was added
	auction, err := suite.keeper.GetBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), auction)
	require.Len(suite.T(), auction.Commits, 1)
	require.Equal(suite.T(), validator, auction.Commits[0].Validator)
	require.Equal(suite.T(), commitHash, auction.Commits[0].CommitHash)

	// Test with nil request
	_, err = suite.msgServer.CommitBid(suite.ctx, nil)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "cannot be nil")

	// Test with empty validator
	emptyValidatorMsg := &consensusv1.MsgCommitBid{
		Validator:   "",
		CommitHash:  commitHash,
		BlockHeight: height,
	}
	_, err = suite.msgServer.CommitBid(suite.ctx, emptyValidatorMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidatorAddress, err)

	// Test with empty commit hash
	emptyHashMsg := &consensusv1.MsgCommitBid{
		Validator:   validator,
		CommitHash:  "",
		BlockHeight: height,
	}
	_, err = suite.msgServer.CommitBid(suite.ctx, emptyHashMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrInvalidCommitHash, err)

	// Test with invalid commit hash length
	shortHashMsg := &consensusv1.MsgCommitBid{
		Validator:   validator,
		CommitHash:  "short",
		BlockHeight: height,
	}
	_, err = suite.msgServer.CommitBid(suite.ctx, shortHashMsg)
	require.Error(suite.T(), err)

	// Test duplicate commit (same validator)
	_, err = suite.msgServer.CommitBid(suite.ctx, msg)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "already committed")

	// Test with zero height (should use current block height)
	zeroHeightMsg := &consensusv1.MsgCommitBid{
		Validator:   "cosmos1validator2",
		CommitHash:  HashCommit("nonce2", "2000000"),
		BlockHeight: 0, // Will use current block height
	}
	resp, err = suite.msgServer.CommitBid(suite.ctx, zeroHeightMsg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.True(suite.T(), resp.Success)
}

func (suite *MsgServerTestSuite) TestRevealBid() {
	validator := "cosmos1validator"
	nonce := "test_nonce_12345"
	bidAmount := "1000000"
	height := uint64(1000)

	// First, commit a bid
	commitHash := HashCommit(nonce, bidAmount)
	commitMsg := &consensusv1.MsgCommitBid{
		Validator:   validator,
		CommitHash:  commitHash,
		BlockHeight: height,
	}
	_, err := suite.msgServer.CommitBid(suite.ctx, commitMsg)
	require.NoError(suite.T(), err)

	// Transition auction to reveal phase
	err = suite.keeper.TransitionAuctionPhase(suite.ctx, height)
	require.NoError(suite.T(), err)

	// Test valid reveal bid
	msg := &consensusv1.MsgRevealBid{
		Validator:   validator,
		Nonce:       nonce,
		BidAmount:   bidAmount,
		BlockHeight: height,
	}

	resp, err := suite.msgServer.RevealBid(suite.ctx, msg)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.True(suite.T(), resp.Success)

	// Verify reveal was added
	auction, err := suite.keeper.GetBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), auction)
	require.Len(suite.T(), auction.Reveals, 1)
	require.Equal(suite.T(), validator, auction.Reveals[0].Validator)
	require.Equal(suite.T(), bidAmount, auction.Reveals[0].BidAmount)
	require.Equal(suite.T(), nonce, auction.Reveals[0].Nonce)

	// Test with nil request
	_, err = suite.msgServer.RevealBid(suite.ctx, nil)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "cannot be nil")

	// Test with empty validator
	emptyValidatorMsg := &consensusv1.MsgRevealBid{
		Validator:   "",
		Nonce:       nonce,
		BidAmount:   bidAmount,
		BlockHeight: height,
	}
	_, err = suite.msgServer.RevealBid(suite.ctx, emptyValidatorMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrEmptyValidatorAddress, err)

	// Test with empty nonce
	emptyNonceMsg := &consensusv1.MsgRevealBid{
		Validator:   validator,
		Nonce:       "",
		BidAmount:   bidAmount,
		BlockHeight: height,
	}
	_, err = suite.msgServer.RevealBid(suite.ctx, emptyNonceMsg)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "nonce cannot be empty")

	// Test with empty bid amount
	emptyBidMsg := &consensusv1.MsgRevealBid{
		Validator:   validator,
		Nonce:       nonce,
		BidAmount:   "",
		BlockHeight: height,
	}
	_, err = suite.msgServer.RevealBid(suite.ctx, emptyBidMsg)
	require.Error(suite.T(), err)
	require.Equal(suite.T(), types.ErrInvalidBidAmount, err)

	// Test with wrong nonce (commit hash mismatch)
	// Create a new auction for this test
	height2 := uint64(2000)
	commitHashWrong := HashCommit("nonce2", "2000000")
	commitMsgWrong := &consensusv1.MsgCommitBid{
		Validator:   "cosmos1validator2",
		CommitHash:  commitHashWrong,
		BlockHeight: height2,
	}
	_, err2 := suite.msgServer.CommitBid(suite.ctx, commitMsgWrong)
	require.NoError(suite.T(), err2)

	err2 = suite.keeper.TransitionAuctionPhase(suite.ctx, height2)
	require.NoError(suite.T(), err2)

	wrongNonceMsg := &consensusv1.MsgRevealBid{
		Validator:   "cosmos1validator2",
		Nonce:       "wrong_nonce",
		BidAmount:   "2000000",
		BlockHeight: height2,
	}
	_, err2 = suite.msgServer.RevealBid(suite.ctx, wrongNonceMsg)
	require.Error(suite.T(), err2)
	require.Equal(suite.T(), types.ErrCommitHashMismatch, err2)

	// Test with zero height (should use current block height)
	// Create new auction for current height
	currentHeight := uint64(suite.ctx.BlockHeight())
	commitHash3 := HashCommit("nonce3", "3000000")
	commitMsg3 := &consensusv1.MsgCommitBid{
		Validator:   "cosmos1validator3",
		CommitHash:  commitHash3,
		BlockHeight: currentHeight,
	}
	_, err3 := suite.msgServer.CommitBid(suite.ctx, commitMsg3)
	require.NoError(suite.T(), err3)

	err3 = suite.keeper.TransitionAuctionPhase(suite.ctx, currentHeight)
	require.NoError(suite.T(), err3)

	zeroHeightMsg := &consensusv1.MsgRevealBid{
		Validator:   "cosmos1validator3",
		Nonce:       "nonce3",
		BidAmount:   "3000000",
		BlockHeight: 0, // Will use current block height
	}
	resp2, err3 := suite.msgServer.RevealBid(suite.ctx, zeroHeightMsg)
	require.NoError(suite.T(), err3)
	require.NotNil(suite.T(), resp2)
	require.True(suite.T(), resp2.Success)
}

func TestMsgServerTestSuite(t *testing.T) {
	suite.Run(t, new(MsgServerTestSuite))
}
