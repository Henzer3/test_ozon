package jwt

import (
	"testing"
	"time"

	"github.com/Henzer3/test-ozon/sso-service/entity"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestNewTokenAndVerify(t *testing.T) {
	user := entity.User{ID: 42, Email: "user@example.com"}
	app := entity.App{ID: 7, Name: "post-service", Secret: "secret"}

	token, err := NewToken(user, app, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	permission, err := Verify(token, app.Secret)
	require.NoError(t, err)
	require.Equal(t, entity.UserPermission{
		ID:    user.ID,
		Email: user.Email,
		AppID: int32(app.ID),
	}, permission)
}

func TestVerify(t *testing.T) {
	const secret = "secret"

	tests := []struct {
		name   string
		token  func(t *testing.T) string
		secret string
	}{
		{
			name: "malformed token",
			token: func(t *testing.T) string {
				return "not-a-jwt"
			},
			secret: secret,
		},
		{
			name: "wrong secret",
			token: func(t *testing.T) string {
				return signedToken(t, jwtlib.MapClaims{
					"uid": int64(1), "email": "user@example.com", "app_id": 2,
				}, secret)
			},
			secret: "another-secret",
		},
		{
			name: "expired token",
			token: func(t *testing.T) string {
				return signedToken(t, jwtlib.MapClaims{
					"uid": int64(1), "email": "user@example.com", "app_id": 2,
					"exp": time.Now().Add(-time.Minute).Unix(),
				}, secret)
			},
			secret: secret,
		},
		{
			name: "unexpected signing method",
			token: func(t *testing.T) string {
				token := jwtlib.NewWithClaims(jwtlib.SigningMethodNone, jwtlib.MapClaims{
					"uid": int64(1), "email": "user@example.com", "app_id": 2,
				})
				value, err := token.SignedString(jwtlib.UnsafeAllowNoneSignatureType)
				require.NoError(t, err)
				return value
			},
			secret: secret,
		},
		{
			name: "invalid uid",
			token: func(t *testing.T) string {
				return signedToken(t, jwtlib.MapClaims{
					"uid": "one", "email": "user@example.com", "app_id": 2,
				}, secret)
			},
			secret: secret,
		},
		{
			name: "invalid email",
			token: func(t *testing.T) string {
				return signedToken(t, jwtlib.MapClaims{
					"uid": int64(1), "email": 123, "app_id": 2,
				}, secret)
			},
			secret: secret,
		},
		{
			name: "invalid app id",
			token: func(t *testing.T) string {
				return signedToken(t, jwtlib.MapClaims{
					"uid": int64(1), "email": "user@example.com", "app_id": "two",
				}, secret)
			},
			secret: secret,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			permission, err := Verify(test.token(t), test.secret)

			require.Error(t, err)
			require.Empty(t, permission)
		})
	}
}

func TestAppIDFromToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		token := signedToken(t, jwtlib.MapClaims{"app_id": 15}, "secret")

		appID, err := AppIDFromToken(token)

		require.NoError(t, err)
		require.Equal(t, int32(15), appID)
	})

	t.Run("signature is not verified", func(t *testing.T) {
		token := signedToken(t, jwtlib.MapClaims{"app_id": 15}, "unknown-secret")

		appID, err := AppIDFromToken(token)

		require.NoError(t, err)
		require.Equal(t, int32(15), appID)
	})

	t.Run("malformed token", func(t *testing.T) {
		appID, err := AppIDFromToken("not-a-jwt")

		require.Error(t, err)
		require.Zero(t, appID)
	})

	for _, claims := range []jwtlib.MapClaims{
		{},
		{"app_id": "fifteen"},
	} {
		t.Run("invalid app id", func(t *testing.T) {
			token := signedToken(t, claims, "secret")

			appID, err := AppIDFromToken(token)

			require.EqualError(t, err, "invalid app_id")
			require.Zero(t, appID)
		})
	}
}

func signedToken(t *testing.T, claims jwtlib.MapClaims, secret string) string {
	t.Helper()

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	value, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	return value
}
