package keeper_test

import (
	"encoding/hex"
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
	"github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

type AdvancedKeeperTestSuite struct {
	suite.Suite
	ctx        sdk.Context
	keeper     *keeper.Keeper
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
}

func TestAdvancedKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(AdvancedKeeperTestSuite))
}

func (suite *AdvancedKeeperTestSuite) SetupTest() {
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)
	suite.storeKey = storetypes.NewKVStoreKey(types.StoreKey)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore = suite.paramStore.WithKeyTable(types.ParamKeyTable())
	suite.keeper = keeper.NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)
	suite.keeper.SetParams(suite.ctx, *types.DefaultParams())
}

// Test CreateBlindAuction
func (suite *AdvancedKeeperTestSuite) TestCreateBlindAuction() {
	height := uint64(1000)
	auction, err := suite.keeper.CreateBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), auction)
	require.Equal(suite.T(), height, auction.BlockHeight)
	require.Equal(suite.T(), consensusv1.AuctionPhase_AUCTION_PHASE_COMMIT, auction.Phase)
	require.Empty(suite.T(), auction.Commits)
	require.Empty(suite.T(), auction.Reveals)
}

func (suite *AdvancedKeeperTestSuite) TestCreateBlindAuction_Duplicate() {
	height := uint64(1000)
	auction1, err := suite.keeper.CreateBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	
	// Try to create again - should return existing
	auction2, err := suite.keeper.CreateBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), auction1.BlockHeight, auction2.BlockHeight)
}

// Test CommitBid
func (suite *AdvancedKeeperTestSuite) TestCommitBid() {
	height := uint64(1000)
	validator := "cosmos1validator"
	
	// Create commit hash using HashCommit function (nonce:bidAmount format)
	nonce := "test_nonce_123"
	bidAmount := "1000000"
	// Use keeper.HashCommit which uses "nonce:bidAmount" format
	commitHash := keeper.HashCommit(nonce, bidAmount)
	
	err := suite.keeper.CommitBid(suite.ctx, validator, commitHash, height)
	require.NoError(suite.T(), err)
	
	// Verify auction was created and commit was added
	auction, err := suite.keeper.GetBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), auction.Commits, 1)
	require.Equal(suite.T(), validator, auction.Commits[0].Validator)
	require.Equal(suite.T(), commitHash, auction.Commits[0].CommitHash)
}

func (suite *AdvancedKeeperTestSuite) TestCommitBid_InvalidHash() {
	height := uint64(1000)
	validator := "cosmos1validator"
	
	// Invalid hash (too short)
	err := suite.keeper.CommitBid(suite.ctx, validator, "short", height)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "invalid commit hash")
}

func (suite *AdvancedKeeperTestSuite) TestCommitBid_Duplicate() {
	height := uint64(1000)
	validator := "cosmos1validator"
	// Create a valid 64-character hex hash
	commitHash := hex.EncodeToString(make([]byte, 32)) // 64 hex chars (32 bytes * 2)
	
	err := suite.keeper.CommitBid(suite.ctx, validator, commitHash, height)
	require.NoError(suite.T(), err)
	
	// Try to commit again - should fail
	err = suite.keeper.CommitBid(suite.ctx, validator, commitHash, height)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "already committed")
}

// Test RevealBid
func (suite *AdvancedKeeperTestSuite) TestRevealBid() {
	height := uint64(1000)
	validator := "cosmos1validator"
	nonce := "test_nonce_123"
	bidAmount := "1000000"
	
	// First commit using HashCommit
	commitHash := keeper.HashCommit(nonce, bidAmount)
	err := suite.keeper.CommitBid(suite.ctx, validator, commitHash, height)
	require.NoError(suite.T(), err)
	
	// Transition to reveal phase
	auction, err := suite.keeper.GetBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	auction.Phase = consensusv1.AuctionPhase_AUCTION_PHASE_REVEAL
	err = suite.keeper.SetBlindAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	
	// Create mock anteil keeper for balance check
	mockAnteilKeeper := &MockAnteilKeeper{
		positions: make(map[string]interface{}),
	}
	suite.keeper.SetAnteilKeeper(mockAnteilKeeper)
	
	// Reveal bid
	err = suite.keeper.RevealBid(suite.ctx, validator, nonce, bidAmount, height)
	require.NoError(suite.T(), err)
	
	// Verify reveal was added
	auction, err = suite.keeper.GetBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), auction.Reveals, 1)
	require.Equal(suite.T(), validator, auction.Reveals[0].Validator)
	require.Equal(suite.T(), bidAmount, auction.Reveals[0].BidAmount)
}

func (suite *AdvancedKeeperTestSuite) TestRevealBid_NoCommit() {
	height := uint64(1000)
	validator := "cosmos1validator"
	
	// Create auction in reveal phase
	auction, err := suite.keeper.CreateBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	auction.Phase = consensusv1.AuctionPhase_AUCTION_PHASE_REVEAL
	err = suite.keeper.SetBlindAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	
	// Try to reveal without commit - should fail
	err = suite.keeper.RevealBid(suite.ctx, validator, "nonce", "1000000", height)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "not committed")
}

func (suite *AdvancedKeeperTestSuite) TestRevealBid_CommitHashMismatch() {
	height := uint64(1000)
	validator := "cosmos1validator"
	
	// Commit with one hash
	commitHash := keeper.HashCommit("nonce1", "amount1")
	err := suite.keeper.CommitBid(suite.ctx, validator, commitHash, height)
	require.NoError(suite.T(), err)
	
	// Transition to reveal phase
	auction, err := suite.keeper.GetBlindAuction(suite.ctx, height)
	require.NoError(suite.T(), err)
	auction.Phase = consensusv1.AuctionPhase_AUCTION_PHASE_REVEAL
	err = suite.keeper.SetBlindAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	
	// Try to reveal with different nonce/amount - should fail
	mockAnteilKeeper := &MockAnteilKeeper{positions: make(map[string]interface{})}
	suite.keeper.SetAnteilKeeper(mockAnteilKeeper)
	
	err = suite.keeper.RevealBid(suite.ctx, validator, "wrong_nonce", "wrong_amount", height)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "commit hash")
}

// Test DistributeBaseRewards (simplified - requires lizenz keeper)
func (suite *AdvancedKeeperTestSuite) TestDistributeBaseRewards_NoLizenzKeeper() {
	height := uint64(1000)
	// Should not fail if lizenz keeper is not set
	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	require.NoError(suite.T(), err)
}

// Mock AnteilKeeper for testing
type MockAnteilKeeper struct {
	positions map[string]interface{}
}

func (m *MockAnteilKeeper) GetUserPosition(ctx sdk.Context, user string) (interface{}, error) {
	pos, ok := m.positions[user]
	if !ok {
		// Return a mock position with sufficient balance
		return &MockUserPosition{
			Owner:      user,
			AntBalance: "10000000", // 10 ANT
		}, nil
	}
	return pos, nil
}

func (m *MockAnteilKeeper) SetUserPosition(ctx sdk.Context, position interface{}) error {
	if pos, ok := position.(*MockUserPosition); ok {
		m.positions[pos.Owner] = pos
	}
	return nil
}

func (m *MockAnteilKeeper) UpdateUserPosition(ctx sdk.Context, user string, antBalance string, orderCount uint32) error {
	pos := &MockUserPosition{
		Owner:      user,
		AntBalance: antBalance,
	}
	m.positions[user] = pos
	return nil
}

type MockUserPosition struct {
	Owner      string
	AntBalance string
}

