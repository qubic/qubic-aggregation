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

func TestGetAssetsForIdentities_MergesOwnedAndPossessed(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return([]domain.OwnedAsset{
		{PublicId: "id1", AssetName: "CFB", IssuerIdentity: "CFBM", ContractIndex: 1, Amount: 1000, Tick: 100},
	}, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return([]domain.PossessedAsset{
		{PublicId: "id1", AssetName: "CFB", IssuerIdentity: "CFBM", ContractIndex: 1, Amount: 1000, Tick: 101},
	}, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "id1", result[0].PublicId)
	assert.Equal(t, "CFB", result[0].AssetName)
	assert.Equal(t, int64(1000), result[0].OwnedAmount)
	assert.Equal(t, int64(1000), result[0].PossessedAmount)
	assert.Equal(t, uint32(100), result[0].OwnedValidForTick)
	assert.Equal(t, uint32(101), result[0].PossessedValidForTick)
}

func TestGetAssetsForIdentities_OwnedOnly(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return([]domain.OwnedAsset{
		{PublicId: "id1", AssetName: "QFT", IssuerIdentity: "TFUY", ContractIndex: 1, Amount: 50, Tick: 200},
	}, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return(nil, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, int64(50), result[0].OwnedAmount)
	assert.Equal(t, int64(0), result[0].PossessedAmount)
	assert.Equal(t, uint32(200), result[0].OwnedValidForTick)
	assert.Equal(t, uint32(0), result[0].PossessedValidForTick)
}

func TestGetAssetsForIdentities_PossessedOnly(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return(nil, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return([]domain.PossessedAsset{
		{PublicId: "id1", AssetName: "RANDOM", IssuerIdentity: "AAAA", ContractIndex: 1, Amount: 10, Tick: 300},
	}, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, int64(0), result[0].OwnedAmount)
	assert.Equal(t, int64(10), result[0].PossessedAmount)
	assert.Equal(t, uint32(0), result[0].OwnedValidForTick)
	assert.Equal(t, uint32(300), result[0].PossessedValidForTick)
}

func TestGetAssetsForIdentities_MultipleIdentitiesPreservesOrder(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return([]domain.OwnedAsset{
		{PublicId: "id1", AssetName: "CFB", IssuerIdentity: "CFBM", ContractIndex: 1, Amount: 1, Tick: 100},
	}, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return(nil, nil)
	live.EXPECT().GetOwnedAssets(gomock.Any(), "id2").Return([]domain.OwnedAsset{
		{PublicId: "id2", AssetName: "QFT", IssuerIdentity: "TFUY", ContractIndex: 1, Amount: 2, Tick: 100},
	}, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id2").Return(nil, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1", "id2"})
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "id1", result[0].PublicId)
	assert.Equal(t, "id2", result[1].PublicId)
}

func TestGetAssetsForIdentities_MultipleAssetsPerIdentity(t *testing.T) {
	svc, live := newTestAssetsService(t)
	ctx := context.Background()

	live.EXPECT().GetOwnedAssets(gomock.Any(), "id1").Return([]domain.OwnedAsset{
		{PublicId: "id1", AssetName: "CFB", IssuerIdentity: "CFBM", ContractIndex: 1, Amount: 1000, Tick: 100},
		{PublicId: "id1", AssetName: "QFT", IssuerIdentity: "TFUY", ContractIndex: 1, Amount: 50, Tick: 100},
	}, nil)
	live.EXPECT().GetPossessedAssets(gomock.Any(), "id1").Return([]domain.PossessedAsset{
		{PublicId: "id1", AssetName: "CFB", IssuerIdentity: "CFBM", ContractIndex: 1, Amount: 1000, Tick: 100},
		{PublicId: "id1", AssetName: "QWALLET", IssuerIdentity: "AAAA", ContractIndex: 1, Amount: 200, Tick: 100},
	}, nil)

	result, err := svc.GetAssetsForIdentities(ctx, []string{"id1"})
	require.NoError(t, err)
	require.Len(t, result, 3)

	byName := map[string]domain.AssetBalance{}
	for _, b := range result {
		byName[b.AssetName] = b
	}
	assert.Equal(t, int64(1000), byName["CFB"].OwnedAmount)
	assert.Equal(t, int64(1000), byName["CFB"].PossessedAmount)
	assert.Equal(t, int64(50), byName["QFT"].OwnedAmount)
	assert.Equal(t, int64(0), byName["QFT"].PossessedAmount)
	assert.Equal(t, int64(0), byName["QWALLET"].OwnedAmount)
	assert.Equal(t, int64(200), byName["QWALLET"].PossessedAmount)
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
