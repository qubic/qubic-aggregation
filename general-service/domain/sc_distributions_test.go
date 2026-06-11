package domain_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/qubic/qubic-aggregation/general-service/domain"
	"github.com/qubic/qubic-aggregation/general-service/domain/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func newTestSCRewardsService(t *testing.T) (
	*domain.SmartContractRewardsService,
	*mocks.MockElasticService,
	*mocks.MockStatusService,
) {
	ctrl := gomock.NewController(t)
	elastic := mocks.NewMockElasticService(ctrl)
	status := mocks.NewMockStatusService(ctrl)
	svc := domain.NewSmartContractRewardsService(zap.NewNop().Sugar(), elastic, status)
	return svc, elastic, status
}

func TestGetRewardsDistributionsForSmartContract_PassesLastProcessedLogTickAsMaxTick(t *testing.T) {
	svc, elastic, status := newTestSCRewardsService(t)
	ctx := context.Background()
	pagination := domain.Pagination{Offset: 0, Size: 10}

	status.EXPECT().GetIngestionPipelineStatus(gomock.Any()).Return(domain.IngestionPipelineStatus{
		LastProcessedTick:    1000,
		LastProcessedLogTick: 950,
	}, nil)

	// maxTick must be the LastProcessedLogTick, not LastProcessedTick.
	elastic.EXPECT().GetSmartContractDividendDistributions(gomock.Any(), "id1", pagination, uint32(950)).Return(
		&domain.SmartContractRewardsDistributionsResult{
			TotalHits:               2,
			TotalAllTimeDistributed: 500,
			ValidForTick:            950,
		}, nil)

	result, err := svc.GetRewardsDistributionsForSmartContract(ctx, "id1", pagination)
	require.NoError(t, err)
	assert.Equal(t, uint32(950), result.ValidForTick)
	assert.Equal(t, uint32(2), result.TotalHits)
	assert.Equal(t, int64(500), result.TotalAllTimeDistributed)
}

func TestGetRewardsDistributionsForSmartContract_StatusServiceError(t *testing.T) {
	svc, _, status := newTestSCRewardsService(t)

	// Elastic has no EXPECT: gomock fails the test if it is called.
	status.EXPECT().GetIngestionPipelineStatus(gomock.Any()).Return(
		domain.IngestionPipelineStatus{}, fmt.Errorf("status unavailable"))

	_, err := svc.GetRewardsDistributionsForSmartContract(context.Background(), "id1", domain.Pagination{Size: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting ingestion pipeline status")
}

func TestGetRewardsDistributionsForSmartContract_ElasticError(t *testing.T) {
	svc, elastic, status := newTestSCRewardsService(t)

	status.EXPECT().GetIngestionPipelineStatus(gomock.Any()).Return(
		domain.IngestionPipelineStatus{LastProcessedLogTick: 50}, nil)
	elastic.EXPECT().GetSmartContractDividendDistributions(gomock.Any(), "id1", gomock.Any(), uint32(50)).Return(
		nil, fmt.Errorf("elastic down"))

	_, err := svc.GetRewardsDistributionsForSmartContract(context.Background(), "id1", domain.Pagination{Size: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting smart contract dividend distributions")
}
