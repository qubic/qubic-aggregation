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
	assetsService   domain.AssetsServicer
}

func NewService(logger *zap.SugaredLogger, bidService domain.BidServicer, balancesService domain.BalancesServicer, assetsService domain.AssetsServicer) *Service {
	return &Service{
		logger:          logger,
		bidsService:     bidService,
		balancesService: balancesService,
		assetsService:   assetsService,
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

func (s *Service) GetIdentitiesAssets(ctx context.Context, req *pb.GetIdentitiesAssetsRequest) (*pb.GetIdentitiesAssetsResponse, error) {
	if len(req.Identities) > 15 {
		return nil, status.Errorf(codes.InvalidArgument, "maximum 15 identities are allowed per query. got: %d", len(req.Identities))
	}

	if len(req.Identities) < 1 {
		return nil, status.Errorf(codes.InvalidArgument, "at least one identity required for this request. got %d", len(req.Identities))
	}

	identitiesAssets, err := s.assetsService.GetAssetsForIdentities(ctx, req.Identities)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting identities assets: %v", err)
	}

	var assets []*pb.IdentityAssets
	for _, ia := range identitiesAssets {
		ownerships := make([]*pb.AssetOwnership, 0, len(ia.Ownerships))
		for _, o := range ia.Ownerships {
			ownerships = append(ownerships, &pb.AssetOwnership{
				AssetIssuer:           o.AssetIssuer,
				AssetName:             o.AssetName,
				ManagingContractIndex: o.ManagingContractIndex,
				NumberOfShares:        o.NumberOfShares,
				TickNumber:            o.TickNumber,
			})
		}
		possessions := make([]*pb.AssetPossession, 0, len(ia.Possessions))
		for _, p := range ia.Possessions {
			possessions = append(possessions, &pb.AssetPossession{
				AssetIssuer:           p.AssetIssuer,
				AssetName:             p.AssetName,
				ManagingContractIndex: p.ManagingContractIndex,
				NumberOfShares:        p.NumberOfShares,
				TickNumber:            p.TickNumber,
			})
		}
		assets = append(assets, &pb.IdentityAssets{
			Identity:    ia.Identity,
			Ownerships:  ownerships,
			Possessions: possessions,
		})
	}

	return &pb.GetIdentitiesAssetsResponse{Assets: assets}, nil
}
