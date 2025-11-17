package keeper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

type QueryServerTestSuite struct {
	suite.Suite

	cdc        codec.Codec
	ctx        sdk.Context
	keeper     *Keeper
	queryServer QueryServer
	storeKey   storetypes.StoreKey
	paramStore paramtypes.Subspace
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
	suite.queryServer = NewQueryServer(suite.keeper)
	// Set default params
	suite.keeper.SetParams(suite.ctx, types.DefaultParams())
}

func TestQueryServerTestSuite(t *testing.T) {
	suite.Run(t, new(QueryServerTestSuite))
}

func (suite *QueryServerTestSuite) TestParams() {
	ctx := context.Background()
	req := &identv1.QueryParamsRequest{}
	
	resp, err := suite.queryServer.Params(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotNil(suite.T(), resp.Params)
	require.Equal(suite.T(), uint64(5), resp.Params.MaxIdentitiesPerAddress)
	require.True(suite.T(), resp.Params.RequireIdentityVerification)
	require.Equal(suite.T(), "default-provider", resp.Params.DefaultVerificationProvider)
}

func (suite *QueryServerTestSuite) TestVerifiedAccount() {
	// Create a verified account
	account := &identv1.VerifiedAccount{
		Address:              "cosmos1test",
		Role:                 identv1.Role_ROLE_CITIZEN,
		IsActive:             true,
		VerificationDate:     timestamppb.Now(),
		VerificationProvider: "test-provider",
		ZkpProof:             "test_proof_1234567890123456789012345678901234567890123456789012345678901234",
		IdentityHash:         "test_hash",
	}
	err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
	require.NoError(suite.T(), err)

	// Query the account
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &identv1.QueryVerifiedAccountRequest{
		Address: "cosmos1test",
	}
	
	resp, err := suite.queryServer.VerifiedAccount(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotNil(suite.T(), resp.VerifiedAccount)
	require.Equal(suite.T(), account.Address, resp.VerifiedAccount.Address)
	require.Equal(suite.T(), account.Role, resp.VerifiedAccount.Role)
}

func (suite *QueryServerTestSuite) TestVerifiedAccount_EmptyAddress() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &identv1.QueryVerifiedAccountRequest{
		Address: "",
	}
	
	_, err := suite.queryServer.VerifiedAccount(ctx, req)
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "empty") // "address cannot be empty" or "empty address"
}

func (suite *QueryServerTestSuite) TestVerifiedAccount_NotFound() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &identv1.QueryVerifiedAccountRequest{
		Address: "cosmos1nonexistent",
	}
	
	_, err := suite.queryServer.VerifiedAccount(ctx, req)
	require.Error(suite.T(), err)
}

func (suite *QueryServerTestSuite) TestVerifiedAccounts() {
	// Increase account limit for testing
	params := suite.keeper.GetParams(suite.ctx)
	params.MaxIdentitiesPerAddress = 10
	suite.keeper.SetParams(suite.ctx, params)
	
	// Create multiple verified accounts with different roles to avoid limit issues
	accounts := []*identv1.VerifiedAccount{
		{
			Address:              "cosmos1test1",
			Role:                 identv1.Role_ROLE_CITIZEN,
			IsActive:             true,
			VerificationDate:     timestamppb.Now(),
			VerificationProvider: "test-provider",
			ZkpProof:             "test_proof_1234567890123456789012345678901234567890123456789012345678901234",
			IdentityHash:         "test_hash1",
		},
		{
			Address:              "cosmos1test2",
			Role:                 identv1.Role_ROLE_VALIDATOR,
			IsActive:             true,
			VerificationDate:     timestamppb.Now(),
			VerificationProvider: "test-provider",
			ZkpProof:             "test_proof_1234567890123456789012345678901234567890123456789012345678901234",
			IdentityHash:         "test_hash2",
		},
		{
			Address:              "cosmos1test3",
			Role:                 identv1.Role_ROLE_CITIZEN,
			IsActive:             true,
			VerificationDate:     timestamppb.Now(),
			VerificationProvider: "test-provider",
			ZkpProof:             "test_proof_1234567890123456789012345678901234567890123456789012345678901234",
			IdentityHash:         "test_hash3",
		},
	}
	
	for _, account := range accounts {
		err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
		require.NoError(suite.T(), err)
	}

	// Query all accounts
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &identv1.QueryVerifiedAccountsRequest{}
	
	resp, err := suite.queryServer.VerifiedAccounts(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.GreaterOrEqual(suite.T(), len(resp.VerifiedAccounts), 3)
}

func (suite *QueryServerTestSuite) TestVerifiedAccounts_WithPagination() {
	// Increase account limit for testing
	params := suite.keeper.GetParams(suite.ctx)
	params.MaxIdentitiesPerAddress = 10
	suite.keeper.SetParams(suite.ctx, params)
	
	// Create multiple verified accounts with different roles to avoid limit issues
	for i := 0; i < 5; i++ {
		role := identv1.Role_ROLE_CITIZEN
		if i%2 == 1 {
			role = identv1.Role_ROLE_VALIDATOR
		}
		account := &identv1.VerifiedAccount{
			Address:              "cosmos1test" + string(rune('0'+i)),
			Role:                 role,
			IsActive:             true,
			VerificationDate:     timestamppb.Now(),
			VerificationProvider: "test-provider",
			ZkpProof:             "test_proof_1234567890123456789012345678901234567890123456789012345678901234",
			IdentityHash:         "test_hash" + string(rune('0'+i)),
		}
		err := suite.keeper.SetVerifiedAccount(suite.ctx, account)
		require.NoError(suite.T(), err)
	}

	// Query with pagination
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &identv1.QueryVerifiedAccountsRequest{
		Pagination: &sdkquery.PageRequest{
			Offset: 1,
			Limit:  2,
		},
	}
	
	resp, err := suite.queryServer.VerifiedAccounts(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.LessOrEqual(suite.T(), len(resp.VerifiedAccounts), 2)
}

