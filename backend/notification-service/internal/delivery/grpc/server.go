package grpc

import (
	"context"

	"github.com/go-playground/validator/v10"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	notificationv1 "github.com/petmatch/petmatch/gen/go/petmatch/notification/v1"
	"github.com/petmatch/petmatch/internal/adapter/pbconv"
	app "github.com/petmatch/petmatch/internal/app/notification"
	domain "github.com/petmatch/petmatch/internal/domain/notification"
)

type Server struct {
	notificationv1.UnimplementedNotificationServiceServer
	service  *app.Service
	validate *validator.Validate
}

func NewServer(service *app.Service) *Server {
	return &Server{service: service, validate: validator.New()}
}

func (s *Server) RegisterDevice(ctx context.Context, req *notificationv1.RegisterDeviceRequest) (*notificationv1.RegisterDeviceResponse, error) {
	if err := s.validate.Var(req.GetProfileId(), "required"); err != nil {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	if err := s.validate.Var(req.GetToken(), "required"); err != nil {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	if err := s.validate.Var(req.GetPlatform(), "required"); err != nil {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	device, err := s.service.RegisterDevice(ctx, req.GetProfileId(), req.GetToken(), req.GetPlatform(), req.GetLocale())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &notificationv1.RegisterDeviceResponse{DeviceToken: pbconv.DeviceTokenToProto(device)}, nil
}

func (s *Server) UnregisterDevice(ctx context.Context, req *notificationv1.UnregisterDeviceRequest) (*notificationv1.UnregisterDeviceResponse, error) {
	if err := s.validate.Var(req.GetDeviceTokenId(), "required"); err != nil {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	removed, err := s.service.UnregisterDevice(ctx, req.GetDeviceTokenId())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &notificationv1.UnregisterDeviceResponse{Removed: removed}, nil
}

func (s *Server) CreateNotification(ctx context.Context, req *notificationv1.CreateNotificationRequest) (*notificationv1.CreateNotificationResponse, error) {
	if err := s.validate.Var(req.GetRecipientProfileId(), "required"); err != nil {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	notification, err := s.service.CreateNotification(ctx, domain.Notification{
		RecipientProfileID: req.GetRecipientProfileId(),
		Type:               pbconv.FromProtoType(req.GetType()),
		Channels:           pbconv.ChannelsFromProto(req.GetChannels()),
		Title:              req.GetTitle(),
		Body:               req.GetBody(),
		Data:               req.GetData(),
		IdempotencyKey:     req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	return &notificationv1.CreateNotificationResponse{Notification: pbconv.NotificationToProto(notification)}, nil
}

func (s *Server) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	if err := s.validate.Var(req.GetRecipientProfileId(), "required"); err != nil {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	statuses := make([]domain.Status, 0, len(req.GetStatuses()))
	for _, status := range req.GetStatuses() {
		statuses = append(statuses, pbconv.FromProtoStatus(status))
	}
	notifications, next, err := s.service.ListNotifications(ctx, req.GetRecipientProfileId(), statuses, domain.PageRequest{
		PageSize:  req.GetPage().GetPageSize(),
		PageToken: req.GetPage().GetPageToken(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	items := make([]*notificationv1.Notification, 0, len(notifications))
	for _, notification := range notifications {
		items = append(items, pbconv.NotificationToProto(notification))
	}
	return &notificationv1.ListNotificationsResponse{
		Notifications: items,
		Page:          &commonv1.PageResponse{NextPageToken: next},
	}, nil
}

func (s *Server) MarkNotificationRead(ctx context.Context, req *notificationv1.MarkNotificationReadRequest) (*notificationv1.MarkNotificationReadResponse, error) {
	if err := s.validate.Var(req.GetNotificationId(), "required"); err != nil {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	if err := s.validate.Var(req.GetRecipientProfileId(), "required"); err != nil {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	notification, err := s.service.MarkNotificationRead(ctx, req.GetNotificationId(), req.GetRecipientProfileId())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &notificationv1.MarkNotificationReadResponse{Notification: pbconv.NotificationToProto(notification)}, nil
}

func (s *Server) StreamNotifications(req *notificationv1.StreamNotificationsRequest, stream notificationv1.NotificationService_StreamNotificationsServer) error {
	if err := s.validate.Var(req.GetRecipientProfileId(), "required"); err != nil {
		return toStatusError(domain.ErrInvalidArgument)
	}
	notifications, cancel, err := s.service.Subscribe(req.GetRecipientProfileId())
	if err != nil {
		return toStatusError(err)
	}
	defer cancel()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case notification, ok := <-notifications:
			if !ok {
				return nil
			}
			if err := stream.Send(pbconv.NotificationToProto(notification)); err != nil {
				return err
			}
		}
	}
}
