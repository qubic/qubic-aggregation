# General Service

General-purpose aggregation service. Currently provides IPO bid transaction aggregation and batch identity balance lookups.

## API

### `GetCurrentIpoBids`

**gRPC**: `qubic.aggregation.general.v1.AggregationGeneralService/GetCurrentIpoBids`
**HTTP**: `POST /getCurrentIpoBids`

**Request**: up to 15 identity strings.

```json
{
  "identities": ["IDENTITY_A", "IDENTITY_B"]
}
```

**Response**: bid transactions grouped by IPO contract.

```json
{
  "ipo_transactions": [
    {
      "asset_name": "RANDOM",
      "contract_index": 5,
      "contract_address": "BAAAA...",
      "transactions": [
        {
          "hash": "abc...",
          "source": "IDENTITY_A",
          "destination": "BAAAA...",
          "tick_number": 22150100,
          "amount": 0,
          "input_size": 10,
          "money_flew": true,
          "bid": {
            "price": 10000000,
            "quantity": 1
          }
        }
      ]
    }
  ]
}
```

### `GetIdentitiesBalances`

**gRPC**: `qubic.aggregation.general.v1.AggregationGeneralService/GetIdentitiesBalances`
**HTTP**: `POST /getIdentitiesBalances`

**Request**: up to 15 identity strings.

```json
{
  "identities": ["IDENTITY_A", "IDENTITY_B"]
}
```

**Response**: balance data for each identity.

```json
{
  "balances": [
    {
      "id": "IDENTITY_A",
      "balance": 1000000,
      "valid_for_tick": 22150100,
      "latest_incoming_transfer_tick": 22150090,
      "latest_outgoing_transfer_tick": 22150080,
      "incoming_amount": 5000000,
      "outgoing_amount": 4000000,
      "number_of_incoming_transfers": 10,
      "number_of_outgoing_transfers": 5
    }
  ]
}
```

## How It Works

### IPO Bids

1. Fetches active IPO contract indices from qubic-http (`GetActiveIpos`), cached with configurable TTL.
2. Derives smart contract addresses from contract indices (contract index in lower 32 bits of the 256-bit public key, remaining bits zeroed).
3. Gets current epoch tick bounds from status-service (`GetTickIntervals`), cached with configurable TTL.
4. For each identity, queries archive-query-service (`GetTransactionsForIdentity`) with destination = SC address, tick range = current epoch, amount = 0.
5. Filters results by `input_size == 16` and `amount == 0` to isolate IPO bid transactions.
6. Parses `input_data` (base64) as `ContractIPOBid`: price (int64 LE, 8 bytes) + quantity (uint16 LE, 2 bytes) + padding (6 bytes).

### Identity Balances

1. For each identity, concurrently fetches the balance from qubic-http (`GetBalance`).
2. Returns all balance fields (current balance, transfer ticks, amounts, transfer counts).

## Upstream Dependencies

| Service | Method | Purpose |
|---------|--------|---------|
| qubic-http | `GetActiveIpos` | Active IPO contract indices |
| qubic-http | `GetTickInfo` | Current epoch for tick range |
| qubic-http | `GetBalance` | Identity balance data |
| status-service | `GetTickIntervals` | Epoch tick boundaries |
| archive-query-service | `GetTransactionsForIdentity` | Bid transaction queries |

## Configuration

Environment variable prefix: `QUBIC_AGGREGATION_GENERAL_SERVICE`

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HTTP_HOST` | `0.0.0.0:8000` | HTTP/gRPC-Gateway listen address |
| `SERVER_GRPC_HOST` | `0.0.0.0:8001` | gRPC listen address |
| `METRICS_PORT` | `9999` | Prometheus metrics port |
| `UPSTREAM_QUBIC_HTTP_HOST` | `localhost:8001` | qubic-http gRPC address |
| `UPSTREAM_ARCHIVE_QUERY_SERVICE_HOST` | `localhost:8001` | archive-query-service gRPC address |
| `UPSTREAM_STATUS_SERVICE_HOST` | `localhost:9901` | status-service gRPC address |
| `CACHE_IPO_TTL` | `20m` | Active IPOs cache TTL |
| `CACHE_TICK_INTERVALS_TTL` | `20m` | Tick intervals cache TTL |

## Development

```bash
# Generate protobuf code
make proto-gen

# Generate mocks
make mocks

# Run tests
make test

# Build
go build ./cmd/general-service

# Build Docker image
docker build -t general-service .
```

## Project Structure

```
general-service/
  cmd/general-service/    Entry point, config, bootstrap
  api/                    Proto definitions + generated gRPC/gateway code
  domain/                 Business logic, interfaces, models
  grpc/                   gRPC service handler + server setup
  clients/                Upstream service client wrappers
```