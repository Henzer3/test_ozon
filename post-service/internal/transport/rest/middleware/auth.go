package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/Henzer3/test-ozon/post-service/internal/port"
	"github.com/gin-gonic/gin"
)

type TokenVerifier interface {
	Verify(ctx context.Context, token string) (port.UserPermissions, error)
}

func Auth(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := tokenFromAuthorization(c.GetHeader("Authorization"))
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		userPerms, err := verifier.Verify(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		ctx := port.WithUser(c.Request.Context(), userPerms)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func WebsocketInit(verifier TokenVerifier) transport.WebsocketInitFunc {
	return func(
		ctx context.Context,
		payload transport.InitPayload,
	) (context.Context, *transport.InitPayload, error) {
		token, ok := tokenFromAuthorization(payload.Authorization())
		if !ok {
			return ctx, nil, errors.New("authentication required")
		}

		user, err := verifier.Verify(ctx, token)
		if err != nil {
			return ctx, nil, errors.New("authentication failed")
		}

		return port.WithUser(ctx, user), nil, nil
	}
}

func tokenFromAuthorization(auth string) (string, bool) {
	if auth == "" {
		return "", false
	}

	const prefix = "Token "
	if !strings.HasPrefix(auth, prefix) {
		return "", false
	}

	token := strings.TrimPrefix(auth, prefix)
	if token == "" {
		return "", false
	}

	return token, true
}
