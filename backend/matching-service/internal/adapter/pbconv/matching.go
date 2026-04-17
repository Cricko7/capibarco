// Package pbconv converts between domain types and generated protobuf messages.
package pbconv

import (
	"time"

	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	domain "github.com/petmatch/petmatch/internal/domain/matching"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SwipeToProto converts a domain swipe to protobuf.
func SwipeToProto(s domain.Swipe) *matchingv1.Swipe {
	pb := &matchingv1.Swipe{
		SwipeId:        s.ID,
		ActorId:        s.ActorID,
		ActorIsGuest:   s.ActorIsGuest,
		AnimalId:       s.AnimalID,
		OwnerProfileId: s.OwnerProfileID,
		Direction:      SwipeDirectionToProto(s.Direction),
		SwipedAt:       timestamp(s.SwipedAt),
	}
	if s.FeedCardID != "" {
		pb.FeedCardId = &s.FeedCardID
	}
	if s.FeedSessionID != "" {
		pb.FeedSessionId = &s.FeedSessionID
	}
	return pb
}

// MatchToProto converts a domain match to protobuf.
func MatchToProto(m domain.Match) *matchingv1.Match {
	return &matchingv1.Match{
		MatchId:          m.ID,
		AnimalId:         m.AnimalID,
		AdopterProfileId: m.AdopterProfileID,
		OwnerProfileId:   m.OwnerProfileID,
		ConversationId:   m.ConversationID,
		Status:           MatchStatusToProto(m.Status),
		CreatedAt:        timestamp(m.CreatedAt),
		UpdatedAt:        timestamp(m.UpdatedAt),
	}
}

// SwipeDirectionToProto converts a domain direction to protobuf.
func SwipeDirectionToProto(direction domain.SwipeDirection) matchingv1.SwipeDirection {
	switch direction {
	case domain.SwipeDirectionLeft:
		return matchingv1.SwipeDirection_SWIPE_DIRECTION_LEFT
	case domain.SwipeDirectionRight:
		return matchingv1.SwipeDirection_SWIPE_DIRECTION_RIGHT
	default:
		return matchingv1.SwipeDirection_SWIPE_DIRECTION_UNSPECIFIED
	}
}

// SwipeDirectionFromProto converts a protobuf direction to domain.
func SwipeDirectionFromProto(direction matchingv1.SwipeDirection) domain.SwipeDirection {
	switch direction {
	case matchingv1.SwipeDirection_SWIPE_DIRECTION_LEFT:
		return domain.SwipeDirectionLeft
	case matchingv1.SwipeDirection_SWIPE_DIRECTION_RIGHT:
		return domain.SwipeDirectionRight
	default:
		return domain.SwipeDirectionUnspecified
	}
}

// MatchStatusToProto converts a domain match status to protobuf.
func MatchStatusToProto(status domain.MatchStatus) matchingv1.MatchStatus {
	switch status {
	case domain.MatchStatusActive:
		return matchingv1.MatchStatus_MATCH_STATUS_ACTIVE
	case domain.MatchStatusArchived:
		return matchingv1.MatchStatus_MATCH_STATUS_ARCHIVED
	case domain.MatchStatusBlocked:
		return matchingv1.MatchStatus_MATCH_STATUS_BLOCKED
	default:
		return matchingv1.MatchStatus_MATCH_STATUS_UNSPECIFIED
	}
}

// MatchStatusFromProto converts a protobuf match status to domain.
func MatchStatusFromProto(status matchingv1.MatchStatus) domain.MatchStatus {
	switch status {
	case matchingv1.MatchStatus_MATCH_STATUS_ACTIVE:
		return domain.MatchStatusActive
	case matchingv1.MatchStatus_MATCH_STATUS_ARCHIVED:
		return domain.MatchStatusArchived
	case matchingv1.MatchStatus_MATCH_STATUS_BLOCKED:
		return domain.MatchStatusBlocked
	default:
		return domain.MatchStatusUnspecified
	}
}

func timestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t.UTC())
}
