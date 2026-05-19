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

// assetKey groups owned and possessed entries for the same asset held by the
// same identity. The same (assetName, issuerIdentity) can be held under
// different managing contracts, so contractIndex is part of the key.
type assetKey struct {
	publicId       string
	assetName      string
	issuerIdentity string
	contractIndex  uint32
}

func (s *AssetsService) GetAssetsForIdentities(ctx context.Context, identities []string) ([]AssetBalance, error) {
	perIdentity := make([][]AssetBalance, len(identities))
	g, ctx := errgroup.WithContext(ctx)

	for i, identity := range identities {
		g.Go(func() error {
			merged, err := s.fetchAssetsForIdentity(ctx, identity)
			if err != nil {
				return fmt.Errorf("fetching assets for identity %s: %w", identity, err)
			}
			perIdentity[i] = merged
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var result []AssetBalance
	for _, assets := range perIdentity {
		result = append(result, assets...)
	}
	return result, nil
}

func (s *AssetsService) fetchAssetsForIdentity(ctx context.Context, identity string) ([]AssetBalance, error) {
	var owned []OwnedAsset
	var possessed []PossessedAsset

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		o, err := s.liveService.GetOwnedAssets(ctx, identity)
		if err != nil {
			return fmt.Errorf("requesting owned assets: %w", err)
		}
		owned = o
		return nil
	})
	g.Go(func() error {
		p, err := s.liveService.GetPossessedAssets(ctx, identity)
		if err != nil {
			return fmt.Errorf("requesting possessed assets: %w", err)
		}
		possessed = p
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Preserve insertion order so output is deterministic across runs.
	indexByKey := make(map[assetKey]int, len(owned)+len(possessed))
	balances := make([]AssetBalance, 0, len(owned)+len(possessed))

	for _, o := range owned {
		key := assetKey{publicId: o.PublicId, assetName: o.AssetName, issuerIdentity: o.IssuerIdentity, contractIndex: o.ContractIndex}
		idx, ok := indexByKey[key]
		if !ok {
			balances = append(balances, AssetBalance{
				PublicId:          o.PublicId,
				AssetName:         o.AssetName,
				IssuerIdentity:    o.IssuerIdentity,
				ContractIndex:     o.ContractIndex,
				OwnedAmount:       o.Amount,
				OwnedValidForTick: o.Tick,
			})
			indexByKey[key] = len(balances) - 1
			continue
		}
		balances[idx].OwnedAmount = o.Amount
		balances[idx].OwnedValidForTick = o.Tick
	}

	for _, p := range possessed {
		key := assetKey{publicId: p.PublicId, assetName: p.AssetName, issuerIdentity: p.IssuerIdentity, contractIndex: p.ContractIndex}
		idx, ok := indexByKey[key]
		if !ok {
			balances = append(balances, AssetBalance{
				PublicId:              p.PublicId,
				AssetName:             p.AssetName,
				IssuerIdentity:        p.IssuerIdentity,
				ContractIndex:         p.ContractIndex,
				PossessedAmount:       p.Amount,
				PossessedValidForTick: p.Tick,
			})
			indexByKey[key] = len(balances) - 1
			continue
		}
		balances[idx].PossessedAmount = p.Amount
		balances[idx].PossessedValidForTick = p.Tick
	}

	return balances, nil
}
