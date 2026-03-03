package clients

import (
	"context"
	"fmt"
	"testing"

	"github.com/qubic/go-data-publisher/status-service/protobuf"
	"github.com/qubic/qubic-aggregation/general-service/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func newTestStatusClient(t *testing.T) (*StatusServiceClient, *mocks.MockStatusServiceClient) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockStatusServiceClient(ctrl)
	return &StatusServiceClient{client: mock, logger: zap.NewNop().Sugar()}, mock
}

func TestGetTickIntervals_GroupsByEpoch(t *testing.T) {
	ssc, mock := newTestStatusClient(t)
	ctx := context.Background()

	mock.EXPECT().GetTickIntervals(ctx, gomock.Any()).Return(&protobuf.GetTickIntervalsResponse{
		Intervals: []*protobuf.TickInterval{
			{Epoch: 10, FirstTick: 100, LastTick: 200},
			{Epoch: 10, FirstTick: 201, LastTick: 300},
			{Epoch: 11, FirstTick: 301, LastTick: 400},
		},
	}, nil)

	result, err := ssc.GetTickIntervals(ctx)
	require.NoError(t, err)
	require.Len(t, result, 2)

	require.Len(t, result[10], 2)
	assert.Equal(t, uint32(100), result[10][0].First)
	assert.Equal(t, uint32(200), result[10][0].Last)
	assert.Equal(t, uint32(201), result[10][1].First)
	assert.Equal(t, uint32(300), result[10][1].Last)

	require.Len(t, result[11], 1)
	assert.Equal(t, uint32(301), result[11][0].First)
	assert.Equal(t, uint32(400), result[11][0].Last)
}

func TestGetTickIntervals_UpstreamError(t *testing.T) {
	ssc, mock := newTestStatusClient(t)
	ctx := context.Background()

	mock.EXPECT().GetTickIntervals(ctx, gomock.Any()).Return(nil, fmt.Errorf("connection refused"))

	_, err := ssc.GetTickIntervals(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting tick intervals from status service")
}

func TestGetTickIntervals_EmptyResponse(t *testing.T) {
	ssc, mock := newTestStatusClient(t)
	ctx := context.Background()

	mock.EXPECT().GetTickIntervals(ctx, gomock.Any()).Return(&protobuf.GetTickIntervalsResponse{
		Intervals: nil,
	}, nil)

	result, err := ssc.GetTickIntervals(ctx)
	require.NoError(t, err)
	assert.Empty(t, result)
}
