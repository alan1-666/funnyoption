package grpcx

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type RegisterFunc func(server *grpc.Server)

func Run(ctx context.Context, logger *slog.Logger, serviceName, addr string, register RegisterFunc) error {
	if addr == "" {
		return errors.New("grpc listen address is empty")
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	if register != nil {
		register(grpcServer)
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("grpc service listening", "service", serviceName, "addr", addr)
		serveErr <- grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		shutdownLogger(logger, serviceName)
		gracefulStop(grpcServer)
		return nil
	case err := <-serveErr:
		return err
	}
}

func gracefulStop(server *grpc.Server) {
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		server.Stop()
	}
}

func shutdownLogger(logger *slog.Logger, serviceName string) {
	logger.Info("grpc service shutting down", "service", serviceName)
}
