package domain

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type SmartContractRewardsService struct {
	logger         *zap.SugaredLogger
	elasticService ElasticService
	statusService  StatusService
}

func NewSmartContractRewardsService(logger *zap.SugaredLogger, elasticService ElasticService, statusService StatusService) *SmartContractRewardsService {
	return &SmartContractRewardsService{
		logger:         logger,
		elasticService: elasticService,
		statusService:  statusService,
	}
}

func (s *SmartContractRewardsService) GetRewardsDistributionsForSmartContract(ctx context.Context, identity string, pagination Pagination) (SmartContractRewardsDistributionsResult, error) {

	ips, err := s.statusService.GetIngestionPipelineStatus(ctx)
	if err != nil {
		return SmartContractRewardsDistributionsResult{}, fmt.Errorf("getting ingestion pipeline status: %w", err)
	}

	result, err := s.elasticService.GetSmartContractDividendDistributions(ctx, identity, pagination, ips.LastProcessedLogTick)
	if err != nil {
		return SmartContractRewardsDistributionsResult{}, fmt.Errorf("getting smart contract dividend distributions for identity %s: %w", identity, err)
	}

	return *result, nil
}
