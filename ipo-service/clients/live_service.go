package clients

//go:generate mockgen -destination=mocks/mock_live_service.go -package=mocks github.com/qubic/qubic-http/protobuff QubicLiveServiceClient

import (
	"context"
	"fmt"

	"github.com/qubic/qubic-aggregation/ipo-service/domain"
	"github.com/qubic/qubic-http/protobuff"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type LiveServiceClient struct {
	client protobuff.QubicLiveServiceClient
	logger *zap.SugaredLogger
}

func NewLiveServiceClient(conn *grpc.ClientConn, logger *zap.SugaredLogger) *LiveServiceClient {
	return &LiveServiceClient{
		client: protobuff.NewQubicLiveServiceClient(conn),
		logger: logger,
	}
}

func (lsc *LiveServiceClient) GetActiveIpos(ctx context.Context) ([]domain.Ipo, error) {
	activeIposResponse, err := lsc.client.GetActiveIpos(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("requesting active ipos from live service: %w", err)
	}

	var activeIpos []domain.Ipo
	for _, ipo := range activeIposResponse.Ipos {
		address, err := domain.ContractIndexToAddress(ipo.ContractIndex)
		if err != nil {
			return nil, fmt.Errorf("deriving contract address from contract index: %w", err)
		}
		activeIpos = append(activeIpos, domain.Ipo{
			ContractIndex: ipo.ContractIndex,
			AssetName:     ipo.AssetName,
			Address:       address,
		})
	}

	return activeIpos, nil
}

func (lsc *LiveServiceClient) GetContractIpoBids(ctx context.Context, contractIndex uint32) (domain.IpoBidData, error) {
	contractIpoBidsResponse, err := lsc.client.GetContractIpoBids(ctx, &protobuff.GetContractIpoBidsRequest{ContractIndex: contractIndex})
	if err != nil {
		return domain.IpoBidData{}, fmt.Errorf("requesting contract ipo bids from live service: %w", err)
	}

	ipoBids := make(map[string]int64)
	for _, ipoBid := range contractIpoBidsResponse.BidData.Bids {
		ipoBids[ipoBid.Identity] = ipoBid.Amount
	}

	return domain.IpoBidData{
		ContractIndex: contractIpoBidsResponse.BidData.ContractIndex,
		TickNumber:    contractIpoBidsResponse.BidData.TickNumber,
		Bids:          ipoBids,
	}, nil
}

func (lsc *LiveServiceClient) GetTickInfo(ctx context.Context) (domain.TickInfo, error) {
	tickInfoResponse, err := lsc.client.GetTickInfo(ctx, nil)
	if err != nil {
		return domain.TickInfo{}, fmt.Errorf("requesting tick info from live service: %w", err)
	}

	return domain.TickInfo{
		Tick:        tickInfoResponse.TickInfo.Tick,
		Duration:    tickInfoResponse.TickInfo.Duration,
		Epoch:       tickInfoResponse.TickInfo.Epoch,
		InitialTick: tickInfoResponse.TickInfo.InitialTick,
	}, nil
}
