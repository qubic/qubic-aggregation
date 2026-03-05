package clients

//go:generate mockgen -destination=mocks/mock_query_service.go -package=mocks github.com/qubic/archive-query-service/v2/api/archive-query-service/v2 ArchiveQueryServiceClient

import (
	"context"
	"fmt"
	"strconv"

	queryProto "github.com/qubic/archive-query-service/v2/api/archive-query-service/v2"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type QueryServiceClient struct {
	client queryProto.ArchiveQueryServiceClient
	logger *zap.SugaredLogger
}

func NewQueryServiceClient(conn *grpc.ClientConn, logger *zap.SugaredLogger) *QueryServiceClient {
	return &QueryServiceClient{
		client: queryProto.NewArchiveQueryServiceClient(conn),
		logger: logger,
	}
}

func (qsc *QueryServiceClient) GetIPOBidTransactionsForIdentity(
	ctx context.Context,
	identity string,
	destination string,
	tickInterval domain.TickInterval,
) ([]domain.BidTransaction, error) {

	const pageSize = 1000
	var offset uint32 = 0
	var transactions []domain.BidTransaction

	for {
		resp, err := qsc.client.GetTransactionsForIdentity(ctx, &queryProto.GetTransactionsForIdentityRequest{
			Identity: identity,
			Filters: map[string]string{
				"destination": destination,
				"amount":      "0",
			},
			Ranges: map[string]*queryProto.Range{
				"tickNumber": {
					LowerBound: &queryProto.Range_Gte{Gte: strconv.FormatUint(uint64(tickInterval.First), 10)},
					UpperBound: &queryProto.Range_Lte{Lte: strconv.FormatUint(uint64(tickInterval.Last), 10)},
				},
			},
			Pagination: &queryProto.Pagination{
				Offset: offset,
				Size:   pageSize,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("requesting bid transactions from query service: %w", err)
		}

		for _, tx := range resp.Transactions {
			if tx.InputSize != 16 || tx.Amount != 0 {
				continue
			}

			ipoBid, err := domain.ParseBidInputData(tx.InputData)
			if err != nil {
				return nil, fmt.Errorf("parsing bid input data: %w", err)
			}

			transactions = append(transactions, domain.BidTransaction{
				Hash:        tx.Hash,
				Amount:      tx.Amount,
				Source:      tx.Source,
				Destination: tx.Destination,
				TickNumber:  tx.TickNumber,
				Timestamp:   tx.Timestamp,
				InputType:   tx.InputType,
				InputSize:   tx.InputSize,
				InputData:   tx.InputData,
				Signature:   tx.Signature,
				MoneyFlew:   tx.MoneyFlew,
				Bid:         ipoBid,
			})
		}

		offset += pageSize
		if offset >= resp.Hits.Total {
			break
		}
	}

	return transactions, nil
}
