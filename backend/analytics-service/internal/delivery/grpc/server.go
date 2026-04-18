package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	analyticsv1 "github.com/petmatch/petmatch/gen/go/petmatch/analytics/v1"
	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
	"github.com/petmatch/petmatch/internal/platform"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	logger  *slog.Logger
	service *application.Service
}

func NewServer(logger *slog.Logger, service *application.Service) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{logger: logger, service: service}
}

func (s *Server) TrackEvent(ctx context.Context, req *analyticsv1.TrackEventRequest) (*analyticsv1.TrackEventResponse, error) {
	if req == nil || req.Event == nil {
		return nil, status.Error(codes.InvalidArgument, "event is required")
	}
	e, err := toDomainEvent(req.Event)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event: %v", err)
	}
	if err := s.service.IngestEvent(ctx, e); err != nil {
		if errors.Is(err, application.ErrDuplicateEvent) {
			return &analyticsv1.TrackEventResponse{AnalyticsEventId: req.Event.AnalyticsEventId, Accepted: false}, nil
		}
		return nil, status.Errorf(codes.Internal, "ingest event: %v", err)
	}
	return &analyticsv1.TrackEventResponse{AnalyticsEventId: req.Event.AnalyticsEventId, Accepted: true}, nil
}

func (s *Server) GetAnimalStats(ctx context.Context, req *analyticsv1.GetAnimalStatsRequest) (*analyticsv1.GetAnimalStatsResponse, error) {
	if req == nil || req.AnimalId == "" || req.TimeRange == nil {
		return nil, status.Error(codes.InvalidArgument, "animal_id and time_range are required")
	}
	from := req.TimeRange.GetFrom().AsTime()
	to := req.TimeRange.GetTo().AsTime()
	bucket, err := toDomainBucket(req.Bucket)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "bucket: %v", err)
	}

	stats, err := s.service.ExtendedStats(ctx, domain.RoleOwner, req.AnimalId, from, to)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}
		return nil, status.Errorf(codes.Internal, "stats: %v", err)
	}
	metrics, err := s.service.MetricsByBucket(ctx, req.AnimalId, from, to, bucket)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "metrics: %v", err)
	}
	return &analyticsv1.GetAnimalStatsResponse{Stats: toProtoAnimalStats(req.AnimalId, stats, metrics, req.Bucket)}, nil
}

func (s *Server) BatchTrackEvents(context.Context, *analyticsv1.BatchTrackEventsRequest) (*analyticsv1.BatchTrackEventsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "BatchTrackEvents is not implemented")
}
func (s *Server) QueryMetrics(context.Context, *analyticsv1.QueryMetricsRequest) (*analyticsv1.QueryMetricsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "QueryMetrics is not implemented")
}
func (s *Server) StreamAnimalStats(*analyticsv1.StreamAnimalStatsRequest, grpc.ServerStreamingServer[analyticsv1.AnimalStats]) error {
	return status.Error(codes.Unimplemented, "StreamAnimalStats is not implemented")
}

func ListenAndServe(ctx context.Context, logger *slog.Logger, addr string, service *application.Service, errCh chan<- error) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen grpc: %w", err)
	}
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(requestIDInterceptor, recoveryInterceptor(logger)))
	analyticsv1.RegisterAnalyticsServiceServer(grpcServer, NewServer(logger, service))
	platform.Go(ctx, logger, "grpc-server", func() error {
		logger.InfoContext(ctx, "grpc server listening", slog.String("addr", addr))
		if err := grpcServer.Serve(lis); err != nil {
			return fmt.Errorf("serve grpc: %w", err)
		}
		return nil
	}, errCh)
	return grpcServer, nil
}

func requestIDInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if rid := first(md.Get("x-request-id")); rid != "" {
			_ = rid
		}
	}
	return handler(ctx, req)
}
func recoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorContext(ctx, "grpc panic recovered", slog.String("method", info.FullMethod), slog.Any("panic", r))
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}
func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func toDomainEvent(ev *analyticsv1.AnalyticsEvent) (domain.Event, error) {
	eventType, err := toDomainEventType(ev.GetType())
	if err != nil {
		return domain.Event{}, err
	}
	actor := ev.GetActorId()
	if actor == "" {
		actor = "system"
	}
	occurred := ev.GetOccurredAt().AsTime()
	if occurred.IsZero() {
		occurred = time.Now().UTC()
	}
	return domain.Event{EventID: ev.GetAnalyticsEventId(), ProfileID: ev.GetAnimalId(), ActorID: actor, Type: eventType, OccurredAt: occurred, Metadata: ev.GetDimensions()}, nil
}

func toProtoAnimalStats(animalID string, stats domain.ExtendedStats, metrics []domain.ProfileMetric, bucket analyticsv1.TimeBucket) *analyticsv1.AnimalStats {
	points := make([]*analyticsv1.MetricPoint, 0, len(metrics)*4)
	for _, m := range metrics {
		for eventType, value := range m.Counters {
			metricName, ok := eventTypeToMetric(eventType)
			if !ok {
				continue
			}
			points = append(points, &analyticsv1.MetricPoint{Metric: metricName, Value: float64(value), BucketStart: timestamppb.New(m.Bucket), BucketEnd: timestamppb.New(m.Bucket.Add(durationForBucket(bucket)))})
		}
	}
	return &analyticsv1.AnimalStats{AnimalId: animalID, OwnerProfileId: stats.ProfileID, Metrics: points, Ctr: stats.CTR, CalculatedAt: timestamppb.Now()}
}

func eventTypeToMetric(eventType domain.EventType) (analyticsv1.MetricName, bool) {
	switch eventType {
	case domain.EventView:
		return analyticsv1.MetricName_METRIC_NAME_VIEWS, true
	case domain.EventCardOpen:
		return analyticsv1.MetricName_METRIC_NAME_CARD_OPENS, true
	case domain.EventSwipeLeft:
		return analyticsv1.MetricName_METRIC_NAME_SWIPE_LEFTS, true
	case domain.EventSwipeRight:
		return analyticsv1.MetricName_METRIC_NAME_SWIPE_RIGHTS, true
	case domain.EventChatStart:
		return analyticsv1.MetricName_METRIC_NAME_CHAT_STARTS, true
	case domain.EventDonation:
		return analyticsv1.MetricName_METRIC_NAME_DONATIONS, true
	default:
		return analyticsv1.MetricName_METRIC_NAME_UNSPECIFIED, false
	}
}

func toDomainEventType(v analyticsv1.AnalyticsEventType) (domain.EventType, error) {
	switch v {
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_FEED_CARD_SERVED:
		return domain.EventImpression, nil
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_CARD_OPENED:
		return domain.EventCardOpen, nil
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_SWIPE_LEFT:
		return domain.EventSwipeLeft, nil
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_SWIPE_RIGHT:
		return domain.EventSwipeRight, nil
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_CHAT_STARTED:
		return domain.EventChatStart, nil
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_DONATION_SUCCEEDED:
		return domain.EventDonation, nil
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_BOOST_ACTIVATED:
		return domain.EventBoost, nil
	default:
		return "", fmt.Errorf("unsupported event type %s", v.String())
	}
}

func toDomainBucket(v analyticsv1.TimeBucket) (domain.BucketSize, error) {
	switch v {
	case analyticsv1.TimeBucket_TIME_BUCKET_HOUR:
		return domain.BucketHour, nil
	case analyticsv1.TimeBucket_TIME_BUCKET_DAY:
		return domain.BucketDay, nil
	default:
		return "", fmt.Errorf("unsupported bucket %s", v.String())
	}
}
func durationForBucket(v analyticsv1.TimeBucket) time.Duration {
	if v == analyticsv1.TimeBucket_TIME_BUCKET_DAY {
		return 24 * time.Hour
	}
	return time.Hour
}
