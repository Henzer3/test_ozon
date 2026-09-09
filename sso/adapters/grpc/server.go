package grpc

import (
	"context"
	"errors"
	"log/slog"

	ssopb "github.com/Henzer3/test-ozon/pkg/sso"
	"github.com/Henzer3/test-ozon/sso-service/entity"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ssoServer struct {
	ssopb.UnimplementedAuthServer
	log     *slog.Logger
	service entity.Auther
}

func NewServer(log *slog.Logger, service entity.Auther) *ssoServer {
	return &ssoServer{
		log:     log,
		service: service,
	}
}

func (s *ssoServer) Verify(ctx context.Context, in *ssopb.VerifyRequest) (*ssopb.VerifyResponse, error) {
	if in.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "empty token")
	}

	userPerms, err := s.service.Verify(ctx, in.GetToken())
	if err != nil {
		if errors.Is(err, entity.ErrInvalidToken) {
			return nil, status.Error(codes.InvalidArgument, "invalid token")
		}
		return nil, status.Error(codes.Internal, "InternalError")
	}

	return &ssopb.VerifyResponse{
		UserId:  userPerms.ID,
		Email:   userPerms.Email,
		AppId:   userPerms.AppID,
		IsAdmin: userPerms.IsAdmin,
	}, nil
}

func (s *ssoServer) Register(ctx context.Context, in *ssopb.RegisterRequest) (*ssopb.RegisterResponse, error) {
	if in.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	id, err := s.service.RegisterNewUser(ctx, in.GetEmail(), in.GetPassword())
	if err != nil {
		if errors.Is(err, entity.ErrUserExist) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		s.log.Error("cant Register in adapter", "err", err)
		return nil, status.Error(codes.Internal, "InternalError")
	}
	return &ssopb.RegisterResponse{UserId: int64(id)}, nil
}

func (s *ssoServer) Login(ctx context.Context, in *ssopb.LoginRequest) (*ssopb.LoginResponse, error) {
	if in.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	if in.GetAppId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	token, err := s.service.Login(ctx, in.GetEmail(), in.GetPassword(), in.GetAppId())
	if err != nil {
		if errors.Is(err, entity.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid email or password")
		} else if errors.Is(err, entity.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "NotFound")
		}
		s.log.Error("cant login in adapter", "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssopb.LoginResponse{Token: token}, nil
}

func (s *ssoServer) IsAdmin(ctx context.Context, in *ssopb.IsAdminRequest) (*ssopb.IsAdminResponse, error) {
	if in.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	v, err := s.service.IsAdmin(ctx, in.GetUserId())
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.log.Error("cant IsAdmin in adapter", "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssopb.IsAdminResponse{IsAdmin: v}, nil
}
