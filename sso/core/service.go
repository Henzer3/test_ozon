package core

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Henzer3/test-ozon/sso-service/entity"
	"github.com/Henzer3/test-ozon/sso-service/lib/jwt"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	log          *slog.Logger
	userSaver    entity.UserSaver
	userProvider entity.UserProvider
	appProvider  entity.AppProvider
	tokenTTL     time.Duration
}

func NewService(log *slog.Logger, userSaver entity.UserSaver, userProvider entity.UserProvider,
	appProvider entity.AppProvider, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		log:          log,
		userSaver:    userSaver,
		userProvider: userProvider,
		appProvider:  appProvider,
		tokenTTL:     tokenTTL,
	}
}

func (s *AuthService) RegisterNewUser(ctx context.Context, email string, password string) (int64, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("cant get hash in core", "err", err)
		return 0, err
	}

	userID, err := s.userSaver.SaveUser(ctx, email, hash, false)
	if err != nil {
		if errors.Is(err, entity.ErrUserExist) {
			return 0, err
		}
		s.log.Error("cant saveUser in core", "err", err)
		return 0, err
	}

	return userID, nil
}

func (s *AuthService) EnsureAdmin(ctx context.Context, email string, password string) error {
	v, err := s.userProvider.IsAdminExist(ctx)
	if err != nil {
		s.log.Error("cant do IsAdminExists in core", "err", err)
		return err
	}

	if v {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("cant get hash in core", "err", err)
		return err
	}

	_, err = s.userSaver.SaveUser(ctx, email, hash, true)
	if err != nil {
		if errors.Is(err, entity.ErrUserExist) {
			return err
		}
		s.log.Error("cant saveUser in core", "err", err)
		return err
	}
	return nil
}

func (s *AuthService) EnsureApp(ctx context.Context, id int32, name string, secret string) error {
	if err := s.appProvider.SaveApp(ctx, id, name, secret); err != nil {
		s.log.Error("cant EnsureApp in core", "err", err)
		return err
	}
	return nil
}

func (s *AuthService) Login(ctx context.Context, email string, password string, appID int32) (string, error) {
	user, err := s.userProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return "", err
		}
		s.log.Error("cant userProvider in core", "err", err)
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		return "", entity.ErrInvalidCredentials
	}

	app, err := s.appProvider.App(ctx, appID)
	if err != nil {
		s.log.Error("cant appProvider in core", "err", err)
		return "", err
	}

	token, err := jwt.NewToken(user, app, s.tokenTTL)
	if err != nil {
		s.log.Error("cant create token in core", "err", err)
		return "", err
	}

	return token, nil
}

func (s *AuthService) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	isAdmin, err := s.userProvider.IsAdmin(ctx, userID)
	if err != nil {
		s.log.Error("cant isAdmin in core", "err", err)
		return false, err
	}

	return isAdmin, nil
}

func (s *AuthService) Verify(ctx context.Context, token string) (entity.UserPermission, error) {
	appID, err := jwt.AppIDFromToken(token)
	if err != nil {
		return entity.UserPermission{}, entity.ErrInvalidToken
	}
	app, err := s.appProvider.App(ctx, appID)
	if err != nil {
		s.log.Error("cant appProvider in core", "err", err)
		return entity.UserPermission{}, err
	}

	userPerms, err := jwt.Verify(token, app.Secret)
	if err != nil {
		return entity.UserPermission{}, entity.ErrInvalidToken
	}

	v, err := s.userProvider.IsAdmin(ctx, userPerms.ID)
	if err != nil {
		s.log.Error("cant isAdmin in core", "err", err)
		return entity.UserPermission{}, err
	}

	userPerms.IsAdmin = v

	return userPerms, nil
}
