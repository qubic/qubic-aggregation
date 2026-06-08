package domain

import (
	"context"
	"fmt"
	"go.uber.org/zap"
)

type SmartContractRewardsService struct {
	logger         *zap.SugaredLogger
	elasticService ElasticService
}

func NewSmartContractRewardsService(logger *zap.SugaredLogger, elasticService ElasticService) *SmartContractRewardsService {
	return &SmartContractRewardsService{
		logger:         logger,
		elasticService: elasticService,
	}
}

func (s *SmartContractRewardsService) GetRewardsDistributionsForSmartContract(ctx context.Context, identity string, pagination Pagination) (SmartContractRewardsDistributionsResult, error) {
	result, err := s.elasticService.GetSmartContractDividendDistributions(ctx, identity, pagination)
	if err != nil {
		return SmartContractRewardsDistributionsResult{}, fmt.Errorf("getting smart contract divident distributions for identity %s: %w", identity, err)
	}

	return *result, nil
}
