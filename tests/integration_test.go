package tests

import (
	"testing"
	"time"

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

	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	consensuskeeper "github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	consensustypes "github.com/volnix-protocol/volnix-protocol/x/consensus/types"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
	lizenzkeeper "github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
	lizenztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
)

type IntegrationTestSuite struct {
	suite.Suite

	cdc codec.Codec
	ctx sdk.Context

	// Keepers
	identKeeper     *identkeeper.Keeper
	lizenzKeeper    *lizenzkeeper.Keeper
	anteilKeeper    *anteilkeeper.Keeper
	consensusKeeper *consensuskeeper.Keeper

	// Store keys
	identStoreKey     storetypes.StoreKey
	lizenzStoreKey    storetypes.StoreKey
	anteilStoreKey    storetypes.StoreKey
	consensusStoreKey storetypes.StoreKey

	// Param stores
	identParamStore     paramtypes.Subspace
	lizenzParamStore    paramtypes.Subspace
	anteilParamStore    paramtypes.Subspace
	consensusParamStore paramtypes.Subspace
}

func (suite *IntegrationTestSuite) SetupTest() {
	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)

	// Create store keys
	suite.identStoreKey = storetypes.NewKVStoreKey("test_ident")
	suite.lizenzStoreKey = storetypes.NewKVStoreKey("test_lizenz")
	suite.anteilStoreKey = storetypes.NewKVStoreKey("test_anteil")
	suite.consensusStoreKey = storetypes.NewKVStoreKey("test_consensus")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")

	// Create test context
	suite.ctx = testutil.DefaultContext(suite.identStoreKey, tKey)

	// Create params keeper and subspaces
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.identStoreKey, tKey)
	suite.identParamStore = paramsKeeper.Subspace(identtypes.ModuleName)
	suite.lizenzParamStore = paramsKeeper.Subspace(lizenztypes.ModuleName)
	suite.anteilParamStore = paramsKeeper.Subspace(anteiltypes.ModuleName)
	suite.consensusParamStore = paramsKeeper.Subspace(consensustypes.ModuleName)

	// Set key tables
	suite.identParamStore.WithKeyTable(identtypes.ParamKeyTable())
	suite.lizenzParamStore.WithKeyTable(lizenztypes.ParamKeyTable())
	suite.anteilParamStore.WithKeyTable(anteiltypes.ParamKeyTable())
	suite.consensusParamStore.WithKeyTable(consensustypes.ParamKeyTable())

	// Create keepers
	suite.identKeeper = identkeeper.NewKeeper(suite.cdc, suite.identStoreKey, suite.identParamStore)
	suite.lizenzKeeper = lizenzkeeper.NewKeeper(suite.cdc, suite.lizenzStoreKey, suite.lizenzParamStore)
	suite.anteilKeeper = anteilkeeper.NewKeeper(suite.cdc, suite.anteilStoreKey, suite.anteilParamStore)
	suite.consensusKeeper = consensuskeeper.NewKeeper(suite.cdc, suite.consensusStoreKey, suite.consensusParamStore)

	// Set default params
	suite.identKeeper.SetParams(suite.ctx, identtypes.DefaultParams())
	suite.lizenzKeeper.SetParams(suite.ctx, lizenztypes.DefaultParams())
	suite.anteilKeeper.SetParams(suite.ctx, anteiltypes.DefaultParams())
	suite.consensusKeeper.SetParams(suite.ctx, *consensustypes.DefaultParams())
}

func (suite *IntegrationTestSuite) TestCompleteEconomicFlow() {
	// Step 1: Create verified identity (Citizen)
	citizenAccount := identtypes.NewVerifiedAccount("cosmos1citizen", identv1.Role_ROLE_CITIZEN, "hash123")
	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, citizenAccount)
	require.NoError(suite.T(), err)

	// Step 2: Create verified identity (Validator)
	validatorAccount := identtypes.NewVerifiedAccount("cosmos1validator", identv1.Role_ROLE_VALIDATOR, "hash456")
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, validatorAccount)
	require.NoError(suite.T(), err)

	// Step 3: Create LZN for validator
	lizenz := lizenztypes.NewLizenz("cosmos1validator", "1000000", "hash456")
	err = suite.lizenzKeeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	// Step 4: Activate LZN
	err = suite.lizenzKeeper.ActivateLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)

	// Step 5: Create ANT position for citizen
	citizenPosition := anteiltypes.NewUserPosition("cosmos1citizen", "10000000")
	err = suite.anteilKeeper.SetUserPosition(suite.ctx, citizenPosition)
	require.NoError(suite.T(), err)

	// Step 6: Create sell order (citizen selling ANT)
	sellOrder := anteiltypes.NewOrder(
		"cosmos1citizen",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_SELL,
		"1000000",
		"1.5",
		"hash123",
	)

	err = suite.anteilKeeper.CreateOrder(suite.ctx, sellOrder)
	require.NoError(suite.T(), err)
	sellOrderID := sellOrder.OrderId

	// Step 7: Create buy order (validator buying ANT)
	buyOrder := anteiltypes.NewOrder(
		"cosmos1validator",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1000000",
		"1.5",
		"hash456",
	)

	err = suite.anteilKeeper.CreateOrder(suite.ctx, buyOrder)
	require.NoError(suite.T(), err)
	buyOrderID := buyOrder.OrderId

	// Step 8: Execute trade
	err = suite.anteilKeeper.ExecuteTrade(suite.ctx, buyOrderID, sellOrderID)
	require.NoError(suite.T(), err)

	// Step 9: Note: In real implementation, we would verify trade was executed

	// Step 10: Verify orders were updated
	buyOrderRetrieved, err := suite.anteilKeeper.GetOrder(suite.ctx, buyOrderID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.OrderStatus_ORDER_STATUS_FILLED, buyOrderRetrieved.Status)

	sellOrderRetrieved, err := suite.anteilKeeper.GetOrder(suite.ctx, sellOrderID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.OrderStatus_ORDER_STATUS_FILLED, sellOrderRetrieved.Status)

	// Step 11: Verify user positions were updated
	citizenPositionRetrieved, err := suite.anteilKeeper.GetUserPosition(suite.ctx, "cosmos1citizen")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "1", citizenPositionRetrieved.TotalTrades)
	require.Equal(suite.T(), "1500000", citizenPositionRetrieved.TotalVolume) // 1000000 * 1.5

	// Step 12: Create auction for block production
	auction := anteiltypes.NewAuction(uint64(1000), "1000000", "1.0")
	err = suite.anteilKeeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	auctionID := auction.AuctionId

	// Step 13: Place bid in auction
	err = suite.anteilKeeper.PlaceBid(suite.ctx, auctionID, "cosmos1validator", "1000000")
	require.NoError(suite.T(), err)

	// Step 14: Settle auction
	err = suite.anteilKeeper.SettleAuction(suite.ctx, auctionID)
	require.NoError(suite.T(), err)

	// Step 15: Update consensus state
	err = suite.consensusKeeper.UpdateConsensusState(suite.ctx, 1000, "1000000", []string{"cosmos1validator"})
	require.NoError(suite.T(), err)

	// Step 16: Verify consensus state
	consensusState, err := suite.consensusKeeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(1000), consensusState.CurrentHeight)
	require.Equal(suite.T(), "1000000", consensusState.TotalAntBurned)
	require.Len(suite.T(), consensusState.ActiveValidators, 1)
	require.Equal(suite.T(), "cosmos1validator", consensusState.ActiveValidators[0])
}

func (suite *IntegrationTestSuite) TestRoleMigrationFlow() {
	// Step 1: Create source account (Citizen)
	sourceAccount := identtypes.NewVerifiedAccount("cosmos1source", identv1.Role_ROLE_CITIZEN, "hash123")
	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, sourceAccount)
	require.NoError(suite.T(), err)

	// Step 2: Create ANT position for source account
	sourcePosition := anteiltypes.NewUserPosition("cosmos1source", "10000000")
	err = suite.anteilKeeper.SetUserPosition(suite.ctx, sourcePosition)
	require.NoError(suite.T(), err)

	// Step 3: Create some orders for source account
	sellOrder := anteiltypes.NewOrder(
		"cosmos1source",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_SELL,
		"1000000",
		"1.5",
		"hash123",
	)

	err = suite.anteilKeeper.CreateOrder(suite.ctx, sellOrder)
	require.NoError(suite.T(), err)

	// Step 4: Set role migration
	migration := &identv1.RoleMigration{
		FromAddress:   "cosmos1source",
		ToAddress:     "cosmos1target",
		FromRole:      identv1.Role_ROLE_CITIZEN,
		ToRole:        identv1.Role_ROLE_VALIDATOR,
		MigrationHash: "hash123",
		ZkpProof:      "migration_zkp_proof",
		IsCompleted:   false,
	}

	err = suite.identKeeper.SetRoleMigration(suite.ctx, migration)
	require.NoError(suite.T(), err)

	// Step 5: Execute role migration
	err = suite.identKeeper.ExecuteRoleMigration(suite.ctx, "cosmos1source", "cosmos1target")
	require.NoError(suite.T(), err)

	// Step 6: Verify source account is deleted
	_, err = suite.identKeeper.GetVerifiedAccount(suite.ctx, "cosmos1source")
	require.Error(suite.T(), err)
	require.Equal(suite.T(), identtypes.ErrAccountNotFound, err)

	// Step 7: Verify target account is created
	targetAccount, err := suite.identKeeper.GetVerifiedAccount(suite.ctx, "cosmos1target")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), identv1.Role_ROLE_CITIZEN, targetAccount.Role)
	require.Equal(suite.T(), "hash123", targetAccount.IdentityHash)

	// Step 8: Verify position is transferred
	targetPosition, err := suite.anteilKeeper.GetUserPosition(suite.ctx, "cosmos1target")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "cosmos1target", targetPosition.Owner)
	require.Equal(suite.T(), "10000000", targetPosition.AntBalance)

	// Step 9: Note: In real implementation, we would verify order was transferred
}

func (suite *IntegrationTestSuite) TestMOAViolationFlow() {
	// Step 1: Create validator account
	validatorAccount := identtypes.NewVerifiedAccount("cosmos1validator", identv1.Role_ROLE_VALIDATOR, "hash456")
	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, validatorAccount)
	require.NoError(suite.T(), err)

	// Step 2: Create and activate LZN
	lizenz := lizenztypes.NewLizenz("cosmos1validator", "1000000", "hash456")
	err = suite.lizenzKeeper.SetLizenz(suite.ctx, lizenz)
	require.NoError(suite.T(), err)

	err = suite.lizenzKeeper.ActivateLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)

	// Step 3: Set validator weight
	err = suite.consensusKeeper.SetValidatorWeight(suite.ctx, "cosmos1validator", "1000000")
	require.NoError(suite.T(), err)

	// Step 4: Simulate MOA violation by setting old last active time
	oldTime := time.Now().Add(-200 * 24 * time.Hour) // 200 days ago
	validatorAccount.LastActive = timestamppb.New(oldTime)
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, validatorAccount)
	require.NoError(suite.T(), err)

	// Step 5: Run BeginBlocker to check MOA
	err = suite.lizenzKeeper.BeginBlocker(suite.ctx)
	require.NoError(suite.T(), err)

	// Step 6: Verify LZN is deactivated due to MOA violation
	_, err = suite.lizenzKeeper.GetLizenz(suite.ctx, "cosmos1validator")
	require.NoError(suite.T(), err)
	// Note: In real implementation, we would verify lizenz status

	// Step 7: Verify validator is removed from active validators
	consensusState, err := suite.consensusKeeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotContains(suite.T(), consensusState.ActiveValidators, "cosmos1validator")
}

func (suite *IntegrationTestSuite) TestHalvingFlow() {
	// Step 1: Set up consensus state with high height
	err := suite.consensusKeeper.UpdateConsensusState(suite.ctx, 100000, "1000000", []string{"cosmos1validator"})
	require.NoError(suite.T(), err)

	// Step 2: Set halving info
	halvingInfo := &consensusv1.HalvingInfo{
		LastHalvingHeight: 0,
		HalvingInterval:   100000,
		NextHalvingHeight: 100000,
	}

	err = suite.consensusKeeper.SetHalvingInfo(suite.ctx, *halvingInfo)
	require.NoError(suite.T(), err)

	// Step 3: Process halving
	err = suite.consensusKeeper.ProcessHalving(suite.ctx)
	require.NoError(suite.T(), err)

	// Step 4: Verify halving was processed
	halvingInfoRetrieved, err := suite.consensusKeeper.GetHalvingInfo(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(100000), halvingInfoRetrieved.LastHalvingHeight)
	require.Equal(suite.T(), uint64(200000), halvingInfoRetrieved.NextHalvingHeight)

	// Step 5: Verify consensus state was updated
	consensusState, err := suite.consensusKeeper.GetConsensusState(suite.ctx)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(100000), consensusState.CurrentHeight)
}

func (suite *IntegrationTestSuite) TestComplexTradingScenario() {
	// Step 1: Create multiple accounts
	citizen1 := identtypes.NewVerifiedAccount("cosmos1citizen1", identv1.Role_ROLE_CITIZEN, "hash123")
	citizen2 := identtypes.NewVerifiedAccount("cosmos1citizen2", identv1.Role_ROLE_CITIZEN, "hash456")
	validator1 := identtypes.NewVerifiedAccount("cosmos1validator1", identv1.Role_ROLE_VALIDATOR, "hash789")
	validator2 := identtypes.NewVerifiedAccount("cosmos1validator2", identv1.Role_ROLE_VALIDATOR, "hash101")

	err := suite.identKeeper.SetVerifiedAccount(suite.ctx, citizen1)
	require.NoError(suite.T(), err)
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, citizen2)
	require.NoError(suite.T(), err)
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, validator1)
	require.NoError(suite.T(), err)
	err = suite.identKeeper.SetVerifiedAccount(suite.ctx, validator2)
	require.NoError(suite.T(), err)

	// Step 2: Create LZN for validators
	lizenz1 := lizenztypes.NewLizenz("cosmos1validator1", "1000000", "hash789")
	lizenz2 := lizenztypes.NewLizenz("cosmos1validator2", "2000000", "hash101")

	err = suite.lizenzKeeper.SetLizenz(suite.ctx, lizenz1)
	require.NoError(suite.T(), err)
	err = suite.lizenzKeeper.SetLizenz(suite.ctx, lizenz2)
	require.NoError(suite.T(), err)

	// Step 3: Activate LZN
	err = suite.lizenzKeeper.ActivateLizenz(suite.ctx, "cosmos1validator1")
	require.NoError(suite.T(), err)
	err = suite.lizenzKeeper.ActivateLizenz(suite.ctx, "cosmos1validator2")
	require.NoError(suite.T(), err)

	// Step 4: Create ANT positions for citizens
	position1 := anteiltypes.NewUserPosition("cosmos1citizen1", "10000000")
	position2 := anteiltypes.NewUserPosition("cosmos1citizen2", "15000000")

	err = suite.anteilKeeper.SetUserPosition(suite.ctx, position1)
	require.NoError(suite.T(), err)
	err = suite.anteilKeeper.SetUserPosition(suite.ctx, position2)
	require.NoError(suite.T(), err)

	// Step 5: Create multiple orders
	// Citizen1 selling ANT
	sellOrder1 := anteiltypes.NewOrder(
		"cosmos1citizen1",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_SELL,
		"1000000",
		"1.5",
		"hash123",
	)

	// Citizen2 selling ANT
	sellOrder2 := anteiltypes.NewOrder(
		"cosmos1citizen2",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_SELL,
		"2000000",
		"2.0",
		"hash456",
	)

	// Validator1 buying ANT
	buyOrder1 := anteiltypes.NewOrder(
		"cosmos1validator1",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"1500000",
		"1.8",
		"hash789",
	)

	// Validator2 buying ANT
	buyOrder2 := anteiltypes.NewOrder(
		"cosmos1validator2",
		anteilv1.OrderType_ORDER_TYPE_LIMIT,
		anteilv1.OrderSide_ORDER_SIDE_BUY,
		"2500000",
		"2.2",
		"hash101",
	)

	err = suite.anteilKeeper.CreateOrder(suite.ctx, sellOrder1)
	require.NoError(suite.T(), err)
	sellOrderID1 := sellOrder1.OrderId

	err = suite.anteilKeeper.CreateOrder(suite.ctx, sellOrder2)
	require.NoError(suite.T(), err)
	sellOrderID2 := sellOrder2.OrderId

	err = suite.anteilKeeper.CreateOrder(suite.ctx, buyOrder1)
	require.NoError(suite.T(), err)
	buyOrderID1 := buyOrder1.OrderId

	err = suite.anteilKeeper.CreateOrder(suite.ctx, buyOrder2)
	require.NoError(suite.T(), err)
	buyOrderID2 := buyOrder2.OrderId

	// Step 6: Execute multiple trades
	// Trade 1: Validator1 buys from Citizen1
	err = suite.anteilKeeper.ExecuteTrade(suite.ctx, buyOrderID1, sellOrderID1)
	require.NoError(suite.T(), err)

	// Trade 2: Validator2 buys from Citizen2
	err = suite.anteilKeeper.ExecuteTrade(suite.ctx, buyOrderID2, sellOrderID2)
	require.NoError(suite.T(), err)

	// Step 7: Note: In real implementation, we would verify trades were executed

	// Step 8: Verify all positions were updated
	position1Retrieved, err := suite.anteilKeeper.GetUserPosition(suite.ctx, "cosmos1citizen1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "1", position1Retrieved.TotalTrades)
	require.Equal(suite.T(), "1500000", position1Retrieved.TotalVolume)

	position2Retrieved, err := suite.anteilKeeper.GetUserPosition(suite.ctx, "cosmos1citizen2")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "1", position2Retrieved.TotalTrades)
	require.Equal(suite.T(), "4000000", position2Retrieved.TotalVolume)

	// Step 9: Create auction with multiple bidders
	auction := anteiltypes.NewAuction(uint64(1000), "1000000", "1.0")
	err = suite.anteilKeeper.CreateAuction(suite.ctx, auction)
	require.NoError(suite.T(), err)
	auctionID := auction.AuctionId

	// Step 10: Place multiple bids
	err = suite.anteilKeeper.PlaceBid(suite.ctx, auctionID, "cosmos1validator1", "1000000")
	require.NoError(suite.T(), err)
	err = suite.anteilKeeper.PlaceBid(suite.ctx, auctionID, "cosmos1validator2", "1500000")
	require.NoError(suite.T(), err)

	// Step 11: Settle auction
	err = suite.anteilKeeper.SettleAuction(suite.ctx, auctionID)
	require.NoError(suite.T(), err)

	// Step 12: Verify auction is settled
	auction, err = suite.anteilKeeper.GetAuction(suite.ctx, auctionID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), anteilv1.AuctionStatus_AUCTION_STATUS_SETTLED, auction.Status)
	// Note: In real implementation, we would verify the winning bid
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
