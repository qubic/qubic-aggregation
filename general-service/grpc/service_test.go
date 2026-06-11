package grpc_test

import (
	"context"
	"fmt"
	"strings"
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

func newTestService(t *testing.T) (
	*generalGrpc.Service,
	*mocks.MockBidServicer,
	*mocks.MockBalancesServicer,
	*mocks.MockAssetsServicer,
	*mocks.MockSmartContractRewardsServicer,
	generalGrpc.PageSizeLimits,
) {
	ctrl := gomock.NewController(t)
	mockBid := mocks.NewMockBidServicer(ctrl)
	mockBalances := mocks.NewMockBalancesServicer(ctrl)
	mockAssets := mocks.NewMockAssetsServicer(ctrl)
	mockSCRewards := mocks.NewMockSmartContractRewardsServicer(ctrl)
	pageSizeLimits := generalGrpc.NewPageSizeLimits(1000, 10, 10000)
	logger := zap.NewNop().Sugar()
	return generalGrpc.NewService(logger, mockBid, mockBalances, mockAssets, mockSCRewards, pageSizeLimits), mockBid, mockBalances, mockAssets, mockSCRewards, pageSizeLimits
}

func TestGetCurrentIpoBids_Success(t *testing.T) {
	svc, mockBid, _, _, _, _ := newTestService(t)

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
					InputSize:   16,
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
	assert.Equal(t, uint32(16), tx.InputSize)
	assert.Equal(t, "data", tx.InputData)
	assert.Equal(t, "sig", tx.Signature)
	assert.True(t, tx.MoneyFlew)
	assert.Equal(t, int64(1000), tx.Bid.Price)
	assert.Equal(t, uint32(5), tx.Bid.Quantity) // uint16 → uint32 promotion
}

func TestGetCurrentIpoBids_TooManyIdentities(t *testing.T) {
	svc, _, _, _, _, _ := newTestService(t)

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
	svc, mockBid, _, _, _, _ := newTestService(t)

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
	svc, mockBid, _, _, _, _ := newTestService(t)

	mockBid.EXPECT().GetCurrentIPOBidTransactions(gomock.Any(), []string{"id1"}).Return(nil, fmt.Errorf("internal failure"))

	_, err := svc.GetCurrentIpoBids(context.Background(), &pb.GetCurrentIpoBidsRequest{Identities: []string{"id1"}})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestGetCurrentIpoBids_EmptyResult(t *testing.T) {
	svc, mockBid, _, _, _, _ := newTestService(t)

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
	svc, _, mockBalances, _, _, _ := newTestService(t)

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
	svc, _, _, _, _, _ := newTestService(t)

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
	svc, _, mockBalances, _, _, _ := newTestService(t)

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
	svc, _, mockBalances, _, _, _ := newTestService(t)

	mockBalances.EXPECT().GetBalancesForIdentities(gomock.Any(), []string{"id1"}).Return(nil, fmt.Errorf("upstream failure"))

	_, err := svc.GetIdentitiesBalances(context.Background(), &pb.GetIdentitiesBalancesRequest{Identities: []string{"id1"}})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestGetIdentitiesBalances_EmptyResult(t *testing.T) {
	svc, _, mockBalances, _, _, _ := newTestService(t)

	mockBalances.EXPECT().GetBalancesForIdentities(gomock.Any(), []string{"id1"}).Return([]domain.IdentityBalance{}, nil)

	resp, err := svc.GetIdentitiesBalances(context.Background(), &pb.GetIdentitiesBalancesRequest{Identities: []string{"id1"}})
	require.NoError(t, err)
	assert.Empty(t, resp.Balances)
}

func TestGetIdentitiesAssets_Success(t *testing.T) {
	svc, _, _, mockAssets, _, _ := newTestService(t)

	mockAssets.EXPECT().GetAssetsForIdentities(gomock.Any(), []string{"id1"}).Return([]domain.IdentityAssets{
		{
			Identity: "id1",
			Ownerships: []domain.AssetOwnership{
				{AssetIssuer: "CFBM", AssetName: "CFB", ManagingContractIndex: 1, NumberOfShares: 1000, TickNumber: 100},
			},
			Possessions: []domain.AssetPossession{
				{AssetIssuer: "CFBM", AssetName: "CFB", ManagingContractIndex: 1, NumberOfShares: 1000, TickNumber: 101},
			},
		},
	}, nil)

	resp, err := svc.GetIdentitiesAssets(context.Background(), &pb.GetIdentitiesAssetsRequest{Identities: []string{"id1"}})
	require.NoError(t, err)
	require.Len(t, resp.IdentityAssets, 1)

	a := resp.IdentityAssets[0]
	assert.Equal(t, "id1", a.Identity)
	require.Len(t, a.Ownerships, 1)
	require.Len(t, a.Possessions, 1)
	assert.Equal(t, "CFBM", a.Ownerships[0].AssetIssuer)
	assert.Equal(t, "CFB", a.Ownerships[0].AssetName)
	assert.Equal(t, uint32(1), a.Ownerships[0].ManagingContractIndex)
	assert.Equal(t, int64(1000), a.Ownerships[0].NumberOfShares)
	assert.Equal(t, uint32(100), a.Ownerships[0].TickNumber)
	assert.Equal(t, int64(1000), a.Possessions[0].NumberOfShares)
	assert.Equal(t, uint32(101), a.Possessions[0].TickNumber)
}

func TestGetIdentitiesAssets_TooManyIdentities(t *testing.T) {
	svc, _, _, _, _, _ := newTestService(t)

	identities := make([]string, 16)
	for i := range identities {
		identities[i] = fmt.Sprintf("id%d", i)
	}

	_, err := svc.GetIdentitiesAssets(context.Background(), &pb.GetIdentitiesAssetsRequest{Identities: identities})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetIdentitiesAssets_EmptyIdentities(t *testing.T) {
	svc, _, _, _, _, _ := newTestService(t)

	_, err := svc.GetIdentitiesAssets(context.Background(), &pb.GetIdentitiesAssetsRequest{Identities: []string{}})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetIdentitiesAssets_ServiceError(t *testing.T) {
	svc, _, _, mockAssets, _, _ := newTestService(t)

	mockAssets.EXPECT().GetAssetsForIdentities(gomock.Any(), []string{"id1"}).Return(nil, fmt.Errorf("upstream failure"))

	_, err := svc.GetIdentitiesAssets(context.Background(), &pb.GetIdentitiesAssetsRequest{Identities: []string{"id1"}})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestGetSmartContractRewards_Success(t *testing.T) {
	svc, _, _, _, mockSCRewards, _ := newTestService(t)

	mockSCRewards.EXPECT().GetRewardsDistributionsForSmartContract(
		gomock.Any(),
		"EAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAVWRF",
		domain.Pagination{
			Offset: 0,
			Size:   10,
		},
	).Return(domain.SmartContractRewardsDistributionsResult{

		TotalHits:               2,
		TotalAllTimeDistributed: 640000,
		ValidForTick:            18000000,
		Distributions: []domain.SmartContractRewardsDistribution{
			{
				TotalAmount:    128000,
				TransferCount:  128,
				AmountPerShare: 1000,
				TickNumber:     12312313,
				Timestamp:      0,
				Epoch:          100,
			},
			{
				TotalAmount:    512000,
				TransferCount:  256,
				AmountPerShare: 2000,
				TickNumber:     12312312,
				Timestamp:      0,
				Epoch:          100,
			},
		},
	}, nil)

	resp, err := svc.GetSmartContractRewards(context.Background(), &pb.GetSmartContractRewardsRequest{
		SmartContractAddress: "EAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAVWRF",
		Pagination: &pb.Pagination{
			Offset: 0,
			Size:   10,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Distributions, 2)

	firstDistribution := resp.Distributions[0]
	require.Equal(t, int64(128000), firstDistribution.TotalAmount)
	require.Equal(t, uint32(128), firstDistribution.TransferCount)
	require.Equal(t, float64(1000), firstDistribution.AmountPerShare)
	require.Equal(t, uint32(12312313), firstDistribution.TickNumber)
	require.Equal(t, int64(0), firstDistribution.Timestamp)
	require.Equal(t, uint32(100), firstDistribution.Epoch)

	secondDistribution := resp.Distributions[1]
	require.Equal(t, int64(512000), secondDistribution.TotalAmount)
	require.Equal(t, uint32(256), secondDistribution.TransferCount)
	require.Equal(t, float64(2000), secondDistribution.AmountPerShare)
	require.Equal(t, uint32(12312312), secondDistribution.TickNumber)
	require.Equal(t, int64(0), secondDistribution.Timestamp)
	require.Equal(t, uint32(100), secondDistribution.Epoch)

	require.Equal(t, uint32(2), resp.Hits.Total)
	require.Equal(t, uint32(0), resp.Hits.From)
	require.Equal(t, uint32(10), resp.Hits.Size)

	require.Equal(t, "EAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAVWRF", resp.SmartContractAddress)
	require.Equal(t, int64(640000), resp.TotalAllTimeDistributed)
	require.Equal(t, uint32(18000000), resp.ValidForTick)
	require.Equal(t, uint32(len(resp.Distributions)), resp.Hits.Total)

}

const validContractAddress = "EAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAVWRF"
const validNonContractAddress = "JVPZVAKNVAAMMFFULSALKYIUMICASIKFYSTCWLIUQBSDBDQUPQUDDCODTVNJ"

func TestGetSmartContractRewards_BadSmartContractAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{name: "empty", address: ""},
		{name: "too short", address: "ABC"},
		{name: "too long", address: validContractAddress + "A"},
		{name: "lowercase", address: strings.ToLower(validContractAddress)},
		{name: "non-letter characters", address: "1" + strings.Repeat("A", 59)},
		{name: "valid identity but not a contract", address: validNonContractAddress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _, _, _ := newTestService(t)

			// Validation fails before the rewards service is touched, so no mock expectation is set.
			_, err := svc.GetSmartContractRewards(context.Background(), &pb.GetSmartContractRewardsRequest{
				SmartContractAddress: tt.address,
				Pagination:           &pb.Pagination{Offset: 0, Size: 10},
			})
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, st.Code())
		})
	}
}

func TestGetSmartContractRewards_InvalidPagination(t *testing.T) {
	svc, _, _, _, _, _ := newTestService(t)

	// Size exceeds the configured maxPageSize (1000) -> validation error before the service is called.
	_, err := svc.GetSmartContractRewards(context.Background(), &pb.GetSmartContractRewardsRequest{
		SmartContractAddress: validContractAddress,
		Pagination:           &pb.Pagination{Offset: 0, Size: 2000},
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetSmartContractRewards_NilPaginationUsesDefaults(t *testing.T) {
	svc, _, _, _, mockSCRewards, _ := newTestService(t)

	// nil pagination must not panic and must fall back to offset 0 / default size 10,
	// and those validated defaults must be what the service is called with.
	mockSCRewards.EXPECT().GetRewardsDistributionsForSmartContract(
		gomock.Any(),
		validContractAddress,
		domain.Pagination{Offset: 0, Size: 10},
	).Return(domain.SmartContractRewardsDistributionsResult{}, nil)

	resp, err := svc.GetSmartContractRewards(context.Background(), &pb.GetSmartContractRewardsRequest{
		SmartContractAddress: validContractAddress,
		Pagination:           nil,
	})
	require.NoError(t, err)
	assert.Equal(t, uint32(0), resp.Hits.From)
	assert.Equal(t, uint32(10), resp.Hits.Size)
}

func TestGetSmartContractRewards_ServiceError(t *testing.T) {
	svc, _, _, _, mockSCRewards, _ := newTestService(t)

	mockSCRewards.EXPECT().GetRewardsDistributionsForSmartContract(
		gomock.Any(), validContractAddress, domain.Pagination{Offset: 0, Size: 10},
	).Return(domain.SmartContractRewardsDistributionsResult{}, fmt.Errorf("elastic unavailable"))

	_, err := svc.GetSmartContractRewards(context.Background(), &pb.GetSmartContractRewardsRequest{
		SmartContractAddress: validContractAddress,
		Pagination:           &pb.Pagination{Offset: 0, Size: 10},
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestGetSmartContractRewards_EmptyResult(t *testing.T) {
	svc, _, _, _, mockSCRewards, _ := newTestService(t)

	mockSCRewards.EXPECT().GetRewardsDistributionsForSmartContract(
		gomock.Any(), validContractAddress, domain.Pagination{Offset: 0, Size: 10},
	).Return(domain.SmartContractRewardsDistributionsResult{
		TotalHits:               0,
		TotalAllTimeDistributed: 0,
		Distributions:           nil,
	}, nil)

	resp, err := svc.GetSmartContractRewards(context.Background(), &pb.GetSmartContractRewardsRequest{
		SmartContractAddress: validContractAddress,
		Pagination:           &pb.Pagination{Offset: 0, Size: 10},
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Distributions)
	assert.Equal(t, uint32(0), resp.Hits.Total)
	assert.Equal(t, int64(0), resp.TotalAllTimeDistributed)
}
