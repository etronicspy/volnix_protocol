package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
)

var _ lizenzv1.QueryServer = (*QueryServer)(nil)

type QueryServer struct {
	lizenzv1.UnimplementedQueryServer
	k *Keeper
}

func NewQueryServer(k *Keeper) QueryServer {
	return QueryServer{k: k}
}

// GetRewardHistory returns reward history for a validator
func (q QueryServer) GetRewardHistory(ctx context.Context, req *lizenzv1.QueryRewardHistoryRequest) (*lizenzv1.QueryRewardHistoryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get reward history
	history, err := q.k.GetRewardHistory(sdkCtx, req.Validator)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert RewardRecord to proto format
	var protoRecords []*lizenzv1.RewardRecord
	for _, record := range history {
		protoRecords = append(protoRecords, &lizenzv1.RewardRecord{
			BlockHeight:    record.BlockHeight,
			RewardAmount:   record.RewardAmount,
			Timestamp:      record.Timestamp,
			MoaCompliance:  fmt.Sprintf("%.4f", record.MOACompliance),
			PenaltyApplied: fmt.Sprintf("%.4f", record.PenaltyApplied),
			BaseReward:     record.BaseReward,
		})
	}

	return &lizenzv1.QueryRewardHistoryResponse{
		Validator:     req.Validator,
		RewardHistory:  protoRecords,
		TotalRecords:   uint64(len(protoRecords)),
	}, nil
}

// GetRewardStats returns comprehensive reward statistics for a validator
func (q QueryServer) GetRewardStats(ctx context.Context, req *lizenzv1.QueryRewardStatsRequest) (*lizenzv1.QueryRewardStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get reward stats
	stats, err := q.k.GetRewardStats(sdkCtx, req.Validator)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert RewardRecord to proto format
	var protoRecords []*lizenzv1.RewardRecord
	for _, record := range stats.RewardHistory {
		protoRecords = append(protoRecords, &lizenzv1.RewardRecord{
			BlockHeight:    record.BlockHeight,
			RewardAmount:   record.RewardAmount,
			Timestamp:      record.Timestamp,
			MoaCompliance:  fmt.Sprintf("%.4f", record.MOACompliance),
			PenaltyApplied: fmt.Sprintf("%.4f", record.PenaltyApplied),
			BaseReward:     record.BaseReward,
		})
	}

	return &lizenzv1.QueryRewardStatsResponse{
		TotalRewardsEarned: stats.TotalRewardsEarned,
		LastRewardBlock:    stats.LastRewardBlock,
		LastRewardTime:     stats.LastRewardTime,
		RewardHistory:      protoRecords,
		TotalRewardsCount:  uint64(stats.TotalRewardsCount),
	}, nil
}
