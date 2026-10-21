module github.com/qubic/qubic-aggregation/general-service

go 1.26.10

require (
	github.com/ardanlabs/conf/v3 v3.13.0
	github.com/elastic/go-elasticsearch/v8 v8.19.7
	github.com/google/gnostic v0.7.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.32.0
	github.com/jellydator/ttlcache/v3 v3.4.1
	github.com/prometheus/client_golang v1.25.0
	github.com/qubic/archive-query-service/v2 v2.0.0-20260907065350-e6f1aa4a7a31
	github.com/qubic/go-data-publisher/status-service v1.6.0
	github.com/qubic/go-node-connector/v2 v2.4.0
	github.com/qubic/qubic-aggregation/shared v0.1.1
	github.com/qubic/qubic-http v0.11.1
	github.com/stretchr/testify v1.12.1
	go.uber.org/mock v0.6.0
	go.uber.org/zap v1.28.0
	golang.org/x/sync v0.24.0
	google.golang.org/genproto/googleapis/api v0.0.0-20261019180522-ce908d29299b
	google.golang.org/grpc v1.84.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.26.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.6.5 // indirect
	github.com/consensys/gnark-crypto v0.22.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.11.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus v1.1.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3 // indirect
	github.com/linckode/circl v1.3.72 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.3 // indirect
	github.com/prometheus/common v0.72.0 // indirect
	github.com/prometheus/procfs v0.22.0 // indirect
	github.com/qubic/go-schnorrq v1.2.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.58.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261012220534-4341edf3f941 // indirect
)

replace github.com/qubic/qubic-aggregation/shared => ../shared
