package entity

type User struct {
	ID       int64
	Email    string
	PassHash []byte
}

type UserPermission struct {
	ID      int64
	Email   string
	AppID   int32
	IsAdmin bool
}

type App struct {
	ID     int
	Name   string
	Secret string
}
