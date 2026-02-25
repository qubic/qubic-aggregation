# Shared Module

Common infrastructure for Qubic aggregation services. This module provides reusable components that eliminate boilerplate across services in the monorepo.

## Packages

### `config`

Embeddable configuration structs with [`ardanlabs/conf`](https://github.com/ardanlabs/conf) tags for environment variable parsing.

- `Server` — HTTP, gRPC, and profiling listen addresses, timeouts, message size limits
- `Metrics` — Prometheus namespace and port
- `Upstream` — gRPC addresses for archive-query-service, qubic-http, and status-service

Services embed these structs in their own config and extend with service-specific fields.

### `grpcclient`

Factory for creating gRPC client connections to upstream services.

- `NewConnection(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)` — returns a client connection with insecure credentials by default, extensible via additional dial options

### `grpcserver`

Generalized dual-port gRPC + HTTP/gRPC-Gateway server lifecycle.

- `New(cfg, registerService, registerGateway, errCh, interceptors...)` — starts a gRPC server and an HTTP gateway using callback-based service registration
- `GracefulStop()` — graceful server shutdown
- `GRPCListenAddr()` — returns the resolved gRPC listener address

The `RegisterServiceFunc` and `RegisterGatewayFunc` callbacks decouple the server from specific proto service types, allowing any service to use the same server setup.

### `middleware`

Common gRPC unary server interceptors.

- `LogTechnicalErrorInterceptor` — logs `Internal` and `Unknown` gRPC errors with method name and request context. Follows the `GetInterceptor` pattern from archive-query-service v2.
- `NewMetricsInterceptor(namespace string)` — creates and registers Prometheus gRPC server metrics (request counts, latencies, error rates) and returns the interceptor.

### `pagination`

Pagination validation and defaults.

- `Limits` — defines `MaxPageSize`, `DefaultPageSize`, and `MaxOffset`
- `DefaultLimits()` — returns standard limits (max page size: 1000, default: 10, max offset: 10000)
- `Limits.Normalize(offset, size)` — applies defaults, validates against limits, returns effective offset and size or an error