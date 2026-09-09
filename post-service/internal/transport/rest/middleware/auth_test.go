package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Henzer3/test-ozon/post-service/internal/port"
	middlewaremock "github.com/Henzer3/test-ozon/post-service/internal/transport/rest/middleware/mock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAuthRejectsMissingOrInvalidHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, test := range []struct {
		name   string
		header string
	}{
		{name: "missing header"},
		{name: "wrong prefix", header: "Bearer token"},
		{name: "empty token", header: "Token "},
	} {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			verifier := middlewaremock.NewMockTokenVerifier(controller)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.header != "" {
				request.Header.Set("Authorization", test.header)
			}

			router := gin.New()
			router.GET("/protected", Auth(verifier), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})
			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}
}

func TestAuthRejectsInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := gomock.NewController(t)
	verifier := middlewaremock.NewMockTokenVerifier(controller)
	verifier.EXPECT().Verify(gomock.Any(), "invalid").Return(port.UserPermissions{}, errors.New("invalid token"))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Token invalid")
	router := gin.New()
	router.GET("/protected", Auth(verifier), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAuthAddsUserToContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := gomock.NewController(t)
	verifier := middlewaremock.NewMockTokenVerifier(controller)
	wantUser := port.UserPermissions{ID: 7, Email: "user@example.com", AppID: 1, IsAdmin: true}
	verifier.EXPECT().Verify(gomock.Any(), "valid").Return(wantUser, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Token valid")
	router := gin.New()
	router.GET("/protected", Auth(verifier), func(c *gin.Context) {
		user, ok := port.UserFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, wantUser, user)
		c.Status(http.StatusNoContent)
	})

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}
