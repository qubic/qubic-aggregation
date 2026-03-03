package clients

import (
	"context"
	"fmt"
	"testing"

	"github.com/qubic/qubic-aggregation/general-service/clients/mocks"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"github.com/qubic/qubic-http/protobuff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func newTestLiveClient(t *testing.T) (*LiveServiceClient, *mocks.MockQubicLiveServiceClient) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockQubicLiveServiceClient(ctrl)
	return &LiveServiceClient{client: mock, logger: zap.NewNop().Sugar()}, mock
}

func TestGetActiveIpos_Success(t *testing.T) {
	lsc, mock := newTestLiveClient(t)
	ctx := context.Background()

	mock.EXPECT().GetActiveIpos(ctx, gomock.Any()).Return(&protobuff.GetActiveIposResponse{
		Ipos: []*protobuff.Ipo{
			{ContractIndex: 1, AssetName: "ASSET1"},
			{ContractIndex: 2, AssetName: "ASSET2"},
		},
	}, nil)

	ipos, err := lsc.GetActiveIpos(ctx)
	require.NoError(t, err)
	require.Len(t, ipos, 2)

	assert.Equal(t, uint32(1), ipos[0].ContractIndex)
	assert.Equal(t, "ASSET1", ipos[0].AssetName)
	assert.Len(t, ipos[0].Address, 60)

	assert.Equal(t, uint32(2), ipos[1].ContractIndex)
	assert.Equal(t, "ASSET2", ipos[1].AssetName)
	assert.Len(t, ipos[1].Address, 60)

	// Addresses are derived from ContractIndexToAddress, verify they match
	expectedAddr1, _ := domain.ContractIndexToAddress(1)
	expectedAddr2, _ := domain.ContractIndexToAddress(2)
	assert.Equal(t, expectedAddr1, ipos[0].Address)
	assert.Equal(t, expectedAddr2, ipos[1].Address)
}

func TestGetActiveIpos_UpstreamError(t *testing.T) {
	lsc, mock := newTestLiveClient(t)
	ctx := context.Background()

	mock.EXPECT().GetActiveIpos(ctx, gomock.Any()).Return(nil, fmt.Errorf("upstream error"))

	_, err := lsc.GetActiveIpos(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting active ipos from live service")
}

func TestGetActiveIpos_EmptyList(t *testing.T) {
	lsc, mock := newTestLiveClient(t)
	ctx := context.Background()

	mock.EXPECT().GetActiveIpos(ctx, gomock.Any()).Return(&protobuff.GetActiveIposResponse{Ipos: nil}, nil)

	ipos, err := lsc.GetActiveIpos(ctx)
	require.NoError(t, err)
	assert.Empty(t, ipos)
}

func TestGetTickInfo_Success(t *testing.T) {
	lsc, mock := newTestLiveClient(t)
	ctx := context.Background()

	mock.EXPECT().GetTickInfo(ctx, gomock.Any()).Return(&protobuff.GetTickInfoResponse{
		TickInfo: &protobuff.TickInfo{
			Tick:        1000,
			Duration:    5,
			Epoch:       10,
			InitialTick: 100,
		},
	}, nil)

	info, err := lsc.GetTickInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint32(1000), info.Tick)
	assert.Equal(t, uint32(5), info.Duration)
	assert.Equal(t, uint32(10), info.Epoch)
	assert.Equal(t, uint32(100), info.InitialTick)
}

func TestGetTickInfo_UpstreamError(t *testing.T) {
	lsc, mock := newTestLiveClient(t)
	ctx := context.Background()

	mock.EXPECT().GetTickInfo(ctx, gomock.Any()).Return(nil, fmt.Errorf("unavailable"))

	_, err := lsc.GetTickInfo(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting tick info from live service")
}

func TestGetContractIpoBids_Success(t *testing.T) {
	lsc, mock := newTestLiveClient(t)
	ctx := context.Background()

	mock.EXPECT().GetContractIpoBids(ctx, gomock.Any()).Return(&protobuff.GetContractIpoBidsResponse{
		BidData: &protobuff.IpoBidData{
			ContractIndex: 1,
			TickNumber:    500,
			Bids: map[int32]*protobuff.IpoBid{
				0: {Identity: "ID_A", Amount: 100},
				1: {Identity: "ID_B", Amount: 200},
			},
		},
	}, nil)

	data, err := lsc.GetContractIpoBids(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), data.ContractIndex)
	assert.Equal(t, uint32(500), data.TickNumber)
	assert.Equal(t, int64(100), data.Bids["ID_A"])
	assert.Equal(t, int64(200), data.Bids["ID_B"])
	assert.Len(t, data.Bids, 2)
}

func TestGetContractIpoBids_UpstreamError(t *testing.T) {
	lsc, mock := newTestLiveClient(t)
	ctx := context.Background()

	mock.EXPECT().GetContractIpoBids(ctx, gomock.Any()).Return(nil, fmt.Errorf("unavailable"))

	_, err := lsc.GetContractIpoBids(ctx, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting contract ipo bids from live service")
}

func TestGetBalance_Success(t *testing.T) {
	lsc, mock := newTestLiveClient(t)
	ctx := context.Background()

	mock.EXPECT().GetBalance(ctx, gomock.Any()).Return(&protobuff.GetBalanceResponse{
		Balance: &protobuff.Balance{
			Id:                         "TESTIDENTITY",
			Balance:                    5000,
			ValidForTick:               1000,
			LatestIncomingTransferTick: 990,
			LatestOutgoingTransferTick: 980,
			IncomingAmount:             10000,
			OutgoingAmount:             5000,
			NumberOfIncomingTransfers:  20,
			NumberOfOutgoingTransfers:  10,
		},
	}, nil)

	balance, err := lsc.GetBalance(ctx, "TESTIDENTITY")
	require.NoError(t, err)
	assert.Equal(t, "TESTIDENTITY", balance.Id)
	assert.Equal(t, int64(5000), balance.Balance)
	assert.Equal(t, uint32(1000), balance.ValidForTick)
	assert.Equal(t, uint32(990), balance.LatestIncomingTransferTick)
	assert.Equal(t, uint32(980), balance.LatestOutgoingTransferTick)
	assert.Equal(t, int64(10000), balance.IncomingAmount)
	assert.Equal(t, int64(5000), balance.OutgoingAmount)
	assert.Equal(t, uint32(20), balance.NumberOfIncomingTransfers)
	assert.Equal(t, uint32(10), balance.NumberOfOutgoingTransfers)
}

func TestGetBalance_UpstreamError(t *testing.T) {
	lsc, mock := newTestLiveClient(t)
	ctx := context.Background()

	mock.EXPECT().GetBalance(ctx, gomock.Any()).Return(nil, fmt.Errorf("node unavailable"))

	_, err := lsc.GetBalance(ctx, "TESTIDENTITY")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requesting identity balance from live service")
}
