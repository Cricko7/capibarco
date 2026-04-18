package grpc

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	analyticsv1 "github.com/petmatch/petmatch/gen/go/petmatch/analytics/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
)

type PetmatchServer struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	service *application.Service
}

func NewPetmatchServer(service *application.Service) *PetmatchServer {
	return &PetmatchServer{service: service}
}

func (s *PetmatchServer) TrackEvent(ctx context.Context, req *analyticsv1.TrackEventRequest) (*analyticsv1.TrackEventResponse, error) {
	event, err := eventFromProto(req.GetEvent())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.service.IngestEvent(ctx, event); err != nil && err != application.ErrDuplicateEvent {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &analyticsv1.TrackEventResponse{
		AnalyticsEventId: event.EventID,
		Accepted:         true,
	}, nil
}

func (s *PetmatchServer) BatchTrackEvents(ctx context.Context, req *analyticsv1.BatchTrackEventsRequest) (*analyticsv1.BatchTrackEventsResponse, error) {
	response := &analyticsv1.BatchTrackEventsResponse{
		RejectedEventIds: make([]string, 0),
	}
	for _, item := range req.GetEvents() {
		event, err := eventFromProto(item)
		if err != nil {
			response.RejectedEventIds = append(response.RejectedEventIds, item.GetAnalyticsEventId())
			continue
		}
		if err := s.service.IngestEvent(ctx, event); err != nil && err != application.ErrDuplicateEvent {
			response.RejectedEventIds = append(response.RejectedEventIds, event.EventID)
			continue
		}
		response.AcceptedCount++
	}
	return response, nil
}

func (s *PetmatchServer) GetAnimalStats(ctx context.Context, req *analyticsv1.GetAnimalStatsRequest) (*analyticsv1.GetAnimalStatsResponse, error) {
	if req.GetAnimalId() == "" {
		return nil, status.Error(codes.InvalidArgument, "animal_id is required")
	}
	stats, err := s.collectAnimalStats(ctx, req.GetAnimalId(), req.GetTimeRange(), req.GetBucket(), nil)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.GetAnimalStatsResponse{Stats: stats}, nil
}

func (s *PetmatchServer) QueryMetrics(ctx context.Context, req *analyticsv1.QueryMetricsRequest) (*analyticsv1.QueryMetricsResponse, error) {
	filter := req.GetFilter()
	if filter == nil || filter.GetAnimalId() == "" {
		return nil, status.Error(codes.InvalidArgument, "filter.animal_id is required")
	}
	stats, err := s.collectAnimalStats(ctx, filter.GetAnimalId(), filter.GetTimeRange(), filter.GetBucket(), filter.GetMetrics())
	if err != nil {
		return nil, err
	}
	return &analyticsv1.QueryMetricsResponse{
		Stats: []*analyticsv1.AnimalStats{stats},
		Page:  &commonv1.PageResponse{},
	}, nil
}

func (s *PetmatchServer) StreamAnimalStats(*analyticsv1.StreamAnimalStatsRequest, analyticsv1.AnalyticsService_StreamAnimalStatsServer) error {
	return status.Error(codes.Unimplemented, "stream_animal_stats is not implemented")
}

func (s *PetmatchServer) collectAnimalStats(ctx context.Context, animalID string, timeRange *commonv1.TimeRange, bucket analyticsv1.TimeBucket, filter []analyticsv1.MetricName) (*analyticsv1.AnimalStats, error) {
	from, to, err := resolveTimeRange(timeRange)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	domainBucket := toDomainBucket(bucket)
	metrics, err := s.service.MetricsByBucket(ctx, animalID, from, to, domainBucket)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	points, ctr := metricPointsFromDomain(metrics, filter)
	return &analyticsv1.AnimalStats{
		AnimalId:       animalID,
		Metrics:        points,
		Ctr:            ctr,
		CalculatedAt:   timestamppb.Now(),
		OwnerProfileId: "",
	}, nil
}

func eventFromProto(event *analyticsv1.AnalyticsEvent) (domain.Event, error) {
	if event == nil {
		return domain.Event{}, fmt.Errorf("event is required")
	}
	if event.GetAnimalId() == "" {
		return domain.Event{}, fmt.Errorf("animal_id is required")
	}
	eventID := event.GetAnalyticsEventId()
	if eventID == "" {
		eventID = uuid.NewString()
	}
	occurredAt := time.Now().UTC()
	if ts := event.GetOccurredAt(); ts != nil {
		occurredAt = ts.AsTime().UTC()
	}
	actorID := firstNonEmpty(event.GetActorId(), event.GetOwnerProfileId(), event.GetAnimalId())
	return domain.Event{
		EventID:    eventID,
		ProfileID:  event.GetAnimalId(),
		ActorID:    actorID,
		Type:       toDomainEventType(event.GetType()),
		OccurredAt: occurredAt,
		Metadata:   metadataFromProto(event),
	}, nil
}

func metadataFromProto(event *analyticsv1.AnalyticsEvent) map[string]string {
	metadata := make(map[string]string, len(event.GetDimensions())+5+len(event.GetValues()))
	for key, value := range event.GetDimensions() {
		metadata[key] = value
	}
	if value := event.GetOwnerProfileId(); value != "" {
		metadata["owner_profile_id"] = value
	}
	if value := event.GetFeedCardId(); value != "" {
		metadata["feed_card_id"] = value
	}
	if value := event.GetMatchId(); value != "" {
		metadata["match_id"] = value
	}
	if value := event.GetConversationId(); value != "" {
		metadata["conversation_id"] = value
	}
	for key, value := range event.GetValues() {
		metadata["value."+key] = strconv.FormatFloat(value, 'f', -1, 64)
	}
	return metadata
}

func toDomainEventType(value analyticsv1.AnalyticsEventType) domain.EventType {
	switch value {
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_FEED_CARD_SERVED:
		return domain.EventImpression
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_CARD_OPENED:
		return domain.EventCardOpen
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_SWIPE_LEFT:
		return domain.EventSwipeLeft
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_SWIPE_RIGHT:
		return domain.EventSwipeRight
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_CHAT_STARTED, analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_MESSAGE_SENT:
		return domain.EventChatStart
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_DONATION_SUCCEEDED:
		return domain.EventDonation
	case analyticsv1.AnalyticsEventType_ANALYTICS_EVENT_TYPE_BOOST_ACTIVATED:
		return domain.EventBoost
	default:
		return ""
	}
}

func resolveTimeRange(timeRange *commonv1.TimeRange) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	from := now.Add(-30 * 24 * time.Hour)
	to := now
	if timeRange != nil {
		if ts := timeRange.GetFrom(); ts != nil {
			from = ts.AsTime().UTC()
		}
		if ts := timeRange.GetTo(); ts != nil {
			to = ts.AsTime().UTC()
		}
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid time_range: from is after to")
	}
	return from, to, nil
}

func toDomainBucket(bucket analyticsv1.TimeBucket) domain.BucketSize {
	switch bucket {
	case analyticsv1.TimeBucket_TIME_BUCKET_HOUR:
		return domain.BucketHour
	default:
		return domain.BucketDay
	}
}

func metricPointsFromDomain(metrics []domain.ProfileMetric, filter []analyticsv1.MetricName) ([]*analyticsv1.MetricPoint, float64) {
	allowed := make(map[analyticsv1.MetricName]struct{}, len(filter))
	for _, metric := range filter {
		allowed[metric] = struct{}{}
	}

	var totalViews float64
	var totalImpressions float64
	points := make([]*analyticsv1.MetricPoint, 0)
	for _, item := range metrics {
		start := timestamppb.New(item.Bucket)
		end := timestamppb.New(bucketEnd(item.Bucket, item.Size))

		impressions := float64(item.Counters[domain.EventImpression])
		views := float64(item.Counters[domain.EventView])
		totalImpressions += impressions
		totalViews += views

		points = appendMetricPoint(points, allowed, analyticsv1.MetricName_METRIC_NAME_VIEWS, views, start, end)
		points = appendMetricPoint(points, allowed, analyticsv1.MetricName_METRIC_NAME_CARD_OPENS, float64(item.Counters[domain.EventCardOpen]), start, end)
		points = appendMetricPoint(points, allowed, analyticsv1.MetricName_METRIC_NAME_SWIPE_LEFTS, float64(item.Counters[domain.EventSwipeLeft]), start, end)
		points = appendMetricPoint(points, allowed, analyticsv1.MetricName_METRIC_NAME_SWIPE_RIGHTS, float64(item.Counters[domain.EventSwipeRight]), start, end)
		points = appendMetricPoint(points, allowed, analyticsv1.MetricName_METRIC_NAME_CHAT_STARTS, float64(item.Counters[domain.EventChatStart]), start, end)
		points = appendMetricPoint(points, allowed, analyticsv1.MetricName_METRIC_NAME_DONATIONS, float64(item.Counters[domain.EventDonation]), start, end)
		if impressions > 0 {
			points = appendMetricPoint(points, allowed, analyticsv1.MetricName_METRIC_NAME_CTR, views/impressions, start, end)
		}
	}

	sort.Slice(points, func(i, j int) bool {
		if points[i].GetBucketStart().AsTime().Equal(points[j].GetBucketStart().AsTime()) {
			return points[i].GetMetric() < points[j].GetMetric()
		}
		return points[i].GetBucketStart().AsTime().Before(points[j].GetBucketStart().AsTime())
	})

	if totalImpressions == 0 {
		return points, 0
	}
	return points, totalViews / totalImpressions
}

func appendMetricPoint(points []*analyticsv1.MetricPoint, allowed map[analyticsv1.MetricName]struct{}, metric analyticsv1.MetricName, value float64, start, end *timestamppb.Timestamp) []*analyticsv1.MetricPoint {
	if len(allowed) > 0 {
		if _, ok := allowed[metric]; !ok {
			return points
		}
	}
	return append(points, &analyticsv1.MetricPoint{
		Metric:      metric,
		Value:       value,
		BucketStart: start,
		BucketEnd:   end,
	})
}

func bucketEnd(start time.Time, bucket domain.BucketSize) time.Time {
	switch bucket {
	case domain.BucketHour:
		return start.Add(time.Hour)
	default:
		return start.Add(24 * time.Hour)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
