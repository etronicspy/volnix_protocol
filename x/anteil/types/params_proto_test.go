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
	require.Equal(t, p.TradingFeeRate, proto.TradingFeeRate)
	require.Equal(t, p.AntDenom, proto.AntDenom)
	require.Equal(t, p.MaxOpenOrders, proto.MaxOpenOrders)
	require.NotNil(t, proto.OrderExpiry)
	require.NotNil(t, proto.EpochLength)
	require.Equal(t, p.EpochCoefficient, proto.EpochCoefficient)
	require.Equal(t, p.EpochCoefficientMin, proto.EpochCoefficientMin)
	require.Equal(t, p.EpochCoefficientMax, proto.EpochCoefficientMax)
	require.Equal(t, p.SupplierEpochAntLimit, proto.SupplierEpochAntLimit)
}

func TestParamsFromProto_Nil(t *testing.T) {
	p, err := types.ParamsFromProto(nil)
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams().MinOrderSize, p.MinOrderSize)
}

func TestParamsFromProto_Valid(t *testing.T) {
	proto := &anteilv1.Params{
		MinOrderSize:                "10",
		MaxOrderSize:                "20",
		TradingFeeRate:              "0.01",
		OrderExpiry:                 durationpb.New(time.Hour),
		RequireIdentityVerification: true,
		AntDenom:                    "uant",
		MaxOpenOrders:               5,
		PricePrecision:              "0.001",
		EpochLength:                 durationpb.New(24 * time.Hour),
		EpochCoefficient:            "1.2",
		EpochCoefficientMin:         "0.80",
		EpochCoefficientMax:         "1.40",
		SupplierEpochAntLimit:       "50000000",
		CitizenAntDistributionPeriod: durationpb.New(2 * time.Minute),
	}
	p, err := types.ParamsFromProto(proto)
	require.NoError(t, err)
	require.Equal(t, "10", p.MinOrderSize)
	require.Equal(t, "20", p.MaxOrderSize)
	require.Equal(t, time.Hour, p.OrderExpiry)
	require.Equal(t, uint32(5), p.MaxOpenOrders)
	require.Equal(t, 24*time.Hour, p.EpochLength)
	require.Equal(t, "1.2", p.EpochCoefficient)
	require.Equal(t, "50000000", p.SupplierEpochAntLimit)
	require.Equal(t, 2*time.Minute, p.CitizenAntDistributionPeriod)
}
