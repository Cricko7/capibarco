package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/delivery/grpc/pb"
	"github.com/petmatch/petmatch/internal/domain"
)

type Server struct {
	pb.UnimplementedAnalyticsServiceServer
	service *application.Service
}

func NewServer(service *application.Service) *Server {
	return &Server{service: service}
}

func (s *Server) GetMetrics(ctx context.Context, req *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	from, to, err := parseRange(req.FromRfc3339, req.ToRfc3339)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	bucket := domain.BucketSize(req.Bucket)
	if bucket == "" {
		bucket = domain.BucketHour
	}
	items, err := s.service.MetricsByBucket(ctx, req.ProfileId, from, to, bucket)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	response := &pb.GetMetricsResponse{Items: make([]*pb.ProfileMetric, 0, len(items))}
	for _, item := range items {
		pm := &pb.ProfileMetric{
			ProfileId:    item.ProfileID,
			BucketRfc3339: item.Bucket.Format(time.RFC3339),
			BucketSize:   string(item.Size),
		}
		for eventType, value := range item.Counters {
			pm.Counters = append(pm.Counters, &pb.MetricCounter{EventType: string(eventType), Value: value})
		}
		response.Items = append(response.Items, pm)
	}
	return response, nil
}

func (s *Server) GetExtendedStats(ctx context.Context, req *pb.GetExtendedStatsRequest) (*pb.GetExtendedStatsResponse, error) {
	from, to, err := parseRange(req.FromRfc3339, req.ToRfc3339)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	stats, err := s.service.ExtendedStats(ctx, domain.ExtendedStatsRole(req.Role), req.ProfileId, from, to)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.GetExtendedStatsResponse{
		ProfileId:      stats.ProfileID,
		Views:          stats.Views,
		Impressions:    stats.Impressions,
		Ctr:            stats.CTR,
		CardOpens:      stats.CardOpens,
		ChatStarts:     stats.ChatStarts,
		Donations:      stats.Donations,
		Boosts:         stats.Boosts,
		ProfileChanges: stats.ProfileChanges,
	}, nil
}

func (s *Server) GetRankingFeedback(ctx context.Context, req *pb.GetRankingFeedbackRequest) (*pb.GetRankingFeedbackResponse, error) {
	from, to, err := parseRange(req.FromRfc3339, req.ToRfc3339)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 50
	}
	items, err := s.service.RankingFeedback(ctx, from, to, limit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	response := &pb.GetRankingFeedbackResponse{Items: make([]*pb.RankingFeedback, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, &pb.RankingFeedback{
			ProfileId:      item.ProfileID,
			BucketRfc3339:  item.Bucket.Format(time.RFC3339),
			Ctr:            item.CTR,
			SwipeRightRate: item.SwipeRightRate,
			Engagement:     item.Engagement,
		})
	}
	return response, nil
}

func parseRange(fromRaw, toRaw string) (time.Time, time.Time, error) {
	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from")
	}
	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to")
	}
	return from, to, nil
}
