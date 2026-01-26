package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	
	"github.com/volnix-protocol/volnix-protocol/x/anteil/types"
)

func TestErrors(t *testing.T) {
	// Test that all errors are defined
	require.NotNil(t, types.ErrOrderNotFound)
	require.NotNil(t, types.ErrOrderAlreadyExists)
	require.NotNil(t, types.ErrTradeNotFound)
	require.NotNil(t, types.ErrTradeAlreadyExists)
	require.NotNil(t, types.ErrAuctionNotFound)
	require.NotNil(t, types.ErrAuctionClosed)
	require.NotNil(t, types.ErrAuctionExpired)
	require.NotNil(t, types.ErrBidNotFound)
	require.NotNil(t, types.ErrPositionNotFound)
	require.NotNil(t, types.ErrEmptyOwner)
	require.NotNil(t, types.ErrEmptyAntAmount)
	require.NotNil(t, types.ErrEmptyPrice)
	require.NotNil(t, types.ErrEmptyAuctionID)
	require.NotNil(t, types.ErrEmptyBidder)
	require.NotNil(t, types.ErrEmptyBidAmount)
	require.NotNil(t, types.ErrEmptyReservePrice)
}

func TestErrorMessages(t *testing.T) {
	// Test error messages are not empty
	require.NotEmpty(t, types.ErrOrderNotFound.Error())
	require.NotEmpty(t, types.ErrAuctionClosed.Error())
	require.NotEmpty(t, types.ErrPositionNotFound.Error())
}
