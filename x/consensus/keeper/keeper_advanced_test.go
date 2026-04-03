package keeper_test

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
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func (suite *AdvancedKeeperTestSuite) TestDistributeBaseRewards_NoLizenzKeeper() {
	height := uint64(1000)
	err := suite.keeper.DistributeBaseRewards(suite.ctx, height)
	require.NoError(suite.T(), err)
}

type MockAnteilKeeper struct {
	positions map[string]interface{}
}

func (m *MockAnteilKeeper) GetUserPosition(ctx sdk.Context, user string) (interface{}, error) {
	pos, ok := m.positions[user]
	if !ok {
		return &MockUserPosition{
			Owner:      user,
			AntBalance: "10000000",
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
