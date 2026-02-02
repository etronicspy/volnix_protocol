package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/volnix-protocol/volnix-protocol/x/governance/types"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDurationToProto(t *testing.T) {
	d := 5 * time.Second
	proto := types.DurationToProto(d)
	require.NotNil(t, proto)
	require.Equal(t, d, proto.AsDuration())
}

func TestProtoToDuration(t *testing.T) {
	require.Equal(t, time.Duration(0), types.ProtoToDuration(nil))

	proto := durationpb.New(10 * time.Second)
	require.Equal(t, 10*time.Second, types.ProtoToDuration(proto))
}
