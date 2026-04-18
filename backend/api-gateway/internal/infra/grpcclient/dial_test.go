package grpcclient

import (
	"context"
	"testing"

	"github.com/petmatch/petmatch/internal/app/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryClientInterceptorPropagatesActorMetadata(t *testing.T) {
	interceptor := unaryClientInterceptor(nil)
	ctx := gateway.WithPrincipal(context.Background(), gateway.Principal{ActorID: "profile-1"})

	err := interceptor(ctx, "/petmatch.animal.v1.AnimalService/AddAnimalPhoto", nil, nil, nil,
		func(ctx context.Context, _ string, _ any, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			md, _ := metadata.FromOutgoingContext(ctx)
			values := md.Get("x-actor-id")
			if len(values) != 1 || values[0] != "profile-1" {
				t.Fatalf("x-actor-id metadata = %v, want [profile-1]", values)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
}
