package domain

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type BalancesService struct {
	logger      *zap.SugaredLogger
	liveService LiveService
}

func NewBalancesService(logger *zap.SugaredLogger, liveService LiveService) *BalancesService {
	return &BalancesService{
		logger:      logger,
		liveService: liveService,
	}
}

func (s *BalancesService) GetBalancesForIdentities(ctx context.Context, identities []string) ([]IdentityBalance, error) {
	balances := make([]IdentityBalance, len(identities))
	g, ctx := errgroup.WithContext(ctx)

	// Fetch concurrently
	for i, identity := range identities {
		g.Go(func() error {
			balance, err := s.liveService.GetBalance(ctx, identity)
			if err != nil {
				return fmt.Errorf("requesting balance for identity %s: %w", identity, err)
			}
			balances[i] = balance
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return balances, nil
}
