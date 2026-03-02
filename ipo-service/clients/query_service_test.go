package clients

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"testing"

	queryProto "github.com/qubic/archive-query-service/v2/api/archive-query-service/v2"
	"github.com/qubic/qubic-aggregation/ipo-service/clients/mocks"
	"github.com/qubic/qubic-aggregation/ipo-service/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func makeBidInputData(price int64, quantity uint16) string {
	var buf [10]byte
	binary.LittleEndian.PutUint64(buf[0:8], uint64(price))
	binary.LittleEndian.PutUint16(buf[8:10], quantity)
	return base64.StdEncoding.EncodeToString(buf[:])
}

func newTestQueryClient(t *testing.T) (*QueryServiceClient, *mocks.MockArchiveQueryServiceClient) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockArchiveQueryServiceClient(ctrl)
	return &QueryServiceClient{client: mock, logger: zap.NewNop().Sugar()}, mock
}

func TestGetIPOBidTransactionsForIdentity_SinglePage(t *testing.T) {
	qsc, mock := newTestQueryClient(t)
	ctx := context.Background()

	inputData := makeBidInputData(1000, 5)

	mock.EXPECT().GetTransactionsForIdentity(ctx, gomock.Any()).Return(&queryProto.GetTransactionsForIdentityResponse{
		Hits: &queryProto.Hits{Total: 1},
		Transactions: []*queryProto.Transaction{
			{
				Hash:        "hash1",
				Amount:      0,
				Source:      "src",
				Destination: "dst",
				TickNumber:  500,
				Timestamp:   12345,
				InputType:   1,
				InputSize:   10,
				InputData:   inputData,
				Signature:   "sig",
				MoneyFlew:   true,
			},
		},
	}, nil)

	txs, err := qsc.GetIPOBidTransactionsForIdentity(ctx, "identity1", "destination1", domain.TickInterval{First: 100, Last: 500})
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.Equal(t, "hash1", txs[0].Hash)
	assert.Equal(t, int64(1000), txs[0].Bid.Price)
	assert.Equal(t, uint16(5), txs[0].Bid.Quantity)
	assert.Equal(t, "src", txs[0].Source)
	assert.Equal(t, "dst", txs[0].Destination)
	assert.True(t, txs[0].MoneyFlew)
}

func TestGetIPOBidTransactionsForIdentity_FiltersOutNonBidTransactions(t *testing.T) {
	qsc, mock := newTestQueryClient(t)
	ctx := context.Background()

	inputData := makeBidInputData(1000, 5)

	mock.EXPECT().GetTransactionsForIdentity(ctx, gomock.Any()).Return(&queryProto.GetTransactionsForIdentityResponse{
		Hits: &queryProto.Hits{Total: 4},
		Transactions: []*queryProto.Transaction{
			{Hash: "valid", InputSize: 10, Amount: 0, InputData: inputData},
			{Hash: "wrong_size", InputSize: 8, Amount: 0, InputData: "garbage"},
			{Hash: "wrong_amount", InputSize: 10, Amount: 100, InputData: inputData},
			{Hash: "both_wrong", InputSize: 5, Amount: 50, InputData: "garbage"},
		},
	}, nil)

	txs, err := qsc.GetIPOBidTransactionsForIdentity(ctx, "id1", "dst", domain.TickInterval{First: 1, Last: 100})
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.Equal(t, "valid", txs[0].Hash)
}

func TestGetIPOBidTransactionsForIdentity_Pagination(t *testing.T) {
	qsc, mock := newTestQueryClient(t)
	ctx := context.Background()

	inputData := makeBidInputData(500, 1)

	// First page: 1000 results
	page1Txs := make([]*queryProto.Transaction, 1000)
	for i := range page1Txs {
		page1Txs[i] = &queryProto.Transaction{
			Hash: fmt.Sprintf("tx_%d", i), InputSize: 10, InputData: inputData,
		}
	}

	// Second page: 500 results
	page2Txs := make([]*queryProto.Transaction, 500)
	for i := range page2Txs {
		page2Txs[i] = &queryProto.Transaction{
			Hash: fmt.Sprintf("tx_%d", 1000+i), InputSize: 10, InputData: inputData,
		}
	}

	gomock.InOrder(
		mock.EXPECT().GetTransactionsForIdentity(ctx, gomock.Any()).Return(&queryProto.GetTransactionsForIdentityResponse{
			Hits:         &queryProto.Hits{Total: 1500},
			Transactions: page1Txs,
		}, nil),
		mock.EXPECT().GetTransactionsForIdentity(ctx, gomock.Any()).Return(&queryProto.GetTransactionsForIdentityResponse{
			Hits:         &queryProto.Hits{Total: 1500},
			Transactions: page2Txs,
		}, nil),
	)

	txs, err := qsc.GetIPOBidTransactionsForIdentity(ctx, "id1", "dst", domain.TickInterval{First: 1, Last: 2000})
	require.NoError(t, err)
	assert.Len(t, txs, 1500)
}

func TestGetIPOBidTransactionsForIdentity_UpstreamError(t *testing.T) {
	qsc, mock := newTestQueryClient(t)
	ctx := context.Background()

	mock.EXPECT().GetTransactionsForIdentity(ctx, gomock.Any()).Return(nil, fmt.Errorf("connection refused"))

	_, err := qsc.GetIPOBidTransactionsForIdentity(ctx, "id1", "dst", domain.TickInterval{First: 1, Last: 100})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting bid transactions from query service")
}

func TestGetIPOBidTransactionsForIdentity_EmptyResponse(t *testing.T) {
	qsc, mock := newTestQueryClient(t)
	ctx := context.Background()

	mock.EXPECT().GetTransactionsForIdentity(ctx, gomock.Any()).Return(&queryProto.GetTransactionsForIdentityResponse{
		Hits:         &queryProto.Hits{Total: 0},
		Transactions: nil,
	}, nil)

	txs, err := qsc.GetIPOBidTransactionsForIdentity(ctx, "id1", "dst", domain.TickInterval{First: 1, Last: 100})
	require.NoError(t, err)
	assert.Empty(t, txs)
}
