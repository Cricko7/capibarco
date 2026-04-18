package grpc

import (
	"context"
	"testing"

	notificationv1 "github.com/petmatch/petmatch/gen/go/petmatch/notification/v1"
	app "github.com/petmatch/petmatch/internal/app/notification"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRegisterDeviceValidatesRequest(t *testing.T) {
	t.Parallel()

	server := NewServer(app.NewService(nil, nil, "notification-service", "notification", nil, nil))

	_, err := server.RegisterDevice(context.Background(), &notificationv1.RegisterDeviceRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unexpected status code: %s", status.Code(err))
	}
}
