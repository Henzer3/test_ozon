package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Henzer3/test-ozon/post-service/internal/port"
	"github.com/gin-gonic/gin"
)

type TokenVerifier interface {
	Verify(ctx context.Context, token string) (port.UserPermissions, error)
}

func Auth(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := tokenFromRequest(c)
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

func tokenFromRequest(c *gin.Context) (string, bool) {
	auth := c.GetHeader("Authorization")
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
