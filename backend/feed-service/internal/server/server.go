// Package server wires the feed gRPC service and operational HTTP endpoints.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/petmatch/petmatch/internal/config"

	feedv1 "github.com/petmatch/petmatch/gen/go/petmatch/feed/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server owns the network listeners and graceful shutdown lifecycle.
type Server struct {
	cfg     config.Config
	feed    feedv1.FeedServiceServer
	logger  *slog.Logger
	workers []Worker
}

// Worker is a background component bound to the server lifecycle.
type Worker interface {
	Run(context.Context) error
}

// New creates a feed-service server.
func New(cfg config.Config, feed feedv1.FeedServiceServer, logger *slog.Logger, workers ...Worker) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		cfg:     cfg,
		feed:    feed,
		logger:  logger,
		workers: workers,
	}
}

// Run starts gRPC and HTTP servers until ctx is cancelled or a server fails.
func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	grpcListener, err := net.Listen("tcp", s.cfg.GRPCAddr)
	if err != nil {
		return err
	}
	httpListener, err := net.Listen("tcp", s.cfg.HTTPAddr)
	if err != nil {
		_ = grpcListener.Close()
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryRecoveryInterceptor(s.logger)),
		grpc.ChainStreamInterceptor(streamRecoveryInterceptor(s.logger)),
	)
	feedv1.RegisterFeedServiceServer(grpcServer, s.feed)

	httpServer := &http.Server{
		Handler:           healthHandler(),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	errCh := make(chan error, 2+len(s.workers))
	go func() {
		s.logger.Info("starting grpc server", "addr", s.cfg.GRPCAddr)
		errCh <- grpcServer.Serve(grpcListener)
	}()
	go func() {
		s.logger.Info("starting http server", "addr", s.cfg.HTTPAddr)
		errCh <- httpServer.Serve(httpListener)
	}()
	for _, worker := range s.workers {
		go func(worker Worker) {
			errCh <- worker.Run(ctx)
		}(worker)
	}

	select {
	case <-ctx.Done():
		return s.shutdown(httpServer, grpcServer)
	case err := <-errCh:
		cancel()
		shutdownErr := s.shutdown(httpServer, grpcServer)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return shutdownErr
	}
}

func (s *Server) shutdown(httpServer *http.Server, grpcServer *grpc.Server) error {
	s.logger.Info("stopping servers")
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	httpErr := httpServer.Shutdown(ctx)
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-ctx.Done():
		grpcServer.Stop()
	}

	if httpErr != nil && !errors.Is(httpErr, http.ErrServerClosed) {
		return httpErr
	}
	return nil
}

func healthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
	return mux
}

func unaryRecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		start := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("grpc unary panic", "method", info.FullMethod, "panic", recovered, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal server error")
			}
			logger.Info("grpc unary request", "method", info.FullMethod, "duration", time.Since(start), "error", err)
		}()
		return handler(ctx, req)
	}
}

func streamRecoveryInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		start := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("grpc stream panic", "method", info.FullMethod, "panic", recovered, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal server error")
			}
			logger.Info("grpc stream request", "method", info.FullMethod, "duration", time.Since(start), "error", err)
		}()
		return handler(srv, stream)
	}
}
