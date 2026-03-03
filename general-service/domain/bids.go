package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"go.uber.org/zap"
)

const ipoCacheKey = "ipo"
const tickIntervalsCacheKey = "tickIntervals"

type BidService struct {
	logger        *zap.SugaredLogger
	liveService   LiveService
	statusService StatusService
	queryService  QueryService

	ipoCache           *ttlcache.Cache[string, []Ipo]
	tickIntervalsCache *ttlcache.Cache[string, map[uint32][]TickInterval]
}

func NewBidService(
	logger *zap.SugaredLogger,
	liveService LiveService,
	statusService StatusService,
	queryService QueryService,
	ipoTTL, tickIntervalsTTL time.Duration,
) *BidService {

	service := BidService{
		logger:        logger,
		liveService:   liveService,
		statusService: statusService,
		queryService:  queryService,
	}

	service.ipoCache = ttlcache.New[string, []Ipo](
		ttlcache.WithTTL[string, []Ipo](ipoTTL),
		ttlcache.WithDisableTouchOnHit[string, []Ipo](), // don't refresh cacheTTL upon getting the item from getter
		ttlcache.WithLoader(ttlcache.LoaderFunc[string, []Ipo](
			func(c *ttlcache.Cache[string, []Ipo], key string) *ttlcache.Item[string, []Ipo] {
				ipos, err := service.liveService.GetActiveIpos(context.Background())
				if err != nil {
					logger.Errorw("failed to load active ipos", "error", err)
					return nil
				}
				return c.Set(key, ipos, ttlcache.DefaultTTL)
			},
		)),
	)

	service.tickIntervalsCache = ttlcache.New[string, map[uint32][]TickInterval](
		ttlcache.WithTTL[string, map[uint32][]TickInterval](tickIntervalsTTL),
		ttlcache.WithDisableTouchOnHit[string, map[uint32][]TickInterval](),
		ttlcache.WithLoader(ttlcache.LoaderFunc[string, map[uint32][]TickInterval](
			func(c *ttlcache.Cache[string, map[uint32][]TickInterval], key string) *ttlcache.Item[string, map[uint32][]TickInterval] {
				intervals, err := service.statusService.GetTickIntervals(context.Background())
				if err != nil {
					logger.Errorw("failed to load tick intervals", "error", err)
					return nil
				}
				return c.Set(key, intervals, ttlcache.DefaultTTL)
			},
		)),
	)

	return &service
}

func (s *BidService) GetCurrentIPOBidTransactions(ctx context.Context, identities []string) ([]IpoBidTransactions, error) {

	ipoItem := s.ipoCache.Get(ipoCacheKey)
	if ipoItem == nil {
		return nil, fmt.Errorf("failed to get active ipos")
	}
	activeIpos := ipoItem.Value()
	if len(activeIpos) == 0 {
		return nil, nil
	}

	tickInfo, err := s.liveService.GetTickInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting tick info: %w", err)
	}

	tickIntervalsItem := s.tickIntervalsCache.Get(tickIntervalsCacheKey)
	if tickIntervalsItem == nil {
		return nil, fmt.Errorf("failed to get tick intervals")
	}
	tickIntervals := tickIntervalsItem.Value()
	epochIntervals, ok := tickIntervals[tickInfo.Epoch]
	if !ok || len(epochIntervals) == 0 {
		return nil, fmt.Errorf("no tick intervals found for current epoch %d", tickInfo.Epoch)
	}
	currentEpochInitialTick, _ := GetEpochIntervalsAbsoluteRange(epochIntervals)

	var activeIposBidTransactions []IpoBidTransactions

	for _, ipo := range activeIpos {

		ipoBidTransactions := IpoBidTransactions{
			AssetName:       ipo.AssetName,
			ContractIndex:   ipo.ContractIndex,
			ContractAddress: ipo.Address,
			Transactions:    []BidTransaction{},
		}

		for _, identity := range identities {
			identityTransactions, err := s.queryService.GetIPOBidTransactionsForIdentity(ctx, identity, ipo.Address, TickInterval{First: currentEpochInitialTick, Last: tickInfo.Tick})
			if err != nil {
				return nil, fmt.Errorf("fetching bid transactions for identity %s on ipo %d: %w", identity, ipo.ContractIndex, err)
			}
			ipoBidTransactions.Transactions = append(ipoBidTransactions.Transactions, identityTransactions...)
		}

		activeIposBidTransactions = append(activeIposBidTransactions, ipoBidTransactions)
	}

	return activeIposBidTransactions, nil
}
