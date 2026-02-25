package grpcserver

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

type Config struct {
	GRPCAddr       string
	HTTPAddr       string
	MaxRecvMsgSize int
	MaxSendMsgSize int
}

type RegisterServiceFunc func(srv *grpc.Server)

type RegisterGatewayFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

type Server struct {
	grpcServer *grpc.Server
	grpcLis    net.Listener
}

func New(cfg Config, registerService RegisterServiceFunc, registerGateway RegisterGatewayFunc, errChan chan error, interceptors ...grpc.UnaryServerInterceptor) (*Server, error) {

	srv := grpc.NewServer(
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.MaxSendMsgSize),
		grpc.ChainUnaryInterceptor(interceptors...),
	)

	registerService(srv)
	reflection.Register(srv)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return nil, fmt.Errorf("listening on grpc addr: %w", err)
	}

	go func() {
		if err := srv.Serve(lis); err != nil {
			errChan <- fmt.Errorf("grpc serve: %w", err)
		}
	}()

	if cfg.HTTPAddr != "" && registerGateway != nil {
		go func() {
			mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{EmitDefaultValues: true, EmitUnpopulated: true},
			}))
			opts := []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithDefaultCallOptions(
					grpc.MaxCallRecvMsgSize(cfg.MaxRecvMsgSize),
					grpc.MaxCallSendMsgSize(cfg.MaxSendMsgSize),
				),
			}
			if err := registerGateway(context.Background(), mux, cfg.GRPCAddr, opts); err != nil {
				errChan <- fmt.Errorf("registering http gateway: %w", err)
				return
			}
			if err := http.ListenAndServe(cfg.HTTPAddr, mux); err != nil {
				errChan <- fmt.Errorf("http serve: %w", err)
			}
		}()
	}

	return &Server{grpcServer: srv, grpcLis: lis}, nil
}

func (s *Server) GracefulStop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

func (s *Server) GRPCListenAddr() net.Addr {
	return s.grpcLis.Addr()
}
