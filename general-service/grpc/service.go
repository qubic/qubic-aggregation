package grpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/qubic/go-node-connector/v2/types"
	pb "github.com/qubic/qubic-aggregation/general-service/api/qubic/aggregation/general/v1"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	pb.UnimplementedAggregationGeneralServiceServer
	logger                      *zap.SugaredLogger
	bidsService                 domain.BidServicer
	balancesService             domain.BalancesServicer
	assetsService               domain.AssetsServicer
	smartContractRewardsService domain.SmartContractRewardsServicer

	pageSizeLimits PageSizeLimits
}

func NewService(logger *zap.SugaredLogger,
	bidService domain.BidServicer,
	balancesService domain.BalancesServicer,
	assetsService domain.AssetsServicer,
	smartContractRewardsService domain.SmartContractRewardsServicer,
	pageSizeLimits PageSizeLimits) *Service {
	return &Service{
		logger:                      logger,
		bidsService:                 bidService,
		balancesService:             balancesService,
		assetsService:               assetsService,
		smartContractRewardsService: smartContractRewardsService,
		pageSizeLimits:              pageSizeLimits,
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

func (s *Service) GetSmartContractRewards(ctx context.Context, req *pb.GetSmartContractRewardsRequest) (*pb.GetSmartContractRewardsResponse, error) {

	if len(req.SmartContractAddress) != 60 {
		return nil, status.Errorf(codes.InvalidArgument, "bad smart contract identity length. expected 60 got %d", len(req.SmartContractAddress))
	}

	if req.SmartContractAddress != strings.ToUpper(req.SmartContractAddress) {
		return nil, status.Errorf(codes.InvalidArgument, "bad smart contract identity. expected all characters to be uppercase")
	}

	err := validateSmartContractIdentity(req.SmartContractAddress)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "bad smart contract identity: %v", err)
	}

	from, size, err := s.pageSizeLimits.ValidatePagination(req.GetPagination())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid pagination: %v", err)
	}

	result, err := s.smartContractRewardsService.GetRewardsDistributionsForSmartContract(ctx, req.SmartContractAddress, domain.Pagination{
		Offset: from,
		Size:   size,
	})
	if err != nil {
		s.logger.Errorw("failed to get rewards distributions", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to get rewards distributions")
	}

	var distributions []*pb.SmartContractRewardsDistribution
	for _, distribution := range result.Distributions {
		distributions = append(distributions, &pb.SmartContractRewardsDistribution{
			TotalAmount:    distribution.TotalAmount,
			TransferCount:  distribution.TransferCount,
			AmountPerShare: distribution.AmountPerShare,
			TickNumber:     distribution.TickNumber,
			Timestamp:      distribution.Timestamp,
			Epoch:          distribution.Epoch,
		})
	}

	return &pb.GetSmartContractRewardsResponse{
		Hits: &pb.Hits{
			Total: result.TotalHits,
			From:  from,
			Size:  size,
		},
		SmartContractAddress:    req.SmartContractAddress,
		TotalAllTimeDistributed: result.TotalAllTimeDistributed,
		Distributions:           distributions,
	}, nil
}

var zero24 [24]byte

func validateSmartContractIdentity(identity string) error {

	id := types.Identity(identity)
	pubKey, err := id.ToPubKey(false)
	if err != nil {
		return fmt.Errorf("converting identity to public key: %w", err)
	}

	// Last 24 bytes of SC public key must be empty
	if !bytes.Equal(pubKey[8:32], zero24[:]) {
		return fmt.Errorf("identity does not belong to a smart contract")
	}

	// First 8 bytes of SC public key represent its index and can be 1 - 1023
	if index := binary.LittleEndian.Uint64(pubKey[0:8]); index < 1 || index > 1023 {
		return fmt.Errorf("identity is not a valid for smart contracts")
	}

	return nil
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

	var identityAssets []*pb.IdentityAssets
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
		identityAssets = append(identityAssets, &pb.IdentityAssets{
			Identity:    ia.Identity,
			Ownerships:  ownerships,
			Possessions: possessions,
		})
	}

	return &pb.GetIdentitiesAssetsResponse{IdentityAssets: identityAssets}, nil
}
