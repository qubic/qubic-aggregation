package clients

//go:generate mockgen -destination=mocks/mock_status_service.go -package=mocks github.com/qubic/go-data-publisher/status-service/protobuf StatusServiceClient

import (
	"context"
	"fmt"

	"github.com/qubic/go-data-publisher/status-service/protobuf"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type StatusServiceClient struct {
	client protobuf.StatusServiceClient
	logger *zap.SugaredLogger
}

func NewStatusServiceClient(conn *grpc.ClientConn, logger *zap.SugaredLogger) *StatusServiceClient {
	return &StatusServiceClient{
		client: protobuf.NewStatusServiceClient(conn),
		logger: logger,
	}
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
