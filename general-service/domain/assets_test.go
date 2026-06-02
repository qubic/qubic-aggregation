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

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return([]domain.AssetOwnership{
		{AssetName: "CFB", AssetIssuer: "CFBM", ManagingContractIndex: 1, NumberOfShares: 1000, TickNumber: 100},
	}, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return([]domain.AssetPossession{
		{AssetName: "CFB", AssetIssuer: "CFBM", ManagingContractIndex: 1, NumberOfShares: 1000, TickNumber: 101},
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

func TestGetAssetsForIdentities_OwnedOnly(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return([]domain.AssetOwnership{
		{AssetName: "QFT", AssetIssuer: "TFUY", ManagingContractIndex: 1, NumberOfShares: 50, TickNumber: 200},
	}, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return(nil, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "id1", result[0].Identity)
	require.Len(t, result[0].Ownerships, 1)
	assert.Empty(t, result[0].Possessions)
	assert.Equal(t, "QFT", result[0].Ownerships[0].AssetName)
	assert.Equal(t, int64(50), result[0].Ownerships[0].NumberOfShares)
}

func TestGetAssetsForIdentities_PossessedOnly(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return(nil, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return([]domain.AssetPossession{
		{AssetName: "RANDOM", AssetIssuer: "AAAA", ManagingContractIndex: 1, NumberOfShares: 10, TickNumber: 300},
	}, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Empty(t, result[0].Ownerships)
	require.Len(t, result[0].Possessions, 1)
	assert.Equal(t, "RANDOM", result[0].Possessions[0].AssetName)
	assert.Equal(t, int64(10), result[0].Possessions[0].NumberOfShares)
}

func TestGetAssetsForIdentities_DifferentManagingContracts(t *testing.T) {
	// Regression: an ownership and possession for the same (asset, issuer) but
	// under different managing contracts must each appear as separate rows on
	// their respective list. They must not be merged or have their contract
	// index overwritten.
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return([]domain.AssetOwnership{
		{AssetName: "CFB", AssetIssuer: "CFBM", ManagingContractIndex: 1, NumberOfShares: 1000, TickNumber: 100},
	}, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return([]domain.AssetPossession{
		{AssetName: "CFB", AssetIssuer: "CFBM", ManagingContractIndex: 2, NumberOfShares: 1000, TickNumber: 100},
	}, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Len(t, result[0].Ownerships, 1)
	require.Len(t, result[0].Possessions, 1)
	assert.Equal(t, uint32(1), result[0].Ownerships[0].ManagingContractIndex)
	assert.Equal(t, uint32(2), result[0].Possessions[0].ManagingContractIndex)
}

func TestGetAssetsForIdentities_MultipleIdentitiesPreservesOrder(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return([]domain.AssetOwnership{
		{AssetName: "CFB", AssetIssuer: "CFBM", ManagingContractIndex: 1, NumberOfShares: 1, TickNumber: 100},
	}, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return(nil, nil)
	live.EXPECT().GetOwnedAssets(gomock.Any(), "id2").Return([]domain.AssetOwnership{
		{AssetName: "QFT", AssetIssuer: "TFUY", ManagingContractIndex: 1, NumberOfShares: 2, TickNumber: 100},
	}, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id2").Return(nil, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1", "id2"})
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "id1", result[0].Identity)
	assert.Equal(t, "id2", result[1].Identity)
}

func TestGetAssetsForIdentities_UpstreamOwnedError(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return(nil, fmt.Errorf("node unavailable")).AnyTimes()
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return(nil, nil).AnyTimes()

	_, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting owned assets")
}

func TestGetAssetsForIdentities_UpstreamPossessedError(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return(nil, nil).AnyTimes()
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return(nil, fmt.Errorf("node unavailable")).AnyTimes()

	_, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting possessed assets")
}

func TestGetAssetsForIdentities_Empty(t *testing.T) {
	svc, _ := newTestAssetsService(t)
	ctx := context.Background()

	result, err := svc.GetAssetsForIdentities(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, result)
}
