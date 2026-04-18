package pbconv

import (
	notificationv1 "github.com/petmatch/petmatch/gen/go/petmatch/notification/v1"
	domain "github.com/petmatch/petmatch/internal/domain/notification"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func DeviceTokenToProto(token domain.DeviceToken) *notificationv1.DeviceToken {
	return &notificationv1.DeviceToken{
		DeviceTokenId: token.ID,
		ProfileId:     token.ProfileID,
		Token:         token.Token,
		Platform:      token.Platform,
		Locale:        token.Locale,
		Active:        token.Active,
		CreatedAt:     timestamppb.New(token.CreatedAt),
		UpdatedAt:     timestamppb.New(token.UpdatedAt),
	}
}

func NotificationToProto(n domain.Notification) *notificationv1.Notification {
	out := &notificationv1.Notification{
		NotificationId:     n.ID,
		RecipientProfileId: n.RecipientProfileID,
		Type:               notificationv1.NotificationType(n.Type),
		Title:              n.Title,
		Body:               n.Body,
		Data:               n.Data,
		Status:             notificationv1.NotificationStatus(n.Status),
		CreatedAt:          timestamppb.New(n.CreatedAt),
	}
	for _, channel := range n.Channels {
		out.Channels = append(out.Channels, notificationv1.NotificationChannel(channel))
	}
	if n.ReadAt != nil {
		out.ReadAt = timestamppb.New(*n.ReadAt)
	}
	return out
}

func ChannelsFromProto(channels []notificationv1.NotificationChannel) []domain.Channel {
	out := make([]domain.Channel, 0, len(channels))
	for _, channel := range channels {
		out = append(out, domain.Channel(channel))
	}
	return out
}

func FromProtoType(value notificationv1.NotificationType) domain.Type {
	return domain.Type(value)
}

func FromProtoStatus(value notificationv1.NotificationStatus) domain.Status {
	return domain.Status(value)
}
