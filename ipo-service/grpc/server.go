package grpc

import (
	"github.com/qubic/qubic-aggregation/shared/config"
	"github.com/qubic/qubic-aggregation/shared/grpcserver"
	"github.com/qubic/qubic-aggregation/shared/middleware"

	pb "github.com/qubic/qubic-aggregation/ipo-service/api/qubic/aggregation/ipo/v1"

	"google.golang.org/grpc"
)

func NewServer(cfg config.Server, service *Service, errChan chan error) (*grpcserver.Server, error) {
	logInterceptor := &middleware.LogTechnicalErrorInterceptor{}
	metricsInterceptor := middleware.NewMetricsInterceptor("ipo_service")

	return grpcserver.New(
		grpcserver.Config{
			GRPCAddr:       cfg.GrpcHost,
			HTTPAddr:       cfg.HttpHost,
			MaxRecvMsgSize: cfg.MaxRecvSizeInMb * 1024 * 1024,
			MaxSendMsgSize: cfg.MaxSendSizeInMb * 1024 * 1024,
		},
		func(srv *grpc.Server) {
			pb.RegisterAggregationIpoServiceServer(srv, service)
		},
		pb.RegisterAggregationIpoServiceHandlerFromEndpoint,
		errChan,
		logInterceptor.GetInterceptor,
		metricsInterceptor,
	)
}
