package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/Henzer3/test-ozon/sso-service/entity"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {
	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Close() error {
	if err := db.conn.Close(); err != nil {
		db.log.Error("cant close db conn", "err", err)
		return err
	}
	return nil
}

func (db *DB) SaveUser(ctx context.Context, email string, passHash []byte, isAdmin bool) (int64, error) {
	var userID int64
	if err := db.conn.GetContext(ctx, &userID, `
	INSERT INTO users (email, password_hash, is_admin)
	VALUES ($1, $2, $3)
	RETURNING id
	`, email, string(passHash), isAdmin); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, entity.ErrUserExist
		}
		return 0, err
	}

	return userID, nil
}

func (db *DB) IsAdminExist(ctx context.Context) (bool, error) {
	var exists bool

	err := db.conn.GetContext(ctx, &exists, `
		SELECT EXISTS (
			SELECT 1 FROM users
			WHERE is_admin = true
		)
	`)
	if err != nil {
		return false, err
	}

	return exists, nil
}

type user struct {
	ID           int64  `db:"id"`
	Email        string `db:"email"`
	PasswordHash string `db:"password_hash"`
}

func (db *DB) User(ctx context.Context, email string) (entity.User, error) {
	var res user
	if err := db.conn.GetContext(ctx, &res, `
	SELECT id, email, password_hash FROM users
	WHERE email = $1
	`, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.User{}, entity.ErrUserNotFound
		}
		return entity.User{}, err
	}
	return entity.User{ID: res.ID, Email: res.Email, PassHash: []byte(res.PasswordHash)}, nil
}

func (db *DB) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	var v bool
	if err := db.conn.GetContext(ctx, &v, `
	SELECT is_admin FROM users
	WHERE id = $1
	`, userID); err != nil {
		return false, err
	}
	return v, nil
}

type App struct {
	ID     int    `db:"id"`
	Name   string `db:"name"`
	Secret string `db:"secret"`
}

func (db *DB) App(ctx context.Context, appID int32) (entity.App, error) {
	var res App

	if err := db.conn.GetContext(ctx, &res, `
	SELECT id, name, secret FROM apps
	WHERE id = $1
	`, appID); err != nil {
		return entity.App{}, err
	}

	return entity.App{ID: res.ID, Name: res.Name, Secret: res.Secret}, nil
}

func (db *DB) SaveApp(ctx context.Context, id int32, name string, secret string) error {
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO apps (id, name, secret)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name, secret = EXCLUDED.secret
	`, id, name, secret)
	if err != nil {
		return err
	}

	return nil
}
