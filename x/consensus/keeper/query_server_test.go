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
	"google.golang.org/protobuf/types/known/timestamppb"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
)

type QueryServerTestSuite struct {
	suite.Suite

	cdc         codec.Codec
	ctx         sdk.Context
	keeper      *Keeper
	queryServer QueryServer
	storeKey    storetypes.StoreKey
	paramStore  paramtypes.Subspace
}

func (suite *QueryServerTestSuite) SetupTest() {
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)
	suite.storeKey = storetypes.NewKVStoreKey(types.StoreKey)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	suite.ctx = testutil.DefaultContext(suite.storeKey, tKey)
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), suite.storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(types.ModuleName)
	suite.paramStore = suite.paramStore.WithKeyTable(types.ParamKeyTable())
	suite.keeper = NewKeeper(suite.cdc, suite.storeKey, suite.paramStore)
	suite.queryServer = NewQueryServer(*suite.keeper)
	suite.keeper.SetParams(suite.ctx, *types.DefaultParams())
}

func TestQueryServerTestSuite(t *testing.T) {
	suite.Run(t, new(QueryServerTestSuite))
}

func (suite *QueryServerTestSuite) TestParams() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &consensusv1.QueryParamsRequest{}
	
	resp, err := suite.queryServer.Params(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotNil(suite.T(), resp.Params)
}

func (suite *QueryServerTestSuite) TestValidators() {
	// Create some validators
	validators := []*consensusv1.Validator{
		{
			Validator:     "cosmos1validator1",
			AntBalance:    "1000000",
			ActivityScore: "500",
			Status:        consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
			LastActive:    timestamppb.Now(),
		},
		{
			Validator:     "cosmos1validator2",
			AntBalance:    "2000000",
			ActivityScore: "600",
			Status:        consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
			LastActive:    timestamppb.Now(),
		},
	}
	
	for _, validator := range validators {
		suite.keeper.SetValidator(suite.ctx, validator)
	}

	// Query validators
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &consensusv1.QueryValidatorsRequest{}
	
	resp, err := suite.queryServer.Validators(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.GreaterOrEqual(suite.T(), len(resp.Validators), 2)
}

func (suite *QueryServerTestSuite) TestValidators_Empty() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &consensusv1.QueryValidatorsRequest{}
	
	resp, err := suite.queryServer.Validators(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	// Validators can be nil or empty slice - both are valid for empty result
	if resp.Validators != nil {
		require.Empty(suite.T(), resp.Validators)
	}
}

