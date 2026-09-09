package rest

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Henzer3/test-ozon/post-service/internal/port"
	restmock "github.com/Henzer3/test-ozon/post-service/internal/transport/rest/mock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRegisterHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("bad JSON", func(t *testing.T) {
		controller := gomock.NewController(t)
		auth := restmock.NewMockAuthenticator(controller)
		response := performAPIRequest(t, NewRegisterHandler(testLogger(), auth), `{"name":`)
		require.Equal(t, http.StatusBadRequest, response.Code)
		require.JSONEq(t, `{"error":"bad request"}`, response.Body.String())
	})

	t.Run("success", func(t *testing.T) {
		controller := gomock.NewController(t)
		auth := restmock.NewMockAuthenticator(controller)
		auth.EXPECT().Register(gomock.Any(), "user@example.com", "password").Return(int64(10), nil)
		response := performAPIRequest(t, NewRegisterHandler(testLogger(), auth), `{"name":"user@example.com","password":"password"}`)
		require.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("SSO error", func(t *testing.T) {
		controller := gomock.NewController(t)
		auth := restmock.NewMockAuthenticator(controller)
		auth.EXPECT().Register(gomock.Any(), "user@example.com", "password").Return(int64(0), errors.New("register failed"))
		response := performAPIRequest(t, NewRegisterHandler(testLogger(), auth), `{"name":"user@example.com","password":"password"}`)
		require.Equal(t, http.StatusBadRequest, response.Code)
		require.JSONEq(t, `{"error":"unauthorized"}`, response.Body.String())
	})
}

func TestLoginHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const appID int32 = 3

	t.Run("bad JSON", func(t *testing.T) {
		controller := gomock.NewController(t)
		auth := restmock.NewMockAuthenticator(controller)
		response := performAPIRequest(t, NewLoginHandler(testLogger(), auth, appID), `{"name":`)
		require.Equal(t, http.StatusBadRequest, response.Code)
		require.JSONEq(t, `{"error":"bad request"}`, response.Body.String())
	})

	t.Run("success", func(t *testing.T) {
		controller := gomock.NewController(t)
		auth := restmock.NewMockAuthenticator(controller)
		auth.EXPECT().Login(gomock.Any(), port.LoginRequest{
			Email: "user@example.com", Password: "password", AppId: appID,
		}).Return([]byte("jwt-token"), nil)
		response := performAPIRequest(t, NewLoginHandler(testLogger(), auth, appID), `{"name":"user@example.com","password":"password"}`)
		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, "jwt-token", response.Body.String())
	})

	t.Run("SSO error", func(t *testing.T) {
		controller := gomock.NewController(t)
		auth := restmock.NewMockAuthenticator(controller)
		auth.EXPECT().Login(gomock.Any(), gomock.Any()).Return(nil, errors.New("login failed"))
		response := performAPIRequest(t, NewLoginHandler(testLogger(), auth, appID), `{"name":"user@example.com","password":"password"}`)
		require.Equal(t, http.StatusUnauthorized, response.Code)
		require.JSONEq(t, `{"error":"unauthorized"}`, response.Body.String())
	})
}

func performAPIRequest(t *testing.T, handler gin.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	handler(context)
	return recorder
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
