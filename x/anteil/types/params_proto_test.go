package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestParams_ToProto(t *testing.T) {
	p := types.DefaultParams()
	proto := p.ToProto()
	require.NotNil(t, proto)
	require.Equal(t, p.MinAntAmount, proto.MinAntAmount)
	require.Equal(t, p.MaxAntAmount, proto.MaxAntAmount)
	require.Equal(t, p.TradingFeeRate, proto.TradingFeeRate)
	require.Equal(t, p.AntDenom, proto.AntDenom)
	require.Equal(t, p.MaxOpenOrders, proto.MaxOpenOrders)
	require.NotNil(t, proto.OrderExpiry)
}

func TestParamsFromProto_Nil(t *testing.T) {
	p, err := types.ParamsFromProto(nil)
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams().MinAntAmount, p.MinAntAmount)
}

func TestParamsFromProto_Valid(t *testing.T) {
	proto := &anteilv1.Params{
		MinAntAmount:                "1",
		MaxAntAmount:                "2",
		TradingFeeRate:              "0.01",
		MinOrderSize:                "10",
		MaxOrderSize:                "20",
		OrderExpiry:                 durationpb.New(time.Hour),
		RequireIdentityVerification: true,
		AntDenom:                    "uant",
		MaxOpenOrders:               5,
		PricePrecision:              "0.001",
	}
	p, err := types.ParamsFromProto(proto)
	require.NoError(t, err)
	require.Equal(t, "1", p.MinAntAmount)
	require.Equal(t, "2", p.MaxAntAmount)
	require.Equal(t, time.Hour, p.OrderExpiry)
	require.Equal(t, uint32(5), p.MaxOpenOrders)
}
