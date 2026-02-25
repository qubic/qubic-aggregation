package config

import "time"

type Server struct {
	ReadTimeout     time.Duration `conf:"default:5s"`
	WriteTimeout    time.Duration `conf:"default:5s"`
	ShutdownTimeout time.Duration `conf:"default:5s"`
	HttpHost        string        `conf:"default:0.0.0.0:8000"`
	GrpcHost        string        `conf:"default:0.0.0.0:8001"`
	ProfilingHost   string        `conf:"default:0.0.0.0:8002"`
	MaxRecvSizeInMb int           `conf:"default:1"`
	MaxSendSizeInMb int           `conf:"default:10"`
}

type Metrics struct {
	Namespace string `conf:"default:qubic_aggregation"`
	Port      int    `conf:"default:9999"`
}

type Upstream struct {
	ArchiveQueryServiceHost string `conf:"default:localhost:8001"`
	QubicHttpHost           string `conf:"default:localhost:8001"`
	StatusServiceHost       string `conf:"default:localhost:9901"`
}
