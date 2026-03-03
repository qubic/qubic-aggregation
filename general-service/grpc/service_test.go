package grpc_test

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/qubic/qubic-aggregation/general-service/api/qubic/aggregation/general/v1"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"github.com/qubic/qubic-aggregation/general-service/domain/mocks"
	generalGrpc "github.com/qubic/qubic-aggregation/general-service/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestService(t *testing.T) (*generalGrpc.Service, *mocks.MockBidServicer, *mocks.MockBalancesServicer) {
	ctrl := gomock.NewController(t)
	mockBid := mocks.NewMockBidServicer(ctrl)
	mockBalances := mocks.NewMockBalancesServicer(ctrl)
	logger := zap.NewNop().Sugar()
	return generalGrpc.NewService(logger, mockBid, mockBalances), mockBid, mockBalances
}

func TestGetCurrentIpoBids_Success(t *testing.T) {
	svc, mockBid, _ := newTestService(t)

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
	svc, _, _ := newTestService(t)

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
	svc, mockBid, _ := newTestService(t)

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
	svc, mockBid, _ := newTestService(t)

	mockBid.EXPECT().GetCurrentIPOBidTransactions(gomock.Any(), []string{"id1"}).Return(nil, fmt.Errorf("internal failure"))

	_, err := svc.GetCurrentIpoBids(context.Background(), &pb.GetCurrentIpoBidsRequest{Identities: []string{"id1"}})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestGetCurrentIpoBids_EmptyResult(t *testing.T) {
	svc, mockBid, _ := newTestService(t)

	mockBid.EXPECT().GetCurrentIPOBidTransactions(gomock.Any(), []string{"id1"}).
		Return([]domain.IpoBidTransactions{
			{AssetName: "A", ContractIndex: 1, ContractAddress: "X", Transactions: []domain.BidTransaction{}},
		}, nil)

	resp, err := svc.GetCurrentIpoBids(context.Background(), &pb.GetCurrentIpoBidsRequest{Identities: []string{"id1"}})
	require.NoError(t, err)
	require.Len(t, resp.IpoTransactions, 1)
	assert.Empty(t, resp.IpoTransactions[0].Transactions)
}

func TestGetIdentitiesBalances_Success(t *testing.T) {
	svc, _, mockBalances := newTestService(t)

	mockBalances.EXPECT().GetBalancesForIdentities(gomock.Any(), []string{"id1", "id2"}).Return([]domain.IdentityBalance{
		{
			Id:                         "id1",
			Balance:                    1000,
			ValidForTick:               500,
			LatestIncomingTransferTick: 490,
			LatestOutgoingTransferTick: 480,
			IncomingAmount:             5000,
			OutgoingAmount:             4000,
			NumberOfIncomingTransfers:  10,
			NumberOfOutgoingTransfers:  5,
		},
		{
			Id:                         "id2",
			Balance:                    2000,
			ValidForTick:               500,
			LatestIncomingTransferTick: 495,
			LatestOutgoingTransferTick: 470,
			IncomingAmount:             3000,
			OutgoingAmount:             1000,
			NumberOfIncomingTransfers:  8,
			NumberOfOutgoingTransfers:  3,
		},
	}, nil)

	resp, err := svc.GetIdentitiesBalances(context.Background(), &pb.GetIdentitiesBalancesRequest{Identities: []string{"id1", "id2"}})
	require.NoError(t, err)
	require.Len(t, resp.Balances, 2)

	b1 := resp.Balances[0]
	assert.Equal(t, "id1", b1.Id)
	assert.Equal(t, int64(1000), b1.Balance)
	assert.Equal(t, uint32(500), b1.ValidForTick)
	assert.Equal(t, uint32(490), b1.LatestIncomingTransferTick)
	assert.Equal(t, uint32(480), b1.LatestOutgoingTransferTick)
	assert.Equal(t, int64(5000), b1.IncomingAmount)
	assert.Equal(t, int64(4000), b1.OutgoingAmount)
	assert.Equal(t, uint32(10), b1.NumberOfIncomingTransfers)
	assert.Equal(t, uint32(5), b1.NumberOfOutgoingTransfers)

	b2 := resp.Balances[1]
	assert.Equal(t, "id2", b2.Id)
	assert.Equal(t, int64(2000), b2.Balance)
}

func TestGetIdentitiesBalances_TooManyIdentities(t *testing.T) {
	svc, _, _ := newTestService(t)

	identities := make([]string, 16)
	for i := range identities {
		identities[i] = fmt.Sprintf("id%d", i)
	}

	_, err := svc.GetIdentitiesBalances(context.Background(), &pb.GetIdentitiesBalancesRequest{Identities: identities})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetIdentitiesBalances_Exactly15Identities(t *testing.T) {
	svc, _, mockBalances := newTestService(t)

	identities := make([]string, 15)
	for i := range identities {
		identities[i] = fmt.Sprintf("id%d", i)
	}

	mockBalances.EXPECT().GetBalancesForIdentities(gomock.Any(), identities).Return(nil, nil)

	resp, err := svc.GetIdentitiesBalances(context.Background(), &pb.GetIdentitiesBalancesRequest{Identities: identities})
	require.NoError(t, err)
	assert.Empty(t, resp.Balances)
}

func TestGetIdentitiesBalances_ServiceError(t *testing.T) {
	svc, _, mockBalances := newTestService(t)

	mockBalances.EXPECT().GetBalancesForIdentities(gomock.Any(), []string{"id1"}).Return(nil, fmt.Errorf("upstream failure"))

	_, err := svc.GetIdentitiesBalances(context.Background(), &pb.GetIdentitiesBalancesRequest{Identities: []string{"id1"}})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestGetIdentitiesBalances_EmptyResult(t *testing.T) {
	svc, _, mockBalances := newTestService(t)

	mockBalances.EXPECT().GetBalancesForIdentities(gomock.Any(), []string{"id1"}).Return([]domain.IdentityBalance{}, nil)

	resp, err := svc.GetIdentitiesBalances(context.Background(), &pb.GetIdentitiesBalancesRequest{Identities: []string{"id1"}})
	require.NoError(t, err)
	assert.Empty(t, resp.Balances)
}
