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

	addrTest, addrNonexistent string
	addrTest1, addrTest2, addrTest3 string
}

func (suite *QueryServerTestSuite) SetupTest() {
	cfg := sdk.GetConfig()
	if cfg.GetBech32AccountAddrPrefix() != "cosmos" {
		cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
	}
	suite.addrTest = mustAddr("0000000000000000000000000000000000000001")
	suite.addrNonexistent = mustAddr("0000000000000000000000000000000000000009")
	suite.addrTest1 = mustAddr("000000000000000000000000000000000000000a")
	suite.addrTest2 = mustAddr("000000000000000000000000000000000000000b")
	suite.addrTest3 = mustAddr("000000000000000000000000000000000000000c")
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
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &identv1.QueryParamsRequest{}

	resp, err := suite.queryServer.Params(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotNil(suite.T(), resp.Params)

	defaultParams := types.DefaultParams()
	require.Equal(suite.T(), defaultParams.MaxIdentitiesPerAddress, resp.Params.MaxIdentitiesPerAddress)
	require.Equal(suite.T(), defaultParams.RequireIdentityVerification, resp.Params.RequireIdentityVerification)
	require.Equal(suite.T(), defaultParams.DefaultVerificationProvider, resp.Params.DefaultVerificationProvider)
}

func (suite *QueryServerTestSuite) TestVerifiedAccount() {
	// Create a verified account
	account := &identv1.VerifiedAccount{
		Address:              suite.addrTest,
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
		Address: suite.addrTest,
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
		Address: suite.addrNonexistent,
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
			Address:              suite.addrTest1,
			Role:                 identv1.Role_ROLE_CITIZEN,
			IsActive:             true,
			VerificationDate:     timestamppb.Now(),
			VerificationProvider: "test-provider",
			ZkpProof:             "test_proof_1234567890123456789012345678901234567890123456789012345678901234",
			IdentityHash:         "test_hash1",
		},
		{
			Address:              suite.addrTest2,
			Role:                 identv1.Role_ROLE_VALIDATOR,
			IsActive:             true,
			VerificationDate:     timestamppb.Now(),
			VerificationProvider: "test-provider",
			ZkpProof:             "test_proof_1234567890123456789012345678901234567890123456789012345678901234",
			IdentityHash:         "test_hash2",
		},
		{
			Address:              suite.addrTest3,
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
	pagAddrs := []string{mustAddr("0000000000000000000000000000000000000010"), mustAddr("0000000000000000000000000000000000000011"), mustAddr("0000000000000000000000000000000000000012"), mustAddr("0000000000000000000000000000000000000013"), mustAddr("0000000000000000000000000000000000000014")}
	for i := 0; i < 5; i++ {
		role := identv1.Role_ROLE_CITIZEN
		if i%2 == 1 {
			role = identv1.Role_ROLE_VALIDATOR
		}
		account := &identv1.VerifiedAccount{
			Address:              pagAddrs[i],
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

func (suite *QueryServerTestSuite) TestVerificationProviders() {
	// Add two providers via keeper
	for i, id := range []string{"qp-a", "qp-b"} {
		p := &identv1.VerificationProvider{
			ProviderId:        id,
			ProviderName:      "Query Provider " + string(rune('A'+i)),
			ProviderPublicKey: "pk-" + id,
			AccreditationHash: "hash-" + id,
			IsAccredited:      true,
			AccreditationDate: timestamppb.Now(),
		}
		err := suite.keeper.SetVerificationProvider(suite.ctx, p)
		require.NoError(suite.T(), err)
	}
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &identv1.QueryVerificationProvidersRequest{}
	resp, err := suite.queryServer.VerificationProviders(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.Len(suite.T(), resp.VerificationProviders, 2)
	require.Equal(suite.T(), "qp-a", resp.VerificationProviders[0].ProviderId)
	require.Equal(suite.T(), "qp-b", resp.VerificationProviders[1].ProviderId)
}

func (suite *QueryServerTestSuite) TestVerificationProviders_Empty() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &identv1.QueryVerificationProvidersRequest{}
	resp, err := suite.queryServer.VerificationProviders(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.Empty(suite.T(), resp.VerificationProviders)
}

func (suite *QueryServerTestSuite) TestVerificationProviders_WithPagination() {
	for i := 0; i < 3; i++ {
		id := "pag-" + string(rune('a'+i))
		p := &identv1.VerificationProvider{
			ProviderId:        id,
			ProviderName:      "Paginated " + id,
			ProviderPublicKey: "pk",
			AccreditationHash: "hash",
			IsAccredited:      true,
			AccreditationDate: timestamppb.Now(),
		}
		err := suite.keeper.SetVerificationProvider(suite.ctx, p)
		require.NoError(suite.T(), err)
	}
	ctx := sdk.WrapSDKContext(suite.ctx)
	req := &identv1.QueryVerificationProvidersRequest{
		Pagination: &sdkquery.PageRequest{Offset: 1, Limit: 2},
	}
	resp, err := suite.queryServer.VerificationProviders(ctx, req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.LessOrEqual(suite.T(), len(resp.VerificationProviders), 2)
}

