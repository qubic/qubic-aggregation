package clients

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapDividendAggResponse(t *testing.T) {
	// Mirrors the shape Elasticsearch returns, so the JSON tags are exercised too.
	raw := `{
		"aggregations": {
			"total_distributed": {"value": 1014000},
			"total_ticks": {"value": 2},
			"by_tick": {
				"buckets": [
					{
						"key": 42187300,
						"doc_count": 451,
						"total_amount": {"value": 676000},
						"tick_meta": {"hits": {"hits": [
							{"_source": {"timestamp": 1734429600000, "epoch": 152}}
						]}}
					},
					{
						"key": 42100500,
						"doc_count": 423,
						"total_amount": {"value": 338000},
						"tick_meta": {"hits": {"hits": []}}
					}
				]
			}
		}
	}`

	var esResp scDividendAggResponse
	require.NoError(t, json.Unmarshal([]byte(raw), &esResp))

	result := mapDividendAggResponse(&esResp)

	assert.Equal(t, int64(1014000), result.TotalAllTimeDistributed)
	assert.Equal(t, uint32(2), result.TotalHits)
	require.Len(t, result.Distributions, 2)

	// First bucket: full metadata; amountPerShare = 676000 / 676 = 1000.
	d0 := result.Distributions[0]
	assert.Equal(t, uint32(42187300), d0.TickNumber)
	assert.Equal(t, int64(676000), d0.TotalAmount)
	assert.Equal(t, uint32(451), d0.TransferCount)
	assert.Equal(t, float64(1000), d0.AmountPerShare)
	assert.Equal(t, int64(1734429600000), d0.Timestamp)
	assert.Equal(t, uint32(152), d0.Epoch)

	// Second bucket: empty tick_meta hits -> timestamp/epoch fall back to zero;
	// amountPerShare = 338000 / 676 = 500.
	d1 := result.Distributions[1]
	assert.Equal(t, uint32(42100500), d1.TickNumber)
	assert.Equal(t, int64(338000), d1.TotalAmount)
	assert.Equal(t, uint32(423), d1.TransferCount)
	assert.Equal(t, float64(500), d1.AmountPerShare)
	assert.Equal(t, int64(0), d1.Timestamp)
	assert.Equal(t, uint32(0), d1.Epoch)
}

func TestMapDividendAggResponse_Empty(t *testing.T) {
	var esResp scDividendAggResponse

	result := mapDividendAggResponse(&esResp)

	assert.Equal(t, int64(0), result.TotalAllTimeDistributed)
	assert.Equal(t, uint32(0), result.TotalHits)
	assert.Empty(t, result.Distributions)
}
