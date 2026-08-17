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

func newTestAssetsService(t *testing.T) (*domain.AssetsService, *mocks.MockLiveService) {
	ctrl := gomock.NewController(t)
	live := mocks.NewMockLiveService(ctrl)
	logger := zap.NewNop().Sugar()
	return domain.NewAssetsService(logger, live), live
}

func TestGetAssetsForIdentities_OwnedAndPossessedPopulated(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetAssetsForIdentities(gomock.Any(), []string{"id1"}).Return([]domain.IdentityAssets{
		{
			Identity: "id1",
			Ownerships: []domain.AssetOwnership{
				{AssetName: "CFB", AssetIssuer: "CFBM", ManagingContractIndex: 1, NumberOfShares: 1000, TickNumber: 100},
			},
			Possessions: []domain.AssetPossession{
				{AssetName: "CFB", AssetIssuer: "CFBM", ManagingContractIndex: 1, NumberOfShares: 1000, TickNumber: 101},
			},
		},
	}, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	require.NoError(t, err)
	require.Len(t, result, 1)

	ia := result[0]
	assert.Equal(t, "id1", ia.Identity)
	require.Len(t, ia.Ownerships, 1)
	require.Len(t, ia.Possessions, 1)
	assert.Equal(t, "CFB", ia.Ownerships[0].AssetName)
	assert.Equal(t, int64(1000), ia.Ownerships[0].NumberOfShares)
	assert.Equal(t, uint32(100), ia.Ownerships[0].TickNumber)
	assert.Equal(t, int64(1000), ia.Possessions[0].NumberOfShares)
	assert.Equal(t, uint32(101), ia.Possessions[0].TickNumber)
}

func TestGetAssetsForIdentities_MultipleIdentitiesRequestedInOneBatch(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	// all identities go upstream in a single call, and the returned order is kept
	live.EXPECT().GetAssetsForIdentities(gomock.Any(), []string{"id1", "id2"}).Times(1).Return([]domain.IdentityAssets{
		{
			Identity: "id1",
			Ownerships: []domain.AssetOwnership{
				{AssetName: "CFB", AssetIssuer: "CFBM", ManagingContractIndex: 1, NumberOfShares: 1, TickNumber: 100},
			},
		},
		{
			Identity: "id2",
			Ownerships: []domain.AssetOwnership{
				{AssetName: "QFT", AssetIssuer: "TFUY", ManagingContractIndex: 1, NumberOfShares: 2, TickNumber: 100},
			},
		},
	}, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1", "id2"})
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "id1", result[0].Identity)
	assert.Equal(t, "id2", result[1].Identity)
	assert.Equal(t, "CFB", result[0].Ownerships[0].AssetName)
	assert.Equal(t, "QFT", result[1].Ownerships[0].AssetName)
}

func TestGetAssetsForIdentities_UpstreamError(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetAssetsForIdentities(gomock.Any(), []string{"id1"}).Return(nil, fmt.Errorf("node unavailable"))

	_, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting assets for identities")
	assert.Contains(t, err.Error(), "node unavailable")
}

func TestGetAssetsForIdentities_Empty(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	// nothing to ask for, so the live service is never called
	live.EXPECT().GetAssetsForIdentities(gomock.Any(), gomock.Any()).Times(0)

	result, err := svc.GetAssetsForIdentities(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, result)
}
