package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/Henzer3/test-ozon/sso-service/adapters/db"
	ssogrpc "github.com/Henzer3/test-ozon/sso-service/adapters/grpc"
	"github.com/Henzer3/test-ozon/sso-service/config"
	"github.com/Henzer3/test-ozon/sso-service/core"

	ssopb "github.com/Henzer3/test-ozon/pkg/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const gracefulShutdownTime = 2

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()
	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)

	log.Info("start sso initialization server")
	if err := run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	// database adapter
	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}

	defer func() {
		if err := storage.Close(); err != nil {
			log.Error("close conn in database adapter", "err", err)
		}
	}()

	if err := storage.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate db: %v", err)
	}

	// service
	auther := core.NewService(log, storage, storage, storage, cfg.TokenTTL)

	if err := auther.EnsureAdmin(context.Background(), cfg.Admin.Email, cfg.Admin.Password); err != nil {
		return fmt.Errorf("failed to create admin : %v", err)
	}
	if err := auther.EnsureApp(context.Background(), int32(cfg.App.ID), cfg.App.Name, cfg.App.Secret); err != nil {
		return fmt.Errorf("failed to create app: %v", err)
	}
	// grpc server
	listener, err := net.Listen("tcp", cfg.GRPC.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	ssopb.RegisterAuthServer(server, ssogrpc.NewServer(log, auther))
	reflection.Register(server)

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Serve(listener)
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	select {
	case <-signalCtx.Done():
	case err := <-errChan:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*gracefulShutdownTime)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-ctx.Done():
		log.Error("time is up")
		server.Stop()
	}

	if err := <-errChan; !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}
	return nil
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
