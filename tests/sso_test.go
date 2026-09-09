package api_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const address = "http://localhost:28080"

var client = http.Client{
	Timeout: 10 * time.Minute,
}

func TestRegisterBadJSON(t *testing.T) {
	data := bytes.NewBufferString(`{"name":`)
	req, err := http.NewRequest(http.MethodPost, address+"/api/register", data)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGoodRegister(t *testing.T) {
	registerUser(t, "stas", "password")
}

func TestBadRegister(t *testing.T) {
	tests := []struct {
		name string
		data *bytes.Buffer
	}{
		{
			name: "SameEmailAndPasswordAdmin",
			data: bytes.NewBufferString(`{"name":"admin", "password":"password"}`),
		},
		{
			name: "SameEmailAndPasswordUser",
			data: bytes.NewBufferString(`{"name":"stas", "password":"password"}`),
		},
		{
			name: "SameEmail",
			data: bytes.NewBufferString(`{"name":"admin", "password":"122"}`),
		},
		{
			name: "EmptyPassword",
			data: bytes.NewBufferString(`{"name":"dima", "password":""}`),
		},
		{
			name: "EmptyEmail",
			data: bytes.NewBufferString(`{"name":"", "password":"333"}`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, address+"/api/register", tc.data)
			require.NoError(t, err, "cannot make request")
			resp, err := client.Do(req)
			require.NoError(t, err, "cant send register command")
			defer resp.Body.Close()
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}

}

func registerUser(t *testing.T, name string, password string) {
	data := bytes.NewBufferString(`{"name":"` + name + `", "password":"` + password + `"}`)
	req, err := http.NewRequest(http.MethodPost, address+"/api/register", data)
	require.NoError(t, err, "cannot make request")
	resp, err := client.Do(req)
	require.NoError(t, err, "cant send register command")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func loginNoAdmin(t *testing.T, name string, password string) string {
	data := bytes.NewBufferString(`{"name":"` + name + `", "password":"` + password + `"}`)
	req, err := http.NewRequest(http.MethodPost, address+"/api/login", data)
	require.NoError(t, err, "cannot make request")
	resp, err := client.Do(req)
	require.NoError(t, err, "could not send login command")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	token, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(token)
}

func TestLoginWrongPassword(t *testing.T) {
	data := bytes.NewBufferString(`{"name":"stas", "password":"wrong"}`)
	req, err := http.NewRequest(http.MethodPost, address+"/api/login", data)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestLoginUnknownUser(t *testing.T) {
	data := bytes.NewBufferString(`{"name":"unknown", "password":"wrong"}`)
	req, err := http.NewRequest(http.MethodPost, address+"/api/login", data)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
