package clients

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/qubic/go-data-publisher/status-service/protobuf"
	"github.com/qubic/qubic-aggregation/general-service/clients/mocks"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

func newTestStatusClient(t *testing.T) (*StatusServiceClient, *mocks.MockStatusServiceClient) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockStatusServiceClient(ctrl)
	return &StatusServiceClient{client: mock, logger: zap.NewNop().Sugar()}, mock
}

func newTestStatusClientWithCache(t *testing.T, ttl time.Duration) (*StatusServiceClient, *mocks.MockStatusServiceClient) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockStatusServiceClient(ctrl)
	ipsCache := ttlcache.New[string, domain.IngestionPipelineStatus](
		ttlcache.WithTTL[string, domain.IngestionPipelineStatus](ttl),
		ttlcache.WithDisableTouchOnHit[string, domain.IngestionPipelineStatus](),
	)
	return &StatusServiceClient{
		client:   mock,
		logger:   zap.NewNop().Sugar(),
		ipsCache: ipsCache,
		sfGroup:  &singleflight.Group{},
	}, mock
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

func TestGetIngestionPipelineStatus_MapsAllFields(t *testing.T) {
	ssc, mock := newTestStatusClientWithCache(t, time.Minute)
	ctx := context.Background()

	mock.EXPECT().GetStatus(ctx, gomock.Any()).Return(&protobuf.GetStatusResponse{
		LastProcessedTick:    1000,
		ProcessingEpoch:      42,
		IntervalInitialTick:  900,
		LastProcessedLogTick: 950,
	}, nil)

	status, err := ssc.GetIngestionPipelineStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint32(1000), status.LastProcessedTick)
	assert.Equal(t, uint32(42), status.ProcessingEpoch)
	assert.Equal(t, uint32(900), status.IntervalInitialTick)
	assert.Equal(t, uint32(950), status.LastProcessedLogTick)
}

func TestGetIngestionPipelineStatus_CachesResult(t *testing.T) {
	ssc, mock := newTestStatusClientWithCache(t, time.Minute)
	ctx := context.Background()

	// Upstream must be hit exactly once across two calls.
	mock.EXPECT().GetStatus(ctx, gomock.Any()).Return(&protobuf.GetStatusResponse{
		LastProcessedLogTick: 950,
	}, nil).Times(1)

	first, err := ssc.GetIngestionPipelineStatus(ctx)
	require.NoError(t, err)

	second, err := ssc.GetIngestionPipelineStatus(ctx)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Equal(t, uint32(950), second.LastProcessedLogTick)
}

func TestGetIngestionPipelineStatus_UpstreamError(t *testing.T) {
	ssc, mock := newTestStatusClientWithCache(t, time.Minute)
	ctx := context.Background()

	mock.EXPECT().GetStatus(ctx, gomock.Any()).Return(nil, fmt.Errorf("connection refused"))

	_, err := ssc.GetIngestionPipelineStatus(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requesting ingestion pipeline status from status service")
}
