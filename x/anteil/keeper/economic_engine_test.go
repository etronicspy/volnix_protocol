package keeper_test

import (
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
)

func (suite *KeeperTestSuite) TestNewEconomicEngine() {
	engine := keeper.NewEconomicEngine(suite.keeper)
	require.NotNil(suite.T(), engine)
}

func (suite *KeeperTestSuite) TestProcessOrderMatching() {
	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Create matching buy and sell orders
	buyOrder := &anteilv1.Order{
		OrderId:      "buy1",
		Owner:        "cosmos1buyer",
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
		Owner:        "cosmos1seller",
		OrderType:    anteilv1.OrderType_ORDER_TYPE_LIMIT,
		OrderSide:    anteilv1.OrderSide_ORDER_SIDE_SELL,
		AntAmount:    "1000000",
		Price:        "1.5",
		Status:       anteilv1.OrderStatus_ORDER_STATUS_OPEN,
		CreatedAt:    timestamppb.Now(),
		IdentityHash: "hash_sell",
	}

	err := suite.keeper.SetOrder(suite.ctx, buyOrder)
	require.NoError(suite.T(), err)

	err = suite.keeper.SetOrder(suite.ctx, sellOrder)
	require.NoError(suite.T(), err)

	// Create economic engine and process matching
	engine := keeper.NewEconomicEngine(suite.keeper)
	err = engine.ProcessOrderMatching(suite.ctx)
	require.NoError(suite.T(), err)
}

func (suite *KeeperTestSuite) TestProcessAuctions_EconomicEngine() {
	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Create auction
	pastTime := currentTime.Add(-1 * time.Hour)
	auction := &anteilv1.Auction{
		AuctionId:    "auction1",
		BlockHeight:  1000,
		ReservePrice: "1000000",
		AntAmount:    "1000000",
		StartTime:    timestamppb.New(pastTime.Add(-24 * time.Hour)),
		EndTime:      timestamppb.New(pastTime),
		Status:       anteilv1.AuctionStatus_AUCTION_STATUS_OPEN,
		WinningBid:   "",
	}

	err := suite.keeper.SetAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)

	// Create economic engine and process auctions
	engine := keeper.NewEconomicEngine(suite.keeper)
	err = engine.ProcessAuctions(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify auction was cancelled (no bids)
	retrieved, err := suite.keeper.GetAuction(suite.ctx, "auction1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.AuctionStatus_AUCTION_STATUS_CANCELLED, retrieved.Status)
}

func (suite *KeeperTestSuite) TestProcessMarketMaking() {
	// Set block time
	currentTime := time.Now()
	suite.ctx = suite.ctx.WithBlockTime(currentTime)

	// Create economic engine and process market making
	engine := keeper.NewEconomicEngine(suite.keeper)
	err := engine.ProcessMarketMaking(suite.ctx)
	require.NoError(suite.T(), err)
}

func (suite *KeeperTestSuite) TestNewMatchingEngine() {
	engine := keeper.NewMatchingEngine()
	require.NotNil(suite.T(), engine)
}
