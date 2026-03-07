package keeper

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

	"github.com/volnix-protocol/volnix-protocol/x/governance/types"
	consensustypes "github.com/volnix-protocol/volnix-protocol/x/consensus/types"
	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
)

type ParameterApplierTestSuite struct {
	suite.Suite

	cdc        codec.Codec
	ctx        sdk.Context
	keeper     *Keeper
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
}

func (suite *ParameterApplierTestSuite) SetupTest() {
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
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func TestParameterApplierTestSuite(t *testing.T) {
	suite.Run(t, new(ParameterApplierTestSuite))
}

func (suite *ParameterApplierTestSuite) TestValidateParameterChange() {
	// Test valid parameter change
	change := &governancev1.ParameterChange{
		Module:    "governance",
		Parameter: "voting_period",
		OldValue:  "24h",
		NewValue:  "48h",
	}
	err := suite.keeper.ValidateParameterChange(suite.ctx, change)
	require.NoError(suite.T(), err)

	// Test constitutional parameter (should fail)
	change = &governancev1.ParameterChange{
		Module:    "consensus",
		Parameter: "total_supply", // Not governable
		OldValue:  "1000000",
		NewValue:  "2000000",
	}
	err = suite.keeper.ValidateParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "constitutional")
}

func (suite *ParameterApplierTestSuite) TestValidateParameterChange_EmptyModule() {
	change := &governancev1.ParameterChange{
		Module:    "",
		Parameter: "voting_period",
		NewValue:  "48h",
	}
	err := suite.keeper.ValidateParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	// ValidateParameterChange checks IsGovernable first, which will fail for empty module
	// So we check for either error message
	require.True(suite.T(), 
		err.Error() == "module cannot be empty" || 
		err.Error() == "parameter is constitutional and cannot be changed via governance")
}

func (suite *ParameterApplierTestSuite) TestValidateParameterChange_EmptyParameter() {
	change := &governancev1.ParameterChange{
		Module:    "governance",
		Parameter: "",
		NewValue:  "48h",
	}
	err := suite.keeper.ValidateParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	// ValidateParameterChange checks IsGovernable first, which will fail for empty parameter
	// So we check for either error message
	require.True(suite.T(), 
		err.Error() == "parameter cannot be empty" || 
		err.Error() == "parameter is constitutional and cannot be changed via governance")
}

func (suite *ParameterApplierTestSuite) TestValidateParameterChange_EmptyValue() {
	change := &governancev1.ParameterChange{
		Module:    "governance",
		Parameter: "voting_period",
		NewValue:  "",
	}
	err := suite.keeper.ValidateParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "new value cannot be empty")
}

func (suite *ParameterApplierTestSuite) TestValidateParameterChange_InvalidDuration() {
	change := &governancev1.ParameterChange{
		Module:    "governance",
		Parameter: "voting_period",
		NewValue:  "invalid_duration",
	}
	err := suite.keeper.ValidateParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "invalid duration")
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_Governance() {
	// Test applying governance parameter change
	change := &governancev1.ParameterChange{
		Module:    "governance",
		Parameter: "voting_period",
		OldValue:  "24h",
		NewValue:  "48h",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.NoError(suite.T(), err)

	// Verify the change was applied
	params := suite.keeper.GetParams(suite.ctx)
	expectedDuration, _ := time.ParseDuration("48h")
	require.Equal(suite.T(), expectedDuration, params.VotingPeriod)
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_Constitutional() {
	// Test trying to change constitutional parameter
	change := &governancev1.ParameterChange{
		Module:    "consensus",
		Parameter: "total_supply",
		OldValue:  "1000000",
		NewValue:  "2000000",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "constitutional")
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_UnknownModule() {
	change := &governancev1.ParameterChange{
		Module:    "unknown_module",
		Parameter: "some_param",
		NewValue:  "value",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	// ApplyParameterChange checks IsGovernable first, which will fail for unknown module
	// So we check for either error message
	require.True(suite.T(), 
		err.Error() == "unknown module: unknown_module" || 
		err.Error() == "parameter is constitutional and cannot be changed via governance")
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_Quorum() {
	change := &governancev1.ParameterChange{
		Module:    "governance",
		Parameter: "quorum",
		OldValue:  "0.5",
		NewValue:  "0.6",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.NoError(suite.T(), err)

	params := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), "0.6", params.Quorum)
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_InvalidQuorum() {
	change := &governancev1.ParameterChange{
		Module:    "governance",
		Parameter: "quorum",
		OldValue:  "0.5",
		NewValue:  "1.5", // Invalid: > 1
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "between 0 and 1")
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_Threshold() {
	change := &governancev1.ParameterChange{
		Module:    "governance",
		Parameter: "threshold",
		OldValue:  "0.5",
		NewValue:  "0.6",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.NoError(suite.T(), err)

	params := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), "0.6", params.Threshold)
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_TimelockPeriod() {
	change := &governancev1.ParameterChange{
		Module:    "governance",
		Parameter: "timelock_period",
		OldValue:  "24h",
		NewValue:  "48h",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.NoError(suite.T(), err)

	params := suite.keeper.GetParams(suite.ctx)
	expectedDuration, _ := time.ParseDuration("48h")
	require.Equal(suite.T(), expectedDuration, params.TimelockPeriod)
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_MinDeposit() {
	change := &governancev1.ParameterChange{
		Module:    "governance",
		Parameter: "min_deposit",
		OldValue:  "1000000",
		NewValue:  "2000000",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.NoError(suite.T(), err)

	params := suite.keeper.GetParams(suite.ctx)
	require.Equal(suite.T(), "2000000", params.MinDeposit)
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_LizenzKeeperNotSet() {
	change := &governancev1.ParameterChange{
		Module:    "lizenz",
		Parameter: "activity_coefficient",
		NewValue:  "0.2",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "lizenz keeper not set")
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_AnteilKeeperNotSet() {
	change := &governancev1.ParameterChange{
		Module:    "anteil",
		Parameter: "min_ant_amount",
		NewValue:  "100",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "anteil keeper not set")
}

func (suite *ParameterApplierTestSuite) TestApplyParameterChange_ConsensusKeeperNotSet() {
	change := &governancev1.ParameterChange{
		Module:    "consensus",
		Parameter: "base_block_time",
		NewValue:  "10s",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "consensus keeper not set")
}

// =============================================================================
// Consensus Parameter Change Tests (with mock keeper)
// =============================================================================

type mockConsensusKeeper struct {
	params consensustypes.Params
}

func (m *mockConsensusKeeper) GetParams(_ sdk.Context) consensustypes.Params {
	return m.params
}

func (m *mockConsensusKeeper) SetParams(_ sdk.Context, params consensustypes.Params) {
	m.params = params
}

func (suite *ParameterApplierTestSuite) setupConsensusKeeper() *mockConsensusKeeper {
	mock := &mockConsensusKeeper{
		params: *consensustypes.DefaultParams(),
	}
	suite.keeper.SetConsensusKeeper(mock)
	return mock
}

func (suite *ParameterApplierTestSuite) TestApplyConsensusChange_BaseBlockTime() {
	mock := suite.setupConsensusKeeper()

	change := &governancev1.ParameterChange{
		Module:    "consensus",
		Parameter: "base_block_time",
		OldValue:  "5s",
		NewValue:  "10s",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "10s", mock.params.BaseBlockTime)
}

func (suite *ParameterApplierTestSuite) TestApplyConsensusChange_BaseBlockTimeInvalid() {
	suite.setupConsensusKeeper()

	change := &governancev1.ParameterChange{
		Module:    "consensus",
		Parameter: "base_block_time",
		NewValue:  "not-a-duration",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "invalid duration")
}

func (suite *ParameterApplierTestSuite) TestApplyConsensusChange_HighActivityThreshold() {
	mock := suite.setupConsensusKeeper()

	change := &governancev1.ParameterChange{
		Module:    "consensus",
		Parameter: "high_activity_threshold",
		NewValue:  "5000",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(5000), mock.params.HighActivityThreshold)
}

func (suite *ParameterApplierTestSuite) TestApplyConsensusChange_HighActivityThresholdInvalid() {
	suite.setupConsensusKeeper()

	change := &governancev1.ParameterChange{
		Module:    "consensus",
		Parameter: "high_activity_threshold",
		NewValue:  "not-a-number",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "invalid uint64")
}

func (suite *ParameterApplierTestSuite) TestApplyConsensusChange_LowActivityThreshold() {
	mock := suite.setupConsensusKeeper()

	change := &governancev1.ParameterChange{
		Module:    "consensus",
		Parameter: "low_activity_threshold",
		NewValue:  "100",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(100), mock.params.LowActivityThreshold)
}

func (suite *ParameterApplierTestSuite) TestApplyConsensusChange_UnknownParam() {
	suite.setupConsensusKeeper()

	// Unknown params are caught by IsGovernable before reaching applyConsensusParameterChange
	change := &governancev1.ParameterChange{
		Module:    "consensus",
		Parameter: "nonexistent_param",
		NewValue:  "42",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "constitutional")
}

func (suite *ParameterApplierTestSuite) TestApplyConsensusChange_LowActivityThresholdInvalid() {
	suite.setupConsensusKeeper()

	change := &governancev1.ParameterChange{
		Module:    "consensus",
		Parameter: "low_activity_threshold",
		NewValue:  "xyz",
	}
	err := suite.keeper.ApplyParameterChange(suite.ctx, change)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "invalid uint64")
}

