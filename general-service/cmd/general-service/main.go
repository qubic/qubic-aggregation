package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/qubic/qubic-aggregation/general-service/clients"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"github.com/qubic/qubic-aggregation/general-service/grpc"
	"github.com/qubic/qubic-aggregation/shared/config"
	"github.com/qubic/qubic-aggregation/shared/grpcclient"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()

	if err := run(sugar); err != nil {
		sugar.Fatalw("Failed to run service", "error", err)
	}
}

const confPrefix = "QUBIC_AGGREGATION_SERVICE"

func run(logger *zap.SugaredLogger) error {

	var cfg struct {
		Server   config.Server
		Metrics  config.Metrics
		Upstream config.Upstream
		Cache    struct {
			IpoTtl           time.Duration `conf:"default:20m"`
			TickIntervalsTtl time.Duration `conf:"default:20m"`
		}
	}

	help, err := conf.Parse(confPrefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	fmt.Println(conf.String(&cfg))

	liveServiceGrpcConn, err := grpcclient.NewConnection(cfg.Upstream.QubicHttpUrl)
	if err != nil {
		return fmt.Errorf("creating live service client connection: %w", err)
	}
	defer liveServiceGrpcConn.Close()
	liveClient := clients.NewLiveServiceClient(liveServiceGrpcConn, logger.Named("live-service"))

	queryServiceGrpcConn, err := grpcclient.NewConnection(cfg.Upstream.QueryServiceUrl)
	if err != nil {
		return fmt.Errorf("creating query service client connection: %w", err)
	}
	defer queryServiceGrpcConn.Close()
	queryClient := clients.NewQueryServiceClient(queryServiceGrpcConn, logger.Named("query-service"))

	statusServiceGrpcConn, err := grpcclient.NewConnection(cfg.Upstream.StatusServiceUrl)
	if err != nil {
		return fmt.Errorf("creating status service client connection: %w", err)
	}
	defer statusServiceGrpcConn.Close()
	statusClient := clients.NewStatusServiceClient(statusServiceGrpcConn, logger.Named("status-service"))

	bidService := domain.NewBidService(logger.Named("bid-service"), liveClient, statusClient, queryClient, cfg.Cache.IpoTtl, cfg.Cache.TickIntervalsTtl)
	balancesService := domain.NewBalancesService(logger.Named("balances-service"), liveClient)

	grpcService := grpc.NewService(logger.Named("grpc"), bidService, balancesService)

	errChan := make(chan error, 1)

	srv, err := grpc.NewServer(cfg.Server, grpcService, errChan)
	if err != nil {
		return fmt.Errorf("creating grpc server: %w", err)
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"UP"}`))
	})
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Metrics.Port)
		if err := http.ListenAndServe(addr, metricsMux); err != nil {
			errChan <- fmt.Errorf("metrics/health server: %w", err)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		srv.GracefulStop()
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		logger.Infow("shutdown signal received", "signal", sig)
		srv.GracefulStop()
	}

	return nil
}
