package middleware

import (
	grpcProm "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

func NewMetricsInterceptor(namespace string) grpc.UnaryServerInterceptor {
	srvMetrics := grpcProm.NewServerMetrics(
		grpcProm.WithServerCounterOptions(grpcProm.WithConstLabels(prometheus.Labels{"namespace": namespace})),
	)
	prometheus.DefaultRegisterer.MustRegister(srvMetrics)
	return srvMetrics.UnaryServerInterceptor()
}
