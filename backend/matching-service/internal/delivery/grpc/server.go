package grpc

import (
	"context"
	"time"

	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	"github.com/petmatch/petmatch/internal/adapter/pbconv"
	app "github.com/petmatch/petmatch/internal/app/matching"
	domain "github.com/petmatch/petmatch/internal/domain/matching"
)

// Server implements petmatch.matching.v1.MatchingService.
type Server struct {
	matchingv1.UnimplementedMatchingServiceServer
	service *app.Service
}

// NewServer creates a gRPC matching server.
func NewServer(service *app.Service) *Server {
	return &Server{service: service}
}

// RecordSwipe records a swipe.
func (s *Server) RecordSwipe(ctx context.Context, req *matchingv1.RecordSwipeRequest) (*matchingv1.RecordSwipeResponse, error) {
	principal := req.GetPrincipal()
	if principal == nil || principal.GetActorId() == "" {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	result, err := s.service.RecordSwipe(ctx, app.RecordSwipeCommand{
		ActorID:        principal.GetActorId(),
		ActorIsGuest:   principal.GetIsGuest() || principal.GetActorType() == commonv1.ActorType_ACTOR_TYPE_GUEST,
		AnimalID:       req.GetAnimalId(),
		OwnerProfileID: req.GetOwnerProfileId(),
		Direction:      pbconv.SwipeDirectionFromProto(req.GetDirection()),
		FeedCardID:     req.GetFeedCardId(),
		FeedSessionID:  req.GetFeedSessionId(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	resp := &matchingv1.RecordSwipeResponse{
		Swipe:       pbconv.SwipeToProto(result.Swipe),
		ChatCreated: result.ChatCreated,
	}
	if result.Match != nil {
		resp.Match = pbconv.MatchToProto(*result.Match)
	}
	if result.ConversationID != "" {
		resp.ConversationId = &result.ConversationID
	}
	return resp, nil
}

// GetSwipe returns a swipe.
func (s *Server) GetSwipe(ctx context.Context, req *matchingv1.GetSwipeRequest) (*matchingv1.GetSwipeResponse, error) {
	swipe, err := s.service.GetSwipe(ctx, req.GetSwipeId())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &matchingv1.GetSwipeResponse{Swipe: pbconv.SwipeToProto(swipe)}, nil
}

// ListSwipes returns swipes for an actor.
func (s *Server) ListSwipes(ctx context.Context, req *matchingv1.ListSwipesRequest) (*matchingv1.ListSwipesResponse, error) {
	directions := make([]domain.SwipeDirection, 0, len(req.GetDirections()))
	for _, direction := range req.GetDirections() {
		directions = append(directions, pbconv.SwipeDirectionFromProto(direction))
	}
	result, err := s.service.ListSwipes(ctx, app.ListSwipesQuery{
		ActorID:    req.GetActorId(),
		Directions: directions,
		Page:       pageFromProto(req.GetPage()),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	swipes := make([]*matchingv1.Swipe, 0, len(result.Swipes))
	for _, swipe := range result.Swipes {
		swipes = append(swipes, pbconv.SwipeToProto(swipe))
	}
	return &matchingv1.ListSwipesResponse{Swipes: swipes, Page: pageToProto(result.Page)}, nil
}

// GetMatch returns a match.
func (s *Server) GetMatch(ctx context.Context, req *matchingv1.GetMatchRequest) (*matchingv1.GetMatchResponse, error) {
	match, err := s.service.GetMatch(ctx, req.GetMatchId())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &matchingv1.GetMatchResponse{Match: pbconv.MatchToProto(match)}, nil
}

// ListMatches returns matches for a participant profile.
func (s *Server) ListMatches(ctx context.Context, req *matchingv1.ListMatchesRequest) (*matchingv1.ListMatchesResponse, error) {
	statuses := make([]domain.MatchStatus, 0, len(req.GetStatuses()))
	for _, status := range req.GetStatuses() {
		statuses = append(statuses, pbconv.MatchStatusFromProto(status))
	}
	result, err := s.service.ListMatches(ctx, app.ListMatchesQuery{
		ParticipantProfileID: req.GetParticipantProfileId(),
		Statuses:             statuses,
		Page:                 pageFromProto(req.GetPage()),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	matches := make([]*matchingv1.Match, 0, len(result.Matches))
	for _, match := range result.Matches {
		matches = append(matches, pbconv.MatchToProto(match))
	}
	return &matchingv1.ListMatchesResponse{Matches: matches, Page: pageToProto(result.Page)}, nil
}

// WatchMatches streams match snapshots for a participant.
func (s *Server) WatchMatches(req *matchingv1.WatchMatchesRequest, stream matchingv1.MatchingService_WatchMatchesServer) error {
	if req.GetParticipantProfileId() == "" {
		return toStatusError(domain.ErrInvalidArgument)
	}
	seen := map[string]domain.Match{}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		result, err := s.service.ListMatches(stream.Context(), app.ListMatchesQuery{
			ParticipantProfileID: req.GetParticipantProfileId(),
			Statuses:             []domain.MatchStatus{domain.MatchStatusActive, domain.MatchStatusArchived, domain.MatchStatusBlocked},
			Page:                 app.PageRequest{PageSize: 100},
		})
		if err != nil {
			return toStatusError(err)
		}
		for _, match := range result.Matches {
			if previous, ok := seen[match.ID]; ok && previous.UpdatedAt.Equal(match.UpdatedAt) && previous.Status == match.Status && previous.ConversationID == match.ConversationID {
				continue
			}
			seen[match.ID] = match
			if err := stream.Send(&matchingv1.MatchStreamEvent{Match: pbconv.MatchToProto(match)}); err != nil {
				return err
			}
		}
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
		}
	}
}

func pageFromProto(page *commonv1.PageRequest) app.PageRequest {
	if page == nil {
		return app.PageRequest{}
	}
	return app.PageRequest{PageSize: page.GetPageSize(), PageToken: page.GetPageToken()}
}

func pageToProto(page app.PageResponse) *commonv1.PageResponse {
	resp := &commonv1.PageResponse{NextPageToken: page.NextPageToken}
	if page.TotalSize != nil {
		resp.TotalSize = page.TotalSize
	}
	return resp
}
