package domain

//go:generate mockgen -destination=mocks/mock_interfaces.go -package=mocks github.com/qubic/qubic-aggregation/general-service/domain LiveService,StatusService,QueryService,ElasticService,BidServicer,BalancesServicer,SmartContractRewardsServicer,AssetsServicer

import (
	"context"
)

type LiveService interface {
	GetActiveIpos(ctx context.Context) ([]Ipo, error)
	GetContractIpoBids(ctx context.Context, contractIndex uint32) (IpoBidData, error)
	GetTickInfo(ctx context.Context) (TickInfo, error)
	GetBalance(ctx context.Context, identity string) (IdentityBalance, error)
	GetAssetsForIdentities(ctx context.Context, identities []string) ([]IdentityAssets, error)
}

type StatusService interface {
	GetTickIntervals(ctx context.Context) (map[uint32][]TickInterval, error)
	GetIngestionPipelineStatus(ctx context.Context) (IngestionPipelineStatus, error)
}

type QueryService interface {
	GetIPOBidTransactionsForIdentity(ctx context.Context, identity string, destination string, tickInterval TickInterval) ([]BidTransaction, error)
}

type ElasticService interface {
	GetSmartContractDividendDistributions(ctx context.Context, identity string, pagination Pagination, maxTick uint32) (*SmartContractRewardsDistributionsResult, error)
}

type BidServicer interface {
	GetCurrentIPOBidTransactions(ctx context.Context, identities []string) ([]IpoBidTransactions, error)
}

type BalancesServicer interface {
	GetBalancesForIdentities(ctx context.Context, identities []string) ([]IdentityBalance, error)
}

type AssetsServicer interface {
	GetAssetsForIdentities(ctx context.Context, identities []string) ([]IdentityAssets, error)
}

type SmartContractRewardsServicer interface {
	GetRewardsDistributionsForSmartContract(ctx context.Context, identity string, pagination Pagination) (SmartContractRewardsDistributionsResult, error)
}
