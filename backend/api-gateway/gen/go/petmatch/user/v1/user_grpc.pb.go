package userv1

import (
	"context"
	"google.golang.org/grpc"
)

type UserServiceServer interface {
	GetProfile(context.Context, *GetProfileRequest) (*GetProfileResponse, error)
	BatchGetProfiles(context.Context, *BatchGetProfilesRequest) (*BatchGetProfilesResponse, error)
	SearchProfiles(context.Context, *SearchProfilesRequest) (*SearchProfilesResponse, error)
	UpdateProfile(context.Context, *UpdateProfileRequest) (*UpdateProfileResponse, error)
	CreateReview(context.Context, *CreateReviewRequest) (*CreateReviewResponse, error)
	UpdateReview(context.Context, *UpdateReviewRequest) (*UpdateReviewResponse, error)
	ListReviews(context.Context, *ListReviewsRequest) (*ListReviewsResponse, error)
	GetReputationSummary(context.Context, *GetReputationSummaryRequest) (*GetReputationSummaryResponse, error)
	mustEmbedUnimplementedUserServiceServer()
}

type UnimplementedUserServiceServer struct{}

func (UnimplementedUserServiceServer) mustEmbedUnimplementedUserServiceServer() {}
func (UnimplementedUserServiceServer) GetProfile(context.Context, *GetProfileRequest) (*GetProfileResponse, error) {
	return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "not implemented")
}
func (UnimplementedUserServiceServer) BatchGetProfiles(context.Context, *BatchGetProfilesRequest) (*BatchGetProfilesResponse, error) {
	return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "not implemented")
}
func (UnimplementedUserServiceServer) SearchProfiles(context.Context, *SearchProfilesRequest) (*SearchProfilesResponse, error) {
	return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "not implemented")
}
func (UnimplementedUserServiceServer) UpdateProfile(context.Context, *UpdateProfileRequest) (*UpdateProfileResponse, error) {
	return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "not implemented")
}
func (UnimplementedUserServiceServer) CreateReview(context.Context, *CreateReviewRequest) (*CreateReviewResponse, error) {
	return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "not implemented")
}
func (UnimplementedUserServiceServer) UpdateReview(context.Context, *UpdateReviewRequest) (*UpdateReviewResponse, error) {
	return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "not implemented")
}
func (UnimplementedUserServiceServer) ListReviews(context.Context, *ListReviewsRequest) (*ListReviewsResponse, error) {
	return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "not implemented")
}
func (UnimplementedUserServiceServer) GetReputationSummary(context.Context, *GetReputationSummaryRequest) (*GetReputationSummaryResponse, error) {
	return nil, grpc.Errorf(grpc.Code(grpc.ErrServerStopped), "not implemented")
}

func RegisterUserServiceServer(s grpc.ServiceRegistrar, srv UserServiceServer) {
	s.RegisterService(&UserService_ServiceDesc, srv)
}

var UserService_ServiceDesc = grpc.ServiceDesc{ServiceName: "petmatch.user.v1.UserService", HandlerType: (*UserServiceServer)(nil), Methods: []grpc.MethodDesc{
	{MethodName: "GetProfile", Handler: _UserService_GetProfile_Handler},
	{MethodName: "BatchGetProfiles", Handler: _UserService_BatchGetProfiles_Handler},
	{MethodName: "SearchProfiles", Handler: _UserService_SearchProfiles_Handler},
	{MethodName: "UpdateProfile", Handler: _UserService_UpdateProfile_Handler},
	{MethodName: "CreateReview", Handler: _UserService_CreateReview_Handler},
	{MethodName: "UpdateReview", Handler: _UserService_UpdateReview_Handler},
	{MethodName: "ListReviews", Handler: _UserService_ListReviews_Handler},
	{MethodName: "GetReputationSummary", Handler: _UserService_GetReputationSummary_Handler},
}}

func _UserService_GetProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, it grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if it == nil {
		return srv.(UserServiceServer).GetProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/petmatch.user.v1.UserService/GetProfile"}
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).GetProfile(ctx, req.(*GetProfileRequest))
	}
	return it(ctx, in, info, h)
}
func _UserService_BatchGetProfiles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, it grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BatchGetProfilesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if it == nil {
		return srv.(UserServiceServer).BatchGetProfiles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/petmatch.user.v1.UserService/BatchGetProfiles"}
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).BatchGetProfiles(ctx, req.(*BatchGetProfilesRequest))
	}
	return it(ctx, in, info, h)
}
func _UserService_SearchProfiles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, it grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchProfilesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if it == nil {
		return srv.(UserServiceServer).SearchProfiles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/petmatch.user.v1.UserService/SearchProfiles"}
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).SearchProfiles(ctx, req.(*SearchProfilesRequest))
	}
	return it(ctx, in, info, h)
}
func _UserService_UpdateProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, it grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if it == nil {
		return srv.(UserServiceServer).UpdateProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/petmatch.user.v1.UserService/UpdateProfile"}
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).UpdateProfile(ctx, req.(*UpdateProfileRequest))
	}
	return it(ctx, in, info, h)
}
func _UserService_CreateReview_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, it grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateReviewRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if it == nil {
		return srv.(UserServiceServer).CreateReview(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/petmatch.user.v1.UserService/CreateReview"}
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).CreateReview(ctx, req.(*CreateReviewRequest))
	}
	return it(ctx, in, info, h)
}
func _UserService_UpdateReview_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, it grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateReviewRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if it == nil {
		return srv.(UserServiceServer).UpdateReview(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/petmatch.user.v1.UserService/UpdateReview"}
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).UpdateReview(ctx, req.(*UpdateReviewRequest))
	}
	return it(ctx, in, info, h)
}
func _UserService_ListReviews_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, it grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListReviewsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if it == nil {
		return srv.(UserServiceServer).ListReviews(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/petmatch.user.v1.UserService/ListReviews"}
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).ListReviews(ctx, req.(*ListReviewsRequest))
	}
	return it(ctx, in, info, h)
}
func _UserService_GetReputationSummary_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, it grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetReputationSummaryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if it == nil {
		return srv.(UserServiceServer).GetReputationSummary(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/petmatch.user.v1.UserService/GetReputationSummary"}
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).GetReputationSummary(ctx, req.(*GetReputationSummaryRequest))
	}
	return it(ctx, in, info, h)
}
