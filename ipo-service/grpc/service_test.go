package grpc_test

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/qubic/qubic-aggregation/ipo-service/api/qubic/aggregation/ipo/v1"
	"github.com/qubic/qubic-aggregation/ipo-service/domain"
	"github.com/qubic/qubic-aggregation/ipo-service/domain/mocks"
	ipogrpc "github.com/qubic/qubic-aggregation/ipo-service/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetCurrentIpoBids_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBid := mocks.NewMockBidServicer(ctrl)
	logger := zap.NewNop().Sugar()
	svc := ipogrpc.NewService(logger, mockBid)

	mockBid.EXPECT().GetCurrentIPOBidTransactions(gomock.Any(), []string{"id1"}).Return([]domain.IpoBidTransactions{
		{
			AssetName:       "ASSET1",
			ContractIndex:   1,
			ContractAddress: "ADDR1",
			Transactions: []domain.BidTransaction{
				{
					Hash:        "hash1",
					Amount:      100,
					Source:      "src",
					Destination: "dst",
					TickNumber:  500,
					Timestamp:   12345,
					InputType:   1,
					InputSize:   10,
					InputData:   "data",
					Signature:   "sig",
					MoneyFlew:   true,
					Bid:         domain.IpoBid{Price: 1000, Quantity: 5},
				},
			},
		},
	}, nil)

	resp, err := svc.GetCurrentIpoBids(context.Background(), &pb.GetCurrentIpoBidsRequest{Identities: []string{"id1"}})
	require.NoError(t, err)
	require.Len(t, resp.IpoTransactions, 1)

	ipo := resp.IpoTransactions[0]
	assert.Equal(t, "ASSET1", ipo.AssetName)
	assert.Equal(t, uint32(1), ipo.ContractIndex)
	assert.Equal(t, "ADDR1", ipo.ContractAddress)
	require.Len(t, ipo.Transactions, 1)

	tx := ipo.Transactions[0]
	assert.Equal(t, "hash1", tx.Hash)
	assert.Equal(t, uint64(100), tx.Amount)
	assert.Equal(t, "src", tx.Source)
	assert.Equal(t, "dst", tx.Destination)
	assert.Equal(t, uint32(500), tx.TickNumber)
	assert.Equal(t, uint64(12345), tx.Timestamp)
	assert.Equal(t, uint32(1), tx.InputType)
	assert.Equal(t, uint32(10), tx.InputSize)
	assert.Equal(t, "data", tx.InputData)
	assert.Equal(t, "sig", tx.Signature)
	assert.True(t, tx.MoneyFlew)
	assert.Equal(t, int64(1000), tx.Bid.Price)
	assert.Equal(t, uint32(5), tx.Bid.Quantity) // uint16 → uint32 promotion
}

func TestGetCurrentIpoBids_TooManyIdentities(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBid := mocks.NewMockBidServicer(ctrl)
	logger := zap.NewNop().Sugar()
	svc := ipogrpc.NewService(logger, mockBid)

	identities := make([]string, 16)
	for i := range identities {
		identities[i] = fmt.Sprintf("id%d", i)
	}

	_, err := svc.GetCurrentIpoBids(context.Background(), &pb.GetCurrentIpoBidsRequest{Identities: identities})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetCurrentIpoBids_Exactly15Identities(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBid := mocks.NewMockBidServicer(ctrl)
	logger := zap.NewNop().Sugar()
	svc := ipogrpc.NewService(logger, mockBid)

	identities := make([]string, 15)
	for i := range identities {
		identities[i] = fmt.Sprintf("id%d", i)
	}

	mockBid.EXPECT().GetCurrentIPOBidTransactions(gomock.Any(), identities).Return(nil, nil)

	resp, err := svc.GetCurrentIpoBids(context.Background(), &pb.GetCurrentIpoBidsRequest{Identities: identities})
	require.NoError(t, err)
	assert.Empty(t, resp.IpoTransactions)
}

func TestGetCurrentIpoBids_BidServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBid := mocks.NewMockBidServicer(ctrl)
	logger := zap.NewNop().Sugar()
	svc := ipogrpc.NewService(logger, mockBid)

	mockBid.EXPECT().GetCurrentIPOBidTransactions(gomock.Any(), []string{"id1"}).Return(nil, fmt.Errorf("internal failure"))

	_, err := svc.GetCurrentIpoBids(context.Background(), &pb.GetCurrentIpoBidsRequest{Identities: []string{"id1"}})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestGetCurrentIpoBids_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBid := mocks.NewMockBidServicer(ctrl)
	logger := zap.NewNop().Sugar()
	svc := ipogrpc.NewService(logger, mockBid)

	mockBid.EXPECT().GetCurrentIPOBidTransactions(gomock.Any(), []string{"id1"}).
		Return([]domain.IpoBidTransactions{
			{AssetName: "A", ContractIndex: 1, ContractAddress: "X", Transactions: []domain.BidTransaction{}},
		}, nil)

	resp, err := svc.GetCurrentIpoBids(context.Background(), &pb.GetCurrentIpoBidsRequest{Identities: []string{"id1"}})
	require.NoError(t, err)
	require.Len(t, resp.IpoTransactions, 1)
	assert.Empty(t, resp.IpoTransactions[0].Transactions)
}
