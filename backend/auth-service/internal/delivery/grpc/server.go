package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hackathon/authsvc/internal/domain"
	"github.com/hackathon/authsvc/internal/usecase"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/status"
)

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error)   { return json.Marshal(v) }
func (jsonCodec) Unmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
func (jsonCodec) Name() string                    { return "json" }

// Codec returns the JSON codec used by the manually registered gRPC service.
func Codec() encoding.Codec {
	return jsonCodec{}
}

// Server exposes auth usecases over gRPC.
type Server struct {
	auth *usecase.AuthService
}

type authServiceServer interface {
	Register(context.Context, *RegisterRequest) (*AuthResponse, error)
	Login(context.Context, *LoginRequest) (*AuthResponse, error)
	RefreshToken(context.Context, *RefreshTokenRequest) (*AuthResponse, error)
	ForgotPassword(context.Context, *ForgotPasswordRequest) (*EmptyResponse, error)
	ResetPassword(context.Context, *ResetPasswordRequest) (*EmptyResponse, error)
	ValidateToken(context.Context, *ValidateTokenRequest) (*ValidateTokenResponse, error)
	GetUserInfo(context.Context, *GetUserInfoRequest) (*GetUserInfoResponse, error)
	Authorize(context.Context, *AuthorizeRequest) (*AuthorizeResponse, error)
}

// NewServer creates a gRPC auth server.
func NewServer(auth *usecase.AuthService) *Server {
	return &Server{auth: auth}
}

// RegisterAuthServiceServer registers the service implementation.
func RegisterAuthServiceServer(registrar gogrpc.ServiceRegistrar, server *Server) {
	registrar.RegisterService(&AuthServiceDesc, server)
}

// AuthServiceDesc describes the gRPC service.
var AuthServiceDesc = gogrpc.ServiceDesc{
	ServiceName: "auth.v1.AuthService",
	HandlerType: (*authServiceServer)(nil),
	Methods: []gogrpc.MethodDesc{
		{MethodName: "Register", Handler: registerHandler},
		{MethodName: "Login", Handler: loginHandler},
		{MethodName: "RefreshToken", Handler: refreshHandler},
		{MethodName: "ForgotPassword", Handler: forgotPasswordHandler},
		{MethodName: "ResetPassword", Handler: resetPasswordHandler},
		{MethodName: "ValidateToken", Handler: validateTokenHandler},
		{MethodName: "GetUserInfo", Handler: getUserInfoHandler},
		{MethodName: "Authorize", Handler: authorizeHandler},
	},
	Streams:  []gogrpc.StreamDesc{},
	Metadata: "proto/auth/v1/auth.proto",
}

func registerHandler(srv any, ctx context.Context, dec func(any) error, interceptor gogrpc.UnaryServerInterceptor) (any, error) {
	var req RegisterRequest
	if err := dec(&req); err != nil {
		return nil, err
	}
	info := &gogrpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.v1.AuthService/Register"}
	handler := func(ctx context.Context, request any) (any, error) {
		return srv.(*Server).Register(ctx, request.(*RegisterRequest))
	}
	if interceptor == nil {
		return handler(ctx, &req)
	}
	return interceptor(ctx, &req, info, handler)
}

func loginHandler(srv any, ctx context.Context, dec func(any) error, interceptor gogrpc.UnaryServerInterceptor) (any, error) {
	var req LoginRequest
	if err := dec(&req); err != nil {
		return nil, err
	}
	info := &gogrpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.v1.AuthService/Login"}
	handler := func(ctx context.Context, request any) (any, error) {
		return srv.(*Server).Login(ctx, request.(*LoginRequest))
	}
	if interceptor == nil {
		return handler(ctx, &req)
	}
	return interceptor(ctx, &req, info, handler)
}

func refreshHandler(srv any, ctx context.Context, dec func(any) error, interceptor gogrpc.UnaryServerInterceptor) (any, error) {
	var req RefreshTokenRequest
	if err := dec(&req); err != nil {
		return nil, err
	}
	info := &gogrpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.v1.AuthService/RefreshToken"}
	handler := func(ctx context.Context, request any) (any, error) {
		return srv.(*Server).RefreshToken(ctx, request.(*RefreshTokenRequest))
	}
	if interceptor == nil {
		return handler(ctx, &req)
	}
	return interceptor(ctx, &req, info, handler)
}

func forgotPasswordHandler(srv any, ctx context.Context, dec func(any) error, interceptor gogrpc.UnaryServerInterceptor) (any, error) {
	var req ForgotPasswordRequest
	if err := dec(&req); err != nil {
		return nil, err
	}
	info := &gogrpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.v1.AuthService/ForgotPassword"}
	handler := func(ctx context.Context, request any) (any, error) {
		return srv.(*Server).ForgotPassword(ctx, request.(*ForgotPasswordRequest))
	}
	if interceptor == nil {
		return handler(ctx, &req)
	}
	return interceptor(ctx, &req, info, handler)
}

func resetPasswordHandler(srv any, ctx context.Context, dec func(any) error, interceptor gogrpc.UnaryServerInterceptor) (any, error) {
	var req ResetPasswordRequest
	if err := dec(&req); err != nil {
		return nil, err
	}
	info := &gogrpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.v1.AuthService/ResetPassword"}
	handler := func(ctx context.Context, request any) (any, error) {
		return srv.(*Server).ResetPassword(ctx, request.(*ResetPasswordRequest))
	}
	if interceptor == nil {
		return handler(ctx, &req)
	}
	return interceptor(ctx, &req, info, handler)
}

func validateTokenHandler(srv any, ctx context.Context, dec func(any) error, interceptor gogrpc.UnaryServerInterceptor) (any, error) {
	var req ValidateTokenRequest
	if err := dec(&req); err != nil {
		return nil, err
	}
	info := &gogrpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.v1.AuthService/ValidateToken"}
	handler := func(ctx context.Context, request any) (any, error) {
		return srv.(*Server).ValidateToken(ctx, request.(*ValidateTokenRequest))
	}
	if interceptor == nil {
		return handler(ctx, &req)
	}
	return interceptor(ctx, &req, info, handler)
}

func getUserInfoHandler(srv any, ctx context.Context, dec func(any) error, interceptor gogrpc.UnaryServerInterceptor) (any, error) {
	var req GetUserInfoRequest
	if err := dec(&req); err != nil {
		return nil, err
	}
	info := &gogrpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.v1.AuthService/GetUserInfo"}
	handler := func(ctx context.Context, request any) (any, error) {
		return srv.(*Server).GetUserInfo(ctx, request.(*GetUserInfoRequest))
	}
	if interceptor == nil {
		return handler(ctx, &req)
	}
	return interceptor(ctx, &req, info, handler)
}

func authorizeHandler(srv any, ctx context.Context, dec func(any) error, interceptor gogrpc.UnaryServerInterceptor) (any, error) {
	var req AuthorizeRequest
	if err := dec(&req); err != nil {
		return nil, err
	}
	info := &gogrpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.v1.AuthService/Authorize"}
	handler := func(ctx context.Context, request any) (any, error) {
		return srv.(*Server).Authorize(ctx, request.(*AuthorizeRequest))
	}
	if interceptor == nil {
		return handler(ctx, &req)
	}
	return interceptor(ctx, &req, info, handler)
}

func (s *Server) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	out, err := s.auth.Register(ctx, usecase.RegisterInput{TenantID: req.TenantId, Email: req.Email, Password: req.Password, IP: req.Ip})
	if err != nil {
		return nil, mapError(err)
	}
	return authResponse(out), nil
}

func (s *Server) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	out, err := s.auth.Login(ctx, usecase.LoginInput{TenantID: req.TenantId, Email: req.Email, Password: req.Password, IP: req.Ip})
	if err != nil {
		return nil, mapError(err)
	}
	return authResponse(out), nil
}

func (s *Server) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*AuthResponse, error) {
	out, err := s.auth.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, mapError(err)
	}
	return authResponse(out), nil
}

func (s *Server) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) (*EmptyResponse, error) {
	if err := s.auth.ForgotPassword(ctx, req.TenantId, req.Email, req.Ip); err != nil {
		return nil, mapError(err)
	}
	return &EmptyResponse{}, nil
}

func (s *Server) ResetPassword(ctx context.Context, req *ResetPasswordRequest) (*EmptyResponse, error) {
	if err := s.auth.ResetPassword(ctx, req.TenantId, req.ResetToken, req.NewPassword, req.Ip); err != nil {
		return nil, mapError(err)
	}
	return &EmptyResponse{}, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, error) {
	claims, err := s.auth.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return nil, mapError(err)
	}
	return claimsResponse(claims, true), nil
}

func (s *Server) GetUserInfo(ctx context.Context, req *GetUserInfoRequest) (*GetUserInfoResponse, error) {
	user, err := s.auth.GetUserInfo(ctx, req.AccessToken)
	if err != nil {
		return nil, mapError(err)
	}
	return &GetUserInfoResponse{User: userResponse(user)}, nil
}

func (s *Server) Authorize(ctx context.Context, req *AuthorizeRequest) (*AuthorizeResponse, error) {
	claims, ok, err := s.auth.Authorize(ctx, req.AccessToken, req.Permission)
	if err != nil {
		return nil, mapError(err)
	}
	return &AuthorizeResponse{Allowed: ok, Claims: claimsResponse(claims, true)}, nil
}

func authResponse(out usecase.AuthOutput) *AuthResponse {
	return &AuthResponse{
		User:         userResponse(out.User),
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		ExpiresAt:    out.ExpiresAt.Format(time.RFC3339),
	}
}

func userResponse(user domain.User) *User {
	return &User{
		Id:        user.ID,
		TenantId:  user.TenantID,
		Email:     user.Email,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
}

func claimsResponse(claims domain.TokenClaims, valid bool) *ValidateTokenResponse {
	return &ValidateTokenResponse{
		Valid:       valid,
		Subject:     claims.Subject,
		TenantId:    claims.TenantID,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
		ExpiresAt:   claims.ExpiresAt.Format(time.RFC3339),
		TokenId:     claims.TokenID,
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrPermissionDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrWeakPassword), errors.Is(err, domain.ErrTenantRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrTokenExpired):
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
