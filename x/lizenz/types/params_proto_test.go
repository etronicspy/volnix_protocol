package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestParams_ToProto(t *testing.T) {
	p := types.DefaultParams()
	proto := p.ToProto()
	require.NotNil(t, proto)
	require.Equal(t, p.MaxActivatedPerValidator, proto.MaxActivatedPerValidator)
	require.Equal(t, p.LznDenom, proto.LznDenom)
	require.NotNil(t, proto.DeactivationPeriod)
}

func TestParamsFromProto_Nil(t *testing.T) {
	p, err := types.ParamsFromProto(nil)
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams().LznDenom, p.LznDenom)
}

func TestParamsFromProto_Valid(t *testing.T) {
	proto := &lizenzv1.Params{
		MaxActivatedPerValidator:    5,
		ActivityCoefficient:         "0.1",
		DeactivationPeriod:          durationpb.New(time.Hour),
		InactivityPeriod:            durationpb.New(30 * time.Minute),
		MinLznAmount:                "100",
		MaxLznAmount:                "1000",
		RequireIdentityVerification: true,
		LznDenom:                    "ulzn",
	}
	p, err := types.ParamsFromProto(proto)
	require.NoError(t, err)
	require.Equal(t, uint32(5), p.MaxActivatedPerValidator)
	require.Equal(t, time.Hour, p.DeactivationPeriod)
	require.Equal(t, "ulzn", p.LznDenom)
}
