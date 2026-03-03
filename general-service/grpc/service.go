package grpc

import (
	"context"

	pb "github.com/qubic/qubic-aggregation/general-service/api/qubic/aggregation/general/v1"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	pb.UnimplementedAggregationGeneralServiceServer
	logger          *zap.SugaredLogger
	bidsService     domain.BidServicer
	balancesService domain.BalancesServicer
}

func NewService(logger *zap.SugaredLogger, bidService domain.BidServicer, balancesService domain.BalancesServicer) *Service {
	return &Service{
		logger:          logger,
		bidsService:     bidService,
		balancesService: balancesService,
	}
}

func (s *Service) GetCurrentIpoBids(ctx context.Context, req *pb.GetCurrentIpoBidsRequest) (*pb.GetCurrentIpoBidsResponse, error) {

	if len(req.Identities) > 15 {
		return nil, status.Errorf(codes.InvalidArgument, "maximum 15 identities are allowed per query. got: %d", len(req.Identities))
	}

	if len(req.Identities) < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "at least one identity required for this request. got %d", len(req.Identities))
	}

	ipoBidTransactions, err := s.bidsService.GetCurrentIPOBidTransactions(ctx, req.Identities)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting ipo bid transactions: %v", err)
	}

	var activeIpoTransactions []*pb.IpoBidTransactions
	for _, ipo := range ipoBidTransactions {
		ipoTransactions := pb.IpoBidTransactions{
			AssetName:       ipo.AssetName,
			ContractIndex:   ipo.ContractIndex,
			ContractAddress: ipo.ContractAddress,
			Transactions:    []*pb.BidTransaction{},
		}

		for _, transaction := range ipo.Transactions {
			ipoTransactions.Transactions = append(ipoTransactions.Transactions, &pb.BidTransaction{
				Hash:        transaction.Hash,
				Amount:      transaction.Amount,
				Source:      transaction.Source,
				Destination: transaction.Destination,
				TickNumber:  transaction.TickNumber,
				Timestamp:   transaction.Timestamp,
				InputType:   transaction.InputType,
				InputSize:   transaction.InputSize,
				InputData:   transaction.InputData,
				Signature:   transaction.Signature,
				MoneyFlew:   transaction.MoneyFlew,
				Bid: &pb.IpoBid{
					Price:    transaction.Bid.Price,
					Quantity: uint32(transaction.Bid.Quantity),
				},
			})
		}
		activeIpoTransactions = append(activeIpoTransactions, &ipoTransactions)
	}

	return &pb.GetCurrentIpoBidsResponse{IpoTransactions: activeIpoTransactions}, nil
}

func (s *Service) GetIdentitiesBalances(ctx context.Context, req *pb.GetIdentitiesBalancesRequest) (*pb.GetIdentitiesBalancesResponse, error) {
	if len(req.Identities) > 15 {
		return nil, status.Errorf(codes.InvalidArgument, "maximum 15 identities are allowed per query. got: %d", len(req.Identities))
	}

	if len(req.Identities) < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "at least one identity required for this request. got %d", len(req.Identities))
	}

	identitiesBalances, err := s.balancesService.GetBalancesForIdentities(ctx, req.Identities)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting identities balances: %v", err)
	}

	var balances []*pb.IdentityBalance
	for _, balance := range identitiesBalances {
		balances = append(balances, &pb.IdentityBalance{
			Id:                         balance.Id,
			Balance:                    balance.Balance,
			ValidForTick:               balance.ValidForTick,
			LatestIncomingTransferTick: balance.LatestIncomingTransferTick,
			LatestOutgoingTransferTick: balance.LatestOutgoingTransferTick,
			IncomingAmount:             balance.IncomingAmount,
			OutgoingAmount:             balance.OutgoingAmount,
			NumberOfIncomingTransfers:  balance.NumberOfIncomingTransfers,
			NumberOfOutgoingTransfers:  balance.NumberOfOutgoingTransfers,
		})
	}

	return &pb.GetIdentitiesBalancesResponse{Balances: balances}, nil
}
