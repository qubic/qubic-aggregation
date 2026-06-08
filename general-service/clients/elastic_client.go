package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"go.uber.org/zap"
	"io"
)

type scDividendAggResponse struct {
	Aggregations struct {
		TotalDistributed struct {
			Value float64 `json:"value"`
		} `json:"total_distributed"`

		TotalTicks struct {
			Value uint32 `json:"value"`
		} `json:"total_ticks"`

		ByTick struct {
			Buckets []scTickBucket `json:"buckets"`
		} `json:"by_tick"`
	} `json:"aggregations"`
}

type scTickBucket struct {
	Key      uint32 `json:"key"`       // tickNumber value
	DocCount uint32 `json:"doc_count"` // number of transfers in this tick

	TotalAmount struct {
		Value float64 `json:"value"`
	} `json:"total_amount"`

	TickMeta struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Timestamp int64  `json:"timestamp"`
					Epoch     uint32 `json:"epoch"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	} `json:"tick_meta"`
}

const (
	dividendLogType     = 0
	dividendCategory    = 7
	computorsCount      = 676
	defaultTermsMaxSize = 10_000
)

type ElasticClient struct {
	eventsElasticSearchClient *elasticsearch.Client
	eventsIndex               string
	logger                    *zap.SugaredLogger
}

func NewElasticClient(eventsElasticSearchClient *elasticsearch.Client, eventsIndex string, logger *zap.SugaredLogger) *ElasticClient {
	return &ElasticClient{
		eventsElasticSearchClient: eventsElasticSearchClient,
		eventsIndex:               eventsIndex,
		logger:                    logger,
	}
}

func (c *ElasticClient) GetSmartContractDividendDistributions(ctx context.Context, identity string, pagination domain.Pagination) (*domain.SmartContractRewardsDistributionsResult, error) {

	// terms.size must be at least offset+size so bucket_sort can slice correctly
	termsSize := defaultTermsMaxSize
	if needed := int(pagination.Offset) + int(pagination.Size); needed > termsSize {
		termsSize = needed
	}

	query := map[string]any{
		"size": 0, // we only care about aggregations, not raw hits
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{"term": map[string]any{"source": identity}},
					map[string]any{"term": map[string]any{"logType": dividendLogType}},
					map[string]any{"term": map[string]any{"categories": dividendCategory}},
				},
			},
		},
		"aggs": map[string]any{
			// Sum of ALL amounts across ALL matching docs (no pagination)
			"total_distributed": map[string]any{
				"sum": map[string]any{"field": "amount"},
			},
			// Approximate count of unique ticks — used as total hits for pagination
			"total_ticks": map[string]any{
				"cardinality": map[string]any{
					"field":               "tickNumber",
					"precision_threshold": 40000, // higher = more accurate, more memory
				},
			},
			// One bucket per tick, sorted newest first
			"by_tick": map[string]any{
				"terms": map[string]any{
					"field": "tickNumber",
					"size":  termsSize,
					"order": map[string]any{"_key": "desc"},
				},
				"aggs": map[string]any{
					// Sum of amounts in this tick
					"total_amount": map[string]any{
						"sum": map[string]any{"field": "amount"},
					},
					// Grab timestamp + epoch from one doc in this tick
					"tick_meta": map[string]any{
						"top_hits": map[string]any{
							"size":    1,
							"_source": []string{"timestamp", "epoch"},
						},
					},
					// Offset/size pagination applied to the buckets themselves
					"pagination": map[string]any{
						"bucket_sort": map[string]any{
							"sort": []any{
								map[string]any{"_key": map[string]any{"order": "desc"}},
							},
							"from": pagination.Offset,
							"size": pagination.Size,
						},
					},
				},
			},
		},
	}

	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	res, err := c.eventsElasticSearchClient.Search(
		c.eventsElasticSearchClient.Search.WithContext(ctx),
		c.eventsElasticSearchClient.Search.WithIndex(c.eventsIndex),
		c.eventsElasticSearchClient.Search.WithBody(bytes.NewReader(queryBytes)),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("elasticsearch error [%d]: %s", res.StatusCode, string(bodyBytes))
	}

	var esResp scDividendAggResponse
	if err := json.NewDecoder(res.Body).Decode(&esResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return mapDividendAggResponse(&esResp), nil
}

func mapDividendAggResponse(esResp *scDividendAggResponse) *domain.SmartContractRewardsDistributionsResult {

	result := &domain.SmartContractRewardsDistributionsResult{
		TotalAllTimeDistributed: int64(esResp.Aggregations.TotalDistributed.Value),
		TotalHits:               esResp.Aggregations.TotalTicks.Value,
	}

	for _, bucket := range esResp.Aggregations.ByTick.Buckets {
		var timestamp int64
		var epoch uint32

		if len(bucket.TickMeta.Hits.Hits) > 0 {
			hit := bucket.TickMeta.Hits.Hits[0].Source
			timestamp = hit.Timestamp
			epoch = hit.Epoch
		}

		totalAmount := int64(bucket.TotalAmount.Value)

		result.Distributions = append(result.Distributions, domain.SmartContractRewardsDistribution{
			Epoch:          epoch,
			TickNumber:     bucket.Key,
			TotalAmount:    totalAmount,
			AmountPerShare: float64(totalAmount) / computorsCount,
			TransferCount:  bucket.DocCount,
			Timestamp:      timestamp,
		})
	}

	return result
}
