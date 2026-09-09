package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/Henzer3/test-ozon/sso-service/entity"
	"github.com/golang-jwt/jwt/v5"
)

func NewToken(user entity.User, app entity.App, dur time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["uid"] = user.ID
	claims["email"] = user.Email
	claims["exp"] = time.Now().Add(dur).Unix()
	claims["app_id"] = app.ID

	tokenString, err := token.SignedString([]byte(app.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func Verify(tokenString string, secretKey string) (entity.UserPermission, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return entity.UserPermission{}, err
	}

	if !token.Valid {
		return entity.UserPermission{}, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return entity.UserPermission{}, errors.New("invalid claims")
	}

	idFloat, ok := claims["uid"].(float64)
	if !ok {
		return entity.UserPermission{}, errors.New("uid is invalid")
	}

	emailValue, ok := claims["email"].(string)
	if !ok {
		return entity.UserPermission{}, errors.New("email is invalid")
	}

	appIDFloat, ok := claims["app_id"].(float64)
	if !ok {
		return entity.UserPermission{}, errors.New("app_id is invalid")
	}

	return entity.UserPermission{
		ID:    int64(idFloat),
		Email: emailValue,
		AppID: int32(appIDFloat),
	}, nil
}

func AppIDFromToken(tokenString string) (int32, error) {
	claims := jwt.MapClaims{}

	_, _, err := new(jwt.Parser).ParseUnverified(tokenString, claims)
	if err != nil {
		return 0, err
	}

	appID, ok := claims["app_id"].(float64)
	if !ok {
		return 0, errors.New("invalid app_id")
	}

	return int32(appID), nil
}
