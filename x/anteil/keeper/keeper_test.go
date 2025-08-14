package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

type KeeperTestSuite struct {
	suite.Suite
}

func (suite *KeeperTestSuite) SetupTest() {
	// Simple setup for basic tests
}

func (suite *KeeperTestSuite) TestDefaultParams() {
	// Test that default params are valid
	params := anteiltypes.DefaultParams()
	require.NoError(suite.T(), params.Validate())

	// Test specific values
	require.Equal(suite.T(), "1000000", params.MinAntAmount)
	require.Equal(suite.T(), "1000000000", params.MaxAntAmount)
	require.Equal(suite.T(), "0.001", params.TradingFeeRate)
	require.Equal(suite.T(), "uant", params.AntDenom)
	require.Equal(suite.T(), uint32(10), params.MaxOpenOrders)
}

func (suite *KeeperTestSuite) TestParamsValidation() {
	params := anteiltypes.DefaultParams()

	// Test valid params
	err := params.Validate()
	require.NoError(suite.T(), err)

	// Test invalid params
	invalidParams := params
	invalidParams.MinAntAmount = ""
	err = invalidParams.Validate()
	require.Error(suite.T(), err)
}

func (suite *KeeperTestSuite) TestParamKeyTable() {
	// Test that param key table can be created
	keyTable := anteiltypes.ParamKeyTable()
	require.NotNil(suite.T(), keyTable)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
