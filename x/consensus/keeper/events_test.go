package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	sdk "github.com/cosmos/cosmos-sdk/types"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

type EventsTestSuite struct {
	KeeperTestSuite
}

func TestEventsTestSuite(t *testing.T) {
	suite.Run(t, new(EventsTestSuite))
}

// TestEventTypesExist tests that all event types are defined
func (suite *EventsTestSuite) TestEventTypesExist() {
	require.NotEmpty(suite.T(), types.EventTypeBlockCreatorSelected)
	require.NotEmpty(suite.T(), types.EventTypeBurnExecuted)
	require.NotEmpty(suite.T(), types.EventTypeRewardDistributed)
	require.NotEmpty(suite.T(), types.EventTypeAuctionCompleted)
	require.NotEmpty(suite.T(), types.EventTypeBidCommitted)
	require.NotEmpty(suite.T(), types.EventTypeBidRevealed)
}

// TestBidCommittedEvent tests that bid committed event is emitted
func (suite *EventsTestSuite) TestBidCommittedEvent() {
	validator := sdk.AccAddress("validator1_______________").String()
	nonce := "test_nonce_123"
	bidAmount := "1000000"
	// Generate valid commit hash
	commitHash := keeper.HashCommit(nonce, bidAmount)
	height := uint64(1000)

	suite.ctx = suite.ctx.WithBlockHeight(int64(height))

	err := suite.keeper.CommitBid(suite.ctx, validator, commitHash, height)
	require.NoError(suite.T(), err)

	// Check that event was emitted
	events := suite.ctx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == types.EventTypeBidCommitted {
			found = true
			require.Equal(suite.T(), types.EventTypeBidCommitted, event.Type)
			// Check attributes
			for _, attr := range event.Attributes {
				if string(attr.Key) == types.AttributeKeyValidator {
					require.Equal(suite.T(), validator, string(attr.Value))
				}
				if string(attr.Key) == types.AttributeKeyCommitHash {
					require.Equal(suite.T(), commitHash, string(attr.Value))
				}
			}
			break
		}
	}
	require.True(suite.T(), found, "Bid committed event should be emitted")
}

// TestBidRevealedEvent tests that bid revealed event is emitted
func (suite *EventsTestSuite) TestBidRevealedEvent() {
	validator := sdk.AccAddress("validator1_______________").String()
	nonce := "test_nonce_123"
	bidAmount := "1000000"
	height := uint64(1000)

	suite.ctx = suite.ctx.WithBlockHeight(int64(height))

	// First commit - generate valid hash
	commitHash := keeper.HashCommit(nonce, bidAmount)
	err := suite.keeper.CommitBid(suite.ctx, validator, commitHash, height)
	require.NoError(suite.T(), err)

	// Transition to reveal phase
	auction, _ := suite.keeper.GetBlindAuction(suite.ctx, height)
	if auction != nil {
		auction.Phase = consensusv1.AuctionPhase_AUCTION_PHASE_REVEAL
		suite.keeper.SetBlindAuction(suite.ctx, auction)
	}

	// Then reveal
	err = suite.keeper.RevealBid(suite.ctx, validator, nonce, bidAmount, height)
	require.NoError(suite.T(), err)

	// Check that event was emitted
	events := suite.ctx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == types.EventTypeBidRevealed {
			found = true
			require.Equal(suite.T(), types.EventTypeBidRevealed, event.Type)
			// Check attributes
			for _, attr := range event.Attributes {
				if string(attr.Key) == types.AttributeKeyValidator {
					require.Equal(suite.T(), validator, string(attr.Value))
				}
				if string(attr.Key) == types.AttributeKeyBidAmount {
					require.Equal(suite.T(), bidAmount, string(attr.Value))
				}
			}
			break
		}
	}
	require.True(suite.T(), found, "Bid revealed event should be emitted")
}

// TestRewardDistributedEvent tests that reward distributed event is emitted
func (suite *EventsTestSuite) TestRewardDistributedEvent() {
	mockBankKeeper := NewMockBankKeeper()
	suite.keeper.SetBankKeeper(mockBankKeeper)

	mockLizenzKeeper := &MockLizenzKeeper{
		activatedLizenz: []interface{}{
			map[string]interface{}{
				"validator": sdk.AccAddress("validator1_______________").String(),
				"amount":    "1000000",
			},
		},
		moaCompliance: make(map[string]float64),
	}
	suite.keeper.SetLizenzKeeper(mockLizenzKeeper)

	height := uint64(1000)
	suite.ctx = suite.ctx.WithBlockHeight(int64(height))

	// Distribute rewards
	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	require.NoError(suite.T(), err)

	// Check that event was emitted
	events := suite.ctx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == types.EventTypeRewardDistributed {
			found = true
			require.Equal(suite.T(), types.EventTypeRewardDistributed, event.Type)
			break
		}
	}
	require.True(suite.T(), found, "Reward distributed event should be emitted")
}

// TestBurnExecutedEvent tests that burn executed event type exists
func (suite *EventsTestSuite) TestBurnExecutedEvent() {
	// This test verifies the event type exists
	require.NotEmpty(suite.T(), types.EventTypeBurnExecuted)
	require.Equal(suite.T(), "consensus.burn_executed", types.EventTypeBurnExecuted)
}
