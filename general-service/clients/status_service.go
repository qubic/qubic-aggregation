package clients

//go:generate mockgen -destination=mocks/mock_status_service.go -package=mocks github.com/qubic/go-data-publisher/status-service/protobuf StatusServiceClient

import (
	"context"
	"fmt"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/qubic/go-data-publisher/status-service/protobuf"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"
)

const ipsCacheKey = "ips"

type StatusServiceClient struct {
	client protobuf.StatusServiceClient
	logger *zap.SugaredLogger

	ipsCache *ttlcache.Cache[string, domain.IngestionPipelineStatus]
	sfGroup  *singleflight.Group
}

func NewStatusServiceClient(conn *grpc.ClientConn, logger *zap.SugaredLogger, cacheTTLDuration time.Duration) *StatusServiceClient {

	ipsCache := ttlcache.New[string, domain.IngestionPipelineStatus](
		ttlcache.WithTTL[string, domain.IngestionPipelineStatus](cacheTTLDuration),
		ttlcache.WithDisableTouchOnHit[string, domain.IngestionPipelineStatus](), // Do not reset TTL on hit
	)

	return &StatusServiceClient{
		client:   protobuf.NewStatusServiceClient(conn),
		logger:   logger,
		ipsCache: ipsCache,
		sfGroup:  &singleflight.Group{},
	}
}

func (ssc *StatusServiceClient) Start() {
	ssc.ipsCache.Start()
}

func (ssc *StatusServiceClient) Stop() {
	ssc.ipsCache.Stop()
}

func (ssc *StatusServiceClient) GetTickIntervals(ctx context.Context) (map[uint32][]domain.TickInterval, error) {
	tickIntervalsResponse, err := ssc.client.GetTickIntervals(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("requesting tick intervals from status service: %w", err)
	}

	epochIntervals := make(map[uint32][]domain.TickInterval)
	for _, interval := range tickIntervalsResponse.Intervals {
		epochIntervals[interval.Epoch] = append(epochIntervals[interval.Epoch], domain.TickInterval{
			First: interval.FirstTick,
			Last:  interval.LastTick,
		})
	}

	return epochIntervals, nil
}

func (ssc *StatusServiceClient) GetIngestionPipelineStatus(ctx context.Context) (domain.IngestionPipelineStatus, error) {

	cachedStatus, err, _ := ssc.sfGroup.Do(ipsCacheKey, func() (any, error) {
		// Return cached value if it exists
		if item := ssc.ipsCache.Get(ipsCacheKey); item != nil {
			return item.Value(), nil
		}

		// Fetch new value otherwise
		statusResponse, err := ssc.client.GetStatus(ctx, nil)
		if err != nil {
			return domain.IngestionPipelineStatus{}, fmt.Errorf("requesting ingestion pipeline status from status service: %w", err)
		}
		status := domain.IngestionPipelineStatus{
			LastProcessedTick:    statusResponse.LastProcessedTick,
			ProcessingEpoch:      statusResponse.ProcessingEpoch,
			IntervalInitialTick:  statusResponse.IntervalInitialTick,
			LastProcessedLogTick: statusResponse.LastProcessedLogTick,
		}
		ssc.ipsCache.Set(ipsCacheKey, status, ttlcache.DefaultTTL)

		return status, nil
	})
	if err != nil {
		return domain.IngestionPipelineStatus{}, fmt.Errorf("getting cached ingestion pipeline status: %w", err)
	}

	status, ok := cachedStatus.(domain.IngestionPipelineStatus)
	if !ok {
		return domain.IngestionPipelineStatus{}, fmt.Errorf("invalid type assertion for ingestion pipeline status: expected domain.IngestionPipelineStatus, got %T", cachedStatus)
	}

	return status, nil
}
