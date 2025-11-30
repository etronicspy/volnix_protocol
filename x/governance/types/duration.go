package types

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
)

// DurationToProto converts time.Duration to *durationpb.Duration
func DurationToProto(d time.Duration) *durationpb.Duration {
	return durationpb.New(d)
}

// ProtoToDuration converts *durationpb.Duration to time.Duration
func ProtoToDuration(d *durationpb.Duration) time.Duration {
	if d == nil {
		return 0
	}
	return d.AsDuration()
}

