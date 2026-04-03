package keeper_test

import (
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

func (suite *KeeperTestSuite) TestNewEconomicEngine() {
	engine := keeper.NewEconomicEngine(suite.keeper)
	require.NotNil(suite.T(), engine)
}

func (suite *KeeperTestSuite) TestProcessOrderMatching() {
	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// SELL orders require sufficient ANT balance
	err := suite.keeper.SetUserPosition(suite.ctx, anteiltypes.NewUserPosition(addrSeller, "1000000"))
	require.NoError(suite.T(), err)

	// Create matching buy and sell orders
	buyOrder := &anteilv1.Order{
		OrderId:      "buy1",
		Owner:        addrBuyer,
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_BUY,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_buy",
	}

	sellOrder := &anteilv1.Order{
		OrderId:      "sell1",
		Owner:        addrSeller,
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_SELL,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_sell",
	}

	err = suite.keeper.SetOrder(suite.ctx, buyOrder)
	require.NoError(suite.T(), err)

	err = suite.keeper.SetOrder(suite.ctx, sellOrder)
	require.NoError(suite.T(), err)

	// Create economic engine and process matching
	engine := keeper.NewEconomicEngine(suite.keeper)
	err = engine.ProcessOrderMatching(suite.ctx)
	require.NoError(suite.T(), err)
}

func (suite *KeeperTestSuite) TestNewMatchingEngine() {
	engine := keeper.NewMatchingEngine()
	require.NotNil(suite.T(), engine)
}
