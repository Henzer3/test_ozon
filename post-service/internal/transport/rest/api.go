package rest

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Henzer3/test-ozon/post-service/internal/port"
	"github.com/gin-gonic/gin"
)

type Authenticator interface {
	Login(ctx context.Context, in port.LoginRequest) ([]byte, error)
	Register(ctx context.Context, email string, password string) (int64, error)
	Verify(ctx context.Context, token string) (port.UserPermissions, error)
}

type UserData struct {
	User     string `json:"name"`
	Password string `json:"password"`
}

func NewRegisterHandler(log *slog.Logger, auth Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userData UserData
		if err := c.ShouldBindJSON(&userData); err != nil {
			log.Error("cant bind user data", "err", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		_, err := auth.Register(c.Request.Context(), userData.User, userData.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unauthorized"})
			return
		}

		c.Status(http.StatusOK)

	}
}

func NewLoginHandler(log *slog.Logger, auth Authenticator, appID int32) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userData UserData

		if err := c.ShouldBindJSON(&userData); err != nil {
			log.Error("cant bind user data", "err", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		token, err := auth.Login(c.Request.Context(), port.LoginRequest{Email: userData.User, Password: userData.Password, AppId: appID})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.String(http.StatusOK, "%s", token)
	}
}
