package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

type KeeperTestSuite struct {
	suite.Suite

	cdc        codec.Codec
	paramStore paramtypes.Subspace
}

func (suite *KeeperTestSuite) SetupTest() {
	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)

	// Create a simple test store for params
	storeKey := storetypes.NewKVStoreKey("test_params")
	tKey := storetypes.NewTransientStoreKey("test_transient_store")

	// Create params keeper and subspace
	paramsKeeper := paramskeeper.NewKeeper(suite.cdc, codec.NewLegacyAmino(), storeKey, tKey)
	suite.paramStore = paramsKeeper.Subspace(identtypes.ModuleName)
	suite.paramStore.WithKeyTable(identtypes.ParamKeyTable())
}

func (suite *KeeperTestSuite) TestDefaultParams() {
	// Test that default params are valid
	params := identtypes.DefaultParams()
	require.NoError(suite.T(), params.Validate())

	// Test specific values
	require.Equal(suite.T(), uint64(10000), params.MaxCitizenAccounts)
	require.Equal(suite.T(), uint64(1000), params.MaxValidatorAccounts)
	require.Equal(suite.T(), true, params.RequireIdentityVerification)
}

func (suite *KeeperTestSuite) TestParamsValidation() {
	// Test valid params
	params := identtypes.DefaultParams()
	require.NoError(suite.T(), params.Validate())

	// Test invalid params (zero values)
	invalidParams := identtypes.DefaultParams()
	invalidParams.MaxCitizenAccounts = 0
	invalidParams.MaxValidatorAccounts = 0
	require.Error(suite.T(), invalidParams.Validate())
}

func (suite *KeeperTestSuite) TestParamsToProto() {
	// Test conversion to protobuf
	params := identtypes.DefaultParams()
	protoParams := params.ToProto()

	require.NotNil(suite.T(), protoParams)
	require.Equal(suite.T(), params.MaxCitizenAccounts, protoParams.MaxCitizenAccounts)
	require.Equal(suite.T(), params.MaxValidatorAccounts, protoParams.MaxValidatorAccounts)
	require.Equal(suite.T(), params.RequireIdentityVerification, protoParams.RequireIdentityVerification)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
