package entity

import "context"

type Auther interface {
	RegisterNewUser(ctx context.Context, email string, password string) (int64, error)
	Login(ctx context.Context, email string, password string, appID int32) (string, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
	Verify(ctx context.Context, token string) (UserPermission, error)
}

type UserSaver interface {
	SaveUser(ctx context.Context, email string, passHash []byte, isAdmin bool) (int64, error)
}

type UserProvider interface {
	User(ctx context.Context, email string) (User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
	IsAdminExist(ctx context.Context) (bool, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int32) (App, error)
	SaveApp(ctx context.Context, id int32, name string, secret string) error
}
