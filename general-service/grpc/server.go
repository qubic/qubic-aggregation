package grpc

import (
	"github.com/qubic/qubic-aggregation/shared/config"
	"github.com/qubic/qubic-aggregation/shared/grpcserver"
	"github.com/qubic/qubic-aggregation/shared/middleware"

	pb "github.com/qubic/qubic-aggregation/general-service/api/qubic/aggregation/general/v1"

	"google.golang.org/grpc"
)

func NewServer(cfg config.Server, service *Service, errChan chan error) (*grpcserver.Server, error) {
	logInterceptor := &middleware.LogTechnicalErrorInterceptor{}
	metricsInterceptor := middleware.NewMetricsInterceptor("general_aggregation_service")

	return grpcserver.New(
		grpcserver.Config{
			GRPCAddr:       cfg.GrpcHost,
			HTTPAddr:       cfg.HttpHost,
			MaxRecvMsgSize: cfg.MaxRecvSizeInMb * 1024 * 1024,
			MaxSendMsgSize: cfg.MaxSendSizeInMb * 1024 * 1024,
		},
		func(srv *grpc.Server) {
			pb.RegisterAggregationGeneralServiceServer(srv, service)
		},
		pb.RegisterAggregationGeneralServiceHandlerFromEndpoint,
		errChan,
		logInterceptor.GetInterceptor,
		metricsInterceptor,
	)
}
