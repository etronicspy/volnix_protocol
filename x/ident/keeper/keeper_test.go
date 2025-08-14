package keeper_test

import (
	"testing"

	sdklog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	apppkg "github.com/volnix-protocol/volnix-protocol/app"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	identmod "github.com/volnix-protocol/volnix-protocol/x/ident"
	"github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

type testEnv struct {
	ctx          sdk.Context
	cdc          codec.Codec
	paramsKeeper paramskeeper.Keeper
	identKeeper  keeper.Keeper
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()

	encoding := apppkg.MakeEncodingConfig()
	logger := sdklog.NewNopLogger()
	db := dbm.NewMemDB()

	bapp := baseapp.NewBaseApp("ident-test", logger, db, encoding.TxConfig.TxDecoder())
	bapp.SetInterfaceRegistry(encoding.InterfaceRegistry)

	// store keys
	// store keys
	keyParamsKV := paramtypes.StoreKey
	tkeyParamsName := paramtypes.TStoreKey

	keyParamsStore := storetypes.NewKVStoreKey(keyParamsKV)
	tkeyParamsStore := storetypes.NewTransientStoreKey(tkeyParamsName)
	keyIdent := storetypes.NewKVStoreKey(identtypes.StoreKey)

	// mount stores
	bapp.MountKVStores(map[string]*storetypes.KVStoreKey{
		keyParamsKV:         keyParamsStore,
		identtypes.StoreKey: keyIdent,
	})
	bapp.MountTransientStores(map[string]*storetypes.TransientStoreKey{
		tkeyParamsName: tkeyParamsStore,
	})
	require.NoError(t, bapp.LoadLatestVersion())

	ctx := bapp.NewUncachedContext(false, tmproto.Header{})

	// params keeper + subspace
	pkeeper := paramskeeper.NewKeeper(encoding.Codec, encoding.LegacyAmino, keyParamsStore, tkeyParamsStore)
	subspace := pkeeper.Subspace(identtypes.ModuleName)

	// module keeper
	ik := identmod.NewKeeper(encoding.Codec, keyIdent, subspace)

	return testEnv{ctx: ctx, cdc: encoding.Codec, paramsKeeper: pkeeper, identKeeper: ik}
}

func TestParamsValidateDefault(t *testing.T) {
	params := identtypes.DefaultParams()
	require.NoError(t, params.Validate())
}

func TestGenesisInitExportRoundTrip(t *testing.T) {
	env := newTestEnv(t)

	gen := identmod.DefaultGenesis()
	identmod.InitGenesis(env.ctx, env.identKeeper, gen)
	exp := identmod.ExportGenesis(env.ctx, env.identKeeper)

	// compare params JSON
	got, err := protojson.Marshal(exp.Params)
	require.NoError(t, err)
	want, err := protojson.Marshal(gen.Params)
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(got))
}

func TestQueryParamsReturnsDefault(t *testing.T) {
	env := newTestEnv(t)
	gen := identmod.DefaultGenesis()
	identmod.InitGenesis(env.ctx, env.identKeeper, gen)

	q := keeper.NewQueryServer(env.identKeeper)
	resp, err := q.Params(sdk.WrapSDKContext(env.ctx), &identv1.QueryParamsRequest{})
	require.NoError(t, err)

	// expected protojson
	want, err := protojson.Marshal(identtypes.DefaultParams().ToProto())
	require.NoError(t, err)
	require.JSONEq(t, string(want), resp.Json)
}
