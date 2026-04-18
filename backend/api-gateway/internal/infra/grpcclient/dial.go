// Package grpcclient contains downstream gRPC clients.
package grpcclient

import (
	"context"
	"fmt"

	"github.com/petmatch/petmatch/internal/config"
	"github.com/petmatch/petmatch/internal/metrics"
	"github.com/petmatch/petmatch/internal/pkg/requestid"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Connections owns downstream client connections.
type Connections struct {
	Auth         *grpc.ClientConn
	Animal       *grpc.ClientConn
	Feed         *grpc.ClientConn
	Matching     *grpc.ClientConn
	Chat         *grpc.ClientConn
	Billing      *grpc.ClientConn
	Analytics    *grpc.ClientConn
	Notification *grpc.ClientConn
}

// DialAll creates client connections for all downstream services.
func DialAll(ctx context.Context, cfg config.GRPCConfig, m *metrics.Metrics) (*Connections, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithUnaryInterceptor(unaryClientInterceptor(m)),
	}
	dial := func(addr string) (*grpc.ClientConn, error) {
		dialCtx, cancel := context.WithTimeout(ctx, cfg.RequestTimeout)
		defer cancel()
		conn, err := grpc.DialContext(dialCtx, addr, opts...)
		if err != nil {
			return nil, fmt.Errorf("dial %s: %w", addr, err)
		}
		return conn, nil
	}
	authConn, err := dial(cfg.AuthAddr)
	if err != nil {
		return nil, err
	}
	animalConn, err := dial(cfg.AnimalAddr)
	if err != nil {
		_ = authConn.Close()
		return nil, err
	}
	feedConn, err := dial(cfg.FeedAddr)
	if err != nil {
		closeMany(authConn, animalConn)
		return nil, err
	}
	matchingConn, err := dial(cfg.MatchingAddr)
	if err != nil {
		closeMany(authConn, animalConn, feedConn)
		return nil, err
	}
	chatConn, err := dial(cfg.ChatAddr)
	if err != nil {
		closeMany(authConn, animalConn, feedConn, matchingConn)
		return nil, err
	}
	billingConn, err := dial(cfg.BillingAddr)
	if err != nil {
		closeMany(authConn, animalConn, feedConn, matchingConn, chatConn)
		return nil, err
	}
	analyticsConn, err := dial(cfg.AnalyticsAddr)
	if err != nil {
		closeMany(authConn, animalConn, feedConn, matchingConn, chatConn, billingConn)
		return nil, err
	}
	conns := &Connections{Auth: authConn, Animal: animalConn, Feed: feedConn, Matching: matchingConn, Chat: chatConn, Billing: billingConn, Analytics: analyticsConn}
	if cfg.NotificationEnabled {
		notificationConn, err := dial(cfg.NotificationAddr)
		if err != nil {
			_ = conns.Close()
			return nil, err
		}
		conns.Notification = notificationConn
	}
	return conns, nil
}

// Close closes all downstream connections.
func (c *Connections) Close() error {
	return closeMany(c.Auth, c.Animal, c.Feed, c.Matching, c.Chat, c.Billing, c.Analytics, c.Notification)
}

func closeMany(conns ...*grpc.ClientConn) error {
	var first error
	for _, conn := range conns {
		if conn == nil {
			continue
		}
		if err := conn.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func unaryClientInterceptor(m *metrics.Metrics) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if rid := requestid.From(ctx); rid != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", rid)
		}
		err := invoker(ctx, method, req, reply, cc, opts...)
		if m != nil {
			m.DownstreamCalls.WithLabelValues(cc.Target(), method, status.Code(err).String()).Inc()
		}
		return err
	}
}
