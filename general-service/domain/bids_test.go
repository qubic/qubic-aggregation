package domain_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/qubic/qubic-aggregation/general-service/domain"
	"github.com/qubic/qubic-aggregation/general-service/domain/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func newTestBidService(t *testing.T) (
	*domain.BidService,
	*mocks.MockLiveService,
	*mocks.MockStatusService,
	*mocks.MockQueryService,
) {
	ctrl := gomock.NewController(t)
	live := mocks.NewMockLiveService(ctrl)
	status := mocks.NewMockStatusService(ctrl)
	query := mocks.NewMockQueryService(ctrl)
	logger := zap.NewNop().Sugar()

	svc := domain.NewBidService(logger, live, status, query, time.Minute, time.Minute)
	return svc, live, status, query
}

func TestGetCurrentIPOBidTransactions_HappyPath_SingleIdentitySingleIPO(t *testing.T) {
	svc, live, status, query := newTestBidService(t)
	ctx := context.Background()

	ipos := []domain.Ipo{{ContractIndex: 1, AssetName: "ASSET1", Address: "ADDR1"}}
	live.EXPECT().GetActiveIpos(gomock.Any()).Return(ipos, nil)
	live.EXPECT().GetTickInfo(gomock.Any()).Return(domain.TickInfo{Tick: 500, Epoch: 10, InitialTick: 100}, nil)
	status.EXPECT().GetTickIntervals(gomock.Any()).Return(map[uint32][]domain.TickInterval{
		10: {{First: 100, Last: 300}, {First: 301, Last: 500}},
	}, nil)

	txs := []domain.BidTransaction{{Hash: "hash1", Amount: 42, TickNumber: 150}}
	query.EXPECT().GetIPOBidTransactionsForIdentity(gomock.Any(), "id1", "ADDR1", domain.TickInterval{First: 100, Last: 500}).Return(txs, nil)

	result, err := svc.GetCurrentIPOBidTransactions(ctx, []string{"id1"})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "ASSET1", result[0].AssetName)
	assert.Equal(t, uint32(1), result[0].ContractIndex)
	assert.Equal(t, "ADDR1", result[0].ContractAddress)
	require.Len(t, result[0].Transactions, 1)
	assert.Equal(t, "hash1", result[0].Transactions[0].Hash)
}

func TestGetCurrentIPOBidTransactions_HappyPath_MultipleIdentitiesMultipleIPOs(t *testing.T) {
	svc, live, status, query := newTestBidService(t)
	ctx := context.Background()

	ipos := []domain.Ipo{
		{ContractIndex: 1, AssetName: "ASSET1", Address: "ADDR1"},
		{ContractIndex: 2, AssetName: "ASSET2", Address: "ADDR2"},
	}
	live.EXPECT().GetActiveIpos(gomock.Any()).Return(ipos, nil)
	live.EXPECT().GetTickInfo(gomock.Any()).Return(domain.TickInfo{Tick: 500, Epoch: 10, InitialTick: 100}, nil)
	status.EXPECT().GetTickIntervals(gomock.Any()).Return(map[uint32][]domain.TickInterval{
		10: {{First: 100, Last: 500}},
	}, nil)

	interval := domain.TickInterval{First: 100, Last: 500}

	// 2 IPOs x 2 identities = 4 query calls
	query.EXPECT().GetIPOBidTransactionsForIdentity(gomock.Any(), "id1", "ADDR1", interval).
		Return([]domain.BidTransaction{{Hash: "tx1"}}, nil)
	query.EXPECT().GetIPOBidTransactionsForIdentity(gomock.Any(), "id2", "ADDR1", interval).
		Return([]domain.BidTransaction{{Hash: "tx2"}}, nil)
	query.EXPECT().GetIPOBidTransactionsForIdentity(gomock.Any(), "id1", "ADDR2", interval).
		Return([]domain.BidTransaction{{Hash: "tx3"}}, nil)
	query.EXPECT().GetIPOBidTransactionsForIdentity(gomock.Any(), "id2", "ADDR2", interval).
		Return([]domain.BidTransaction{{Hash: "tx4"}}, nil)

	result, err := svc.GetCurrentIPOBidTransactions(ctx, []string{"id1", "id2"})
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Len(t, result[0].Transactions, 2)
	assert.Len(t, result[1].Transactions, 2)
}

func TestGetCurrentIPOBidTransactions_NoActiveIPOs(t *testing.T) {
	svc, live, _, _ := newTestBidService(t)
	ctx := context.Background()

	live.EXPECT().GetActiveIpos(gomock.Any()).Return([]domain.Ipo{}, nil)

	result, err := svc.GetCurrentIPOBidTransactions(ctx, []string{"id1"})
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetCurrentIPOBidTransactions_IPOCacheLoadFails(t *testing.T) {
	svc, live, _, _ := newTestBidService(t)
	ctx := context.Background()

	live.EXPECT().GetActiveIpos(gomock.Any()).Return(nil, fmt.Errorf("connection refused"))

	_, err := svc.GetCurrentIPOBidTransactions(ctx, []string{"id1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get active ipos")
}

func TestGetCurrentIPOBidTransactions_GetTickInfoFails(t *testing.T) {
	svc, live, _, _ := newTestBidService(t)
	ctx := context.Background()

	live.EXPECT().GetActiveIpos(gomock.Any()).Return([]domain.Ipo{{ContractIndex: 1, AssetName: "A", Address: "X"}}, nil)
	live.EXPECT().GetTickInfo(gomock.Any()).Return(domain.TickInfo{}, fmt.Errorf("tick info unavailable"))

	_, err := svc.GetCurrentIPOBidTransactions(ctx, []string{"id1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "getting tick info")
}

func TestGetCurrentIPOBidTransactions_TickIntervalsCacheLoadFails(t *testing.T) {
	svc, live, status, _ := newTestBidService(t)
	ctx := context.Background()

	live.EXPECT().GetActiveIpos(gomock.Any()).Return([]domain.Ipo{{ContractIndex: 1, AssetName: "A", Address: "X"}}, nil)
	live.EXPECT().GetTickInfo(gomock.Any()).Return(domain.TickInfo{Tick: 500, Epoch: 10}, nil)
	status.EXPECT().GetTickIntervals(gomock.Any()).Return(nil, fmt.Errorf("status unavailable"))

	_, err := svc.GetCurrentIPOBidTransactions(ctx, []string{"id1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get tick intervals")
}

func TestGetCurrentIPOBidTransactions_NoIntervalsForCurrentEpoch(t *testing.T) {
	svc, live, status, _ := newTestBidService(t)
	ctx := context.Background()

	live.EXPECT().GetActiveIpos(gomock.Any()).Return([]domain.Ipo{{ContractIndex: 1, AssetName: "A", Address: "X"}}, nil)
	live.EXPECT().GetTickInfo(gomock.Any()).Return(domain.TickInfo{Tick: 500, Epoch: 10}, nil)
	status.EXPECT().GetTickIntervals(gomock.Any()).Return(map[uint32][]domain.TickInterval{
		9: {{First: 1, Last: 99}}, // different epoch
	}, nil)

	_, err := svc.GetCurrentIPOBidTransactions(ctx, []string{"id1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no tick intervals found for current epoch 10")
}

func TestGetCurrentIPOBidTransactions_QueryServiceFails(t *testing.T) {
	svc, live, status, query := newTestBidService(t)
	ctx := context.Background()

	live.EXPECT().GetActiveIpos(gomock.Any()).Return([]domain.Ipo{{ContractIndex: 1, AssetName: "A", Address: "ADDR1"}}, nil)
	live.EXPECT().GetTickInfo(gomock.Any()).Return(domain.TickInfo{Tick: 500, Epoch: 10, InitialTick: 100}, nil)
	status.EXPECT().GetTickIntervals(gomock.Any()).Return(map[uint32][]domain.TickInterval{
		10: {{First: 100, Last: 500}},
	}, nil)
	query.EXPECT().GetIPOBidTransactionsForIdentity(gomock.Any(), "id1", "ADDR1", gomock.Any()).
		Return(nil, fmt.Errorf("query timeout"))

	_, err := svc.GetCurrentIPOBidTransactions(ctx, []string{"id1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching bid transactions for identity id1")
}
