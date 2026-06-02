package domain

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type AssetsService struct {
	logger      *zap.SugaredLogger
	liveService LiveService
}

func NewAssetsService(logger *zap.SugaredLogger, liveService LiveService) *AssetsService {
	return &AssetsService{
		logger:      logger,
		liveService: liveService,
	}
}

func (s *AssetsService) GetAssetsForIdentities(ctx context.Context, identities []string) ([]IdentityAssets, error) {
	results := make([]IdentityAssets, len(identities))
	g, ctx := errgroup.WithContext(ctx)

	for i, identity := range identities {
		g.Go(func() error {
			ownerships, possessions, err := s.fetchAssetsForIdentity(ctx, identity)
			if err != nil {
				return fmt.Errorf("fetching assets for identity %s: %w", identity, err)
			}
			results[i] = IdentityAssets{
				Identity:    identity,
				Ownerships:  ownerships,
				Possessions: possessions,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

func (s *AssetsService) fetchAssetsForIdentity(ctx context.Context, identity string) ([]AssetOwnership, []AssetPossession, error) {
	var ownerships []AssetOwnership
	var possessions []AssetPossession

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		o, err := s.liveService.GetOwnedAssets(ctx, identity)
		if err != nil {
			return fmt.Errorf("requesting owned assets: %w", err)
		}
		ownerships = o
		return nil
	})
	g.Go(func() error {
		p, err := s.liveService.GetPossessedAssets(ctx, identity)
		if err != nil {
			return fmt.Errorf("requesting possessed assets: %w", err)
		}
		possessions = p
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	return ownerships, possessions, nil
}
