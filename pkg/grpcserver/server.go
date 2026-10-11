package grpcserver

import (
	"context"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	shutdownTimeout      = 15 * time.Second
	defaultCheckInterval = 5 * time.Second
	checkTimeout         = 2 * time.Second
)

// HealthChecker reports whether a dependency the service cannot work
// without (the database) is reachable.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

type Server struct {
	grpcServer    *grpc.Server
	health        *health.Server
	checker       HealthChecker
	checkInterval time.Duration
	logger        *slog.Logger
}

func NewServer(logger *slog.Logger, checker HealthChecker) *Server {
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(recoverPanics(logger)),
	)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	return &Server{
		grpcServer:    grpcServer,
		health:        healthServer,
		checker:       checker,
		checkInterval: defaultCheckInterval,
		logger:        logger,
	}
}

// Run serves until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context, lis net.Listener) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.grpcServer.Serve(lis) }()
	go s.watchHealth(ctx)

	s.logger.Info("grpc server listening", "addr", lis.Addr().String())

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	s.logger.Info("shutdown signal received, draining requests")
	s.health.Shutdown()

	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		s.logger.Info("grpc server stopped cleanly")
	case <-time.After(shutdownTimeout):
		s.grpcServer.Stop()
		s.logger.Warn("shutdown timed out, forced stop")
	}

	return nil
}

func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl any) {
	s.grpcServer.RegisterService(desc, impl)
}

// watchHealth keeps the reported status in step with the dependency.
func (s *Server) watchHealth(ctx context.Context) {
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		s.health.SetServingStatus("", s.probe(ctx))

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) probe(ctx context.Context) healthpb.HealthCheckResponse_ServingStatus {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	if err := s.checker.Ping(ctx); err != nil {
		s.logger.Warn("dependency check failed", "error", err)
		return healthpb.HealthCheckResponse_NOT_SERVING
	}
	return healthpb.HealthCheckResponse_SERVING
}
