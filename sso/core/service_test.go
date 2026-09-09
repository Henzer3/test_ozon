package core

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	coremock "github.com/Henzer3/test-ozon/sso-service/core/mock"
	"github.com/Henzer3/test-ozon/sso-service/entity"
	jwtlib "github.com/Henzer3/test-ozon/sso-service/lib/jwt"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestNewService(t *testing.T) {
	logger := newTestLogger()
	tokenTTL := time.Hour

	service := NewService(logger, nil, nil, nil, tokenTTL)

	require.NotNil(t, service)
	require.Same(t, logger, service.log)
	require.Equal(t, tokenTTL, service.tokenTTL)
}

func TestRegisterNewUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		service, saver, _, _ := newAuthServiceWithMocks(t)
		saver.EXPECT().SaveUser(ctx, "user@example.com", gomock.Any(), false).Return(int64(42), nil)

		userID, err := service.RegisterNewUser(ctx, "user@example.com", "password")

		require.NoError(t, err)
		require.Equal(t, int64(42), userID)
	})

	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "user already exists", err: entity.ErrUserExist},
		{name: "repository error", err: errors.New("save user failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, saver, _, _ := newAuthServiceWithMocks(t)
			saver.EXPECT().SaveUser(ctx, "user@example.com", gomock.Any(), false).Return(int64(0), test.err)

			userID, err := service.RegisterNewUser(ctx, "user@example.com", "password")

			require.ErrorIs(t, err, test.err)
			require.Zero(t, userID)
		})
	}

	t.Run("password is too long", func(t *testing.T) {
		service, _, _, _ := newAuthServiceWithMocks(t)

		userID, err := service.RegisterNewUser(ctx, "user@example.com", strings.Repeat("a", 73))

		require.ErrorIs(t, err, bcrypt.ErrPasswordTooLong)
		require.Zero(t, userID)
	})
}

func TestEnsureAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("admin already exists", func(t *testing.T) {
		service, _, users, _ := newAuthServiceWithMocks(t)
		users.EXPECT().IsAdminExist(ctx).Return(true, nil)

		require.NoError(t, service.EnsureAdmin(ctx, "admin@example.com", "password"))
	})

	t.Run("creates admin", func(t *testing.T) {
		service, saver, users, _ := newAuthServiceWithMocks(t)
		users.EXPECT().IsAdminExist(ctx).Return(false, nil)
		saver.EXPECT().SaveUser(ctx, "admin@example.com", gomock.Any(), true).Return(int64(1), nil)

		require.NoError(t, service.EnsureAdmin(ctx, "admin@example.com", "password"))
	})

	t.Run("checking admin fails", func(t *testing.T) {
		service, _, users, _ := newAuthServiceWithMocks(t)
		wantErr := errors.New("check admin failed")
		users.EXPECT().IsAdminExist(ctx).Return(false, wantErr)

		require.ErrorIs(t, service.EnsureAdmin(ctx, "admin@example.com", "password"), wantErr)
	})

	t.Run("password is too long", func(t *testing.T) {
		service, _, users, _ := newAuthServiceWithMocks(t)
		users.EXPECT().IsAdminExist(ctx).Return(false, nil)

		err := service.EnsureAdmin(ctx, "admin@example.com", strings.Repeat("a", 73))

		require.ErrorIs(t, err, bcrypt.ErrPasswordTooLong)
	})

	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "admin already exists as user", err: entity.ErrUserExist},
		{name: "saving admin fails", err: errors.New("save admin failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, saver, users, _ := newAuthServiceWithMocks(t)
			users.EXPECT().IsAdminExist(ctx).Return(false, nil)
			saver.EXPECT().SaveUser(ctx, "admin@example.com", gomock.Any(), true).Return(int64(0), test.err)

			require.ErrorIs(t, service.EnsureAdmin(ctx, "admin@example.com", "password"), test.err)
		})
	}
}

func TestEnsureApp(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		service, _, _, apps := newAuthServiceWithMocks(t)
		apps.EXPECT().SaveApp(ctx, int32(1), "post-service", "secret").Return(nil)

		require.NoError(t, service.EnsureApp(ctx, 1, "post-service", "secret"))
	})

	t.Run("repository error", func(t *testing.T) {
		service, _, _, apps := newAuthServiceWithMocks(t)
		wantErr := errors.New("save app failed")
		apps.EXPECT().SaveApp(ctx, int32(1), "post-service", "secret").Return(wantErr)

		require.ErrorIs(t, service.EnsureApp(ctx, 1, "post-service", "secret"), wantErr)
	})
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	const password = "password"
	user := entity.User{ID: 7, Email: "user@example.com", PassHash: passwordHash(t, password)}
	app := entity.App{ID: 3, Name: "post-service", Secret: "secret"}

	t.Run("success", func(t *testing.T) {
		service, _, users, apps := newAuthServiceWithMocks(t)
		users.EXPECT().User(ctx, user.Email).Return(user, nil)
		apps.EXPECT().App(ctx, int32(app.ID)).Return(app, nil)

		token, err := service.Login(ctx, user.Email, password, int32(app.ID))

		require.NoError(t, err)
		permissions, err := jwtlib.Verify(token, app.Secret)
		require.NoError(t, err)
		require.Equal(t, entity.UserPermission{ID: user.ID, Email: user.Email, AppID: int32(app.ID)}, permissions)
	})

	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "user not found", err: entity.ErrUserNotFound},
		{name: "repository error", err: errors.New("get user failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, _, users, _ := newAuthServiceWithMocks(t)
			users.EXPECT().User(ctx, user.Email).Return(entity.User{}, test.err)

			token, err := service.Login(ctx, user.Email, password, int32(app.ID))

			require.ErrorIs(t, err, test.err)
			require.Empty(t, token)
		})
	}

	t.Run("wrong password", func(t *testing.T) {
		service, _, users, _ := newAuthServiceWithMocks(t)
		users.EXPECT().User(ctx, user.Email).Return(user, nil)

		token, err := service.Login(ctx, user.Email, "wrong-password", int32(app.ID))

		require.ErrorIs(t, err, entity.ErrInvalidCredentials)
		require.Empty(t, token)
	})

	t.Run("app repository error", func(t *testing.T) {
		service, _, users, apps := newAuthServiceWithMocks(t)
		wantErr := errors.New("get app failed")
		users.EXPECT().User(ctx, user.Email).Return(user, nil)
		apps.EXPECT().App(ctx, int32(app.ID)).Return(entity.App{}, wantErr)

		token, err := service.Login(ctx, user.Email, password, int32(app.ID))

		require.ErrorIs(t, err, wantErr)
		require.Empty(t, token)
	})
}

func TestIsAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		service, _, users, _ := newAuthServiceWithMocks(t)
		users.EXPECT().IsAdmin(ctx, int64(5)).Return(true, nil)

		isAdmin, err := service.IsAdmin(ctx, 5)

		require.NoError(t, err)
		require.True(t, isAdmin)
	})

	t.Run("repository error", func(t *testing.T) {
		service, _, users, _ := newAuthServiceWithMocks(t)
		wantErr := errors.New("is admin failed")
		users.EXPECT().IsAdmin(ctx, int64(5)).Return(false, wantErr)

		isAdmin, err := service.IsAdmin(ctx, 5)

		require.ErrorIs(t, err, wantErr)
		require.False(t, isAdmin)
	})
}

func TestVerify(t *testing.T) {
	ctx := context.Background()
	user := entity.User{ID: 9, Email: "user@example.com"}
	app := entity.App{ID: 4, Name: "post-service", Secret: "secret"}
	token, err := jwtlib.NewToken(user, app, time.Hour)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		service, _, users, apps := newAuthServiceWithMocks(t)
		apps.EXPECT().App(ctx, int32(app.ID)).Return(app, nil)
		users.EXPECT().IsAdmin(ctx, user.ID).Return(true, nil)

		permissions, err := service.Verify(ctx, token)

		require.NoError(t, err)
		require.Equal(t, entity.UserPermission{
			ID:      user.ID,
			Email:   user.Email,
			AppID:   int32(app.ID),
			IsAdmin: true,
		}, permissions)
	})

	t.Run("malformed token", func(t *testing.T) {
		service, _, _, _ := newAuthServiceWithMocks(t)

		permissions, err := service.Verify(ctx, "invalid-token")

		require.ErrorIs(t, err, entity.ErrInvalidToken)
		require.Zero(t, permissions)
	})

	t.Run("app repository error", func(t *testing.T) {
		service, _, _, apps := newAuthServiceWithMocks(t)
		wantErr := errors.New("get app failed")
		apps.EXPECT().App(ctx, int32(app.ID)).Return(entity.App{}, wantErr)

		_, err := service.Verify(ctx, token)

		require.ErrorIs(t, err, wantErr)
	})

	t.Run("wrong secret", func(t *testing.T) {
		service, _, _, apps := newAuthServiceWithMocks(t)
		wrongApp := app
		wrongApp.Secret = "wrong-secret"
		apps.EXPECT().App(ctx, int32(app.ID)).Return(wrongApp, nil)

		_, err := service.Verify(ctx, token)

		require.ErrorIs(t, err, entity.ErrInvalidToken)
	})

	t.Run("checking admin fails", func(t *testing.T) {
		service, _, users, apps := newAuthServiceWithMocks(t)
		wantErr := errors.New("is admin failed")
		apps.EXPECT().App(ctx, int32(app.ID)).Return(app, nil)
		users.EXPECT().IsAdmin(ctx, user.ID).Return(false, wantErr)

		_, err := service.Verify(ctx, token)

		require.ErrorIs(t, err, wantErr)
	})
}

func newAuthServiceWithMocks(
	t *testing.T,
) (*AuthService, *coremock.MockUserSaver, *coremock.MockUserProvider, *coremock.MockAppProvider) {
	t.Helper()

	controller := gomock.NewController(t)
	saver := coremock.NewMockUserSaver(controller)
	users := coremock.NewMockUserProvider(controller)
	apps := coremock.NewMockAppProvider(controller)

	return NewService(newTestLogger(), saver, users, apps, time.Hour), saver, users, apps
}

func passwordHash(t *testing.T, password string) []byte {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return hash
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
