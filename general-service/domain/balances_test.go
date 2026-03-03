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

func newTestBalancesService(t *testing.T) (*domain.BalancesService, *mocks.MockLiveService) {
	ctrl := gomock.NewController(t)
	live := mocks.NewMockLiveService(ctrl)
	logger := zap.NewNop().Sugar()
	return domain.NewBalancesService(logger, live), live
}

func TestGetBalancesForIdentities_Success(t *testing.T) {
	svc, live := newTestBalancesService(t)
	ctx := context.Background()

	live.EXPECT().GetBalance(gomock.Any(), "id1").Return(domain.IdentityBalance{
		Id: "id1", Balance: 1000, ValidForTick: 500,
	}, nil)
	live.EXPECT().GetBalance(gomock.Any(), "id2").Return(domain.IdentityBalance{
		Id: "id2", Balance: 2000, ValidForTick: 500,
	}, nil)

	result, err := svc.GetBalancesForIdentities(ctx, []string{"id1", "id2"})
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Order is preserved (each goroutine writes to its own index)
	assert.Equal(t, "id1", result[0].Id)
	assert.Equal(t, int64(1000), result[0].Balance)
	assert.Equal(t, "id2", result[1].Id)
	assert.Equal(t, int64(2000), result[1].Balance)
}

func TestGetBalancesForIdentities_SingleIdentity(t *testing.T) {
	svc, live := newTestBalancesService(t)
	ctx := context.Background()

	live.EXPECT().GetBalance(gomock.Any(), "id1").Return(domain.IdentityBalance{
		Id: "id1", Balance: 500, ValidForTick: 100,
	}, nil)

	result, err := svc.GetBalancesForIdentities(ctx, []string{"id1"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "id1", result[0].Id)
	assert.Equal(t, int64(500), result[0].Balance)
}

func TestGetBalancesForIdentities_UpstreamError(t *testing.T) {
	svc, live := newTestBalancesService(t)
	ctx := context.Background()

	live.EXPECT().GetBalance(gomock.Any(), "id1").Return(domain.IdentityBalance{}, nil).AnyTimes()
	live.EXPECT().GetBalance(gomock.Any(), "id2").Return(domain.IdentityBalance{}, fmt.Errorf("node unavailable")).AnyTimes()

	_, err := svc.GetBalancesForIdentities(ctx, []string{"id1", "id2"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting balance for identity id2")
}

func TestGetBalancesForIdentities_EmptyIdentities(t *testing.T) {
	svc, _ := newTestBalancesService(t)
	ctx := context.Background()

	result, err := svc.GetBalancesForIdentities(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, result)
}
