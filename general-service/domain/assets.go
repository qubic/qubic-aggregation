package domain

import (
	"context"
	"fmt"

	"go.uber.org/zap"
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

// GetAssetsForIdentities fetches the assets of every identity in a single batched request.
// The live service pipelines the underlying node requests, so this no longer fans out one
// request per identity per asset kind.
func (s *AssetsService) GetAssetsForIdentities(ctx context.Context, identities []string) ([]IdentityAssets, error) {
	if len(identities) == 0 {
		return nil, nil
	}

	identitiesAssets, err := s.liveService.GetAssetsForIdentities(ctx, identities)
	if err != nil {
		return nil, fmt.Errorf("requesting assets for identities: %w", err)
	}

	return identitiesAssets, nil
}
