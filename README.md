# Qubic Aggregation Services

A monorepo of independently deployable Go microservices that aggregate data from existing Qubic infrastructure. Each
service is a gRPC client of one or more upstream services, applies domain-specific computation or transformation, and
exposes the results via its own gRPC + HTTP/REST API.

## Services

| Service                               | Description                                                    | Status      |
|---------------------------------------|----------------------------------------------------------------|-------------|
| [general-service](general-service/)   | IPO bid aggregation and identity balances                      | Implemented |
| [shared](shared/)                     | Common infrastructure (gRPC client/server, config, middleware) | Implemented |

## Architecture

```
                    Consumers (Wallet, Frontend, API)
                                 |
                    +------------v------------+
                    |    Aggregation Layer    |
                    |  +-------------------+  |
                    |  |  general-service  |  |
                    |  +--------+----------+  |
                    |           |             |
                    |  +--------v----------+  |
                    |  |  shared module    |  |
                    |  +-------------------+  |
                    +------------+------------+
                                 |
            +--------------------+---------------------+
            |                    |                     |
  archive-query-service   qubic-http            status-service
   (ES query layer)     (live node data)      (tick intervals)
```

Each service has its own Go module, build artifacts, and Docker image. Services manage their own dependency trees
independently.

## Conventions

| Aspect        | Convention                                                        |
|---------------|-------------------------------------------------------------------|
| Go version    | 1.26                                                              |
| Module naming | `github.com/qubic/qubic-aggregation/{service-name}`               |
| Configuration | `ardanlabs/conf/v3` with env prefix `QUBIC_AGGREGATION_{SERVICE}` |
| Ports         | HTTP/Gateway: 8000, gRPC: 8001, Metrics: 9999                     |
| Testing       | `stretchr/testify` + `go.uber.org/mock`                           |
| Docker        | Multi-stage: `golang` builder, `alpine` runtime                   |

## Adding a New Service

1. Create a new directory at the repo root (e.g., `new-service/`).
2. Initialize a Go module (`github.com/qubic/qubic-aggregation/new-service`).
3. Add dependency on `shared/` module.
4. Follow the internal layout: `cmd/`, `api/`, `domain/`, `grpc/`, `clients/`.
5. Add a Dockerfile and Makefile.