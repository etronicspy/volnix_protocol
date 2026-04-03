package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

type EventsTestSuite struct {
	KeeperTestSuite
}

func TestEventsTestSuite(t *testing.T) {
	suite.Run(t, new(EventsTestSuite))
}

func (suite *EventsTestSuite) TestEventTypesExist() {
	require.NotEmpty(suite.T(), types.EventTypeBurnExecuted)
	require.NotEmpty(suite.T(), types.EventTypeRewardDistributed)
	require.NotEmpty(suite.T(), types.EventTypeHalving)
	require.NotEmpty(suite.T(), types.EventTypeConsensusStateUpdate)
	require.NotEmpty(suite.T(), types.EventTypeValidatorPowerUpdate)
	require.NotEmpty(suite.T(), types.EventTypeFeeDistributed)
	require.NotEmpty(suite.T(), types.EventTypePerHeightBurn)
}

func (suite *EventsTestSuite) TestBurnExecutedEvent() {
	require.NotEmpty(suite.T(), types.EventTypeBurnExecuted)
	require.Equal(suite.T(), "consensus.burn_executed", types.EventTypeBurnExecuted)
}

func (suite *EventsTestSuite) TestRewardDistributedEvent() {
	require.NotEmpty(suite.T(), types.EventTypeRewardDistributed)
	require.Equal(suite.T(), "consensus.reward_distributed", types.EventTypeRewardDistributed)
}

func (suite *EventsTestSuite) TestPerHeightBurnEvent() {
	require.NotEmpty(suite.T(), types.EventTypePerHeightBurn)
	require.Equal(suite.T(), "consensus.per_height_burn", types.EventTypePerHeightBurn)
}

func (suite *EventsTestSuite) TestPoVBAttributes() {
	require.NotEmpty(suite.T(), types.AttributeKeySi)
	require.NotEmpty(suite.T(), types.AttributeKeyBi)
	require.NotEmpty(suite.T(), types.AttributeKeyIncI)
	require.NotEmpty(suite.T(), types.AttributeKeyLTot)
	require.NotEmpty(suite.T(), types.AttributeKeyLambda)
	require.NotEmpty(suite.T(), types.AttributeKeyTotalFees)
	require.NotEmpty(suite.T(), types.AttributeKeyTotalB)
}
