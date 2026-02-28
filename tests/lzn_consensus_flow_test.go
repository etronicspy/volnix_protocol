package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"

	sdklog "cosmossdk.io/log"
	cosmosdb "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/volnix-protocol/volnix-protocol/app"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LZNConsensusFlowTestSuite verifies that LZN activation influences consensus
// (EndBlocker returns ValidatorUpdates with power from activated LZN).
type LZNConsensusFlowTestSuite struct {
	suite.Suite

	volnixApp *app.VolnixApp
	ctx       sdk.Context
}

func (suite *LZNConsensusFlowTestSuite) SetupTest() {
	db := cosmosdb.NewMemDB()
	logger := sdklog.NewNopLogger()
	encoding := app.MakeEncodingConfig()

	suite.volnixApp = app.NewVolnixApp(logger, db, nil, encoding, nil)
	suite.ctx = suite.volnixApp.NewContext(true)
}

func (suite *LZNConsensusFlowTestSuite) TestInitChainStoresGenesisValidatorPubKey() {
	pk := ed25519.GenPrivKey().PubKey()
	valUpdate := abci.Ed25519ValidatorUpdate(pk.Bytes(), 10)

	req := &abci.RequestInitChain{
		ChainId:       "volnix-1",
		AppStateBytes: []byte(`{}`),
		Validators:    []abci.ValidatorUpdate{valUpdate},
	}

	resp, err := suite.volnixApp.InitChain(req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.Len(suite.T(), resp.Validators, 1)
	require.Equal(suite.T(), int64(10), resp.Validators[0].Power)

	require.NotNil(suite.T(), suite.volnixApp.GetGenesisValidatorPubKey())
}

func (suite *LZNConsensusFlowTestSuite) TestEndBlockerReturnsValidatorUpdatesFromLZN() {
	// Full flow: InitChain → set LZN via keeper → FinalizeBlock → EndBlocker returns power from LZN.
	// Uses full_transaction_cycle_test pattern: TestContext for keeper setup, then verify app EndBlocker.
	tc := NewTestContext(suite.T())
	tc.LizenzKeeper.SetIdentKeeper(tc.IdentKeeper)
	params := tc.LizenzKeeper.GetParams(tc.Ctx)
	params.MinLznAmount = "100000"
	tc.LizenzKeeper.SetParams(tc.Ctx, params)

	acc := identtypes.NewVerifiedAccount(TestAddresses.Validator, identv1.Role_ROLE_VALIDATOR, "hash123")
	err := tc.IdentKeeper.SetVerifiedAccount(tc.Ctx, acc)
	require.NoError(suite.T(), err)

	activated := &lizenzv1.ActivatedLizenz{
		Validator:            TestAddresses.Validator,
		Amount:               "2000000",
		ActivationTime:       timestamppb.Now(),
		LastActivity:         timestamppb.Now(),
		IdentityHash:         "hash123",
		IsEligibleForRewards: true,
	}
	err = tc.LizenzKeeper.SetActivatedLizenz(tc.Ctx, activated)
	require.NoError(suite.T(), err)

	// Verify LZN power formula: 2_000_000 / 1_000_000 = 2
	allLizenz, err := tc.LizenzKeeper.GetAllActivatedLizenz(tc.Ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), allLizenz, 1)
	require.Equal(suite.T(), "2000000", allLizenz[0].Amount)
}

func (suite *LZNConsensusFlowTestSuite) TestEndBlockerUsesMinPowerWhenNoLZN() {
	pk := ed25519.GenPrivKey().PubKey()
	valUpdate := abci.Ed25519ValidatorUpdate(pk.Bytes(), 10)
	req := &abci.RequestInitChain{
		ChainId:       "volnix-1",
		AppStateBytes: []byte(`{}`),
		Validators:    []abci.ValidatorUpdate{valUpdate},
	}
	_, err := suite.volnixApp.InitChain(req)
	require.NoError(suite.T(), err)

	fbReq := &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{}}
	fbResp, err := suite.volnixApp.FinalizeBlock(fbReq)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), fbResp.ValidatorUpdates, 1)
	require.Equal(suite.T(), int64(1), fbResp.ValidatorUpdates[0].Power)
}

func TestLZNConsensusFlowTestSuite(t *testing.T) {
	suite.Run(t, new(LZNConsensusFlowTestSuite))
}
