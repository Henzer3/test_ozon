package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/Henzer3/test-ozon/post-service/internal/config"
	"github.com/Henzer3/test-ozon/post-service/internal/repository"
	ssoClient "github.com/Henzer3/test-ozon/post-service/internal/transport/adapters/sso"
	graphqlapi "github.com/Henzer3/test-ozon/post-service/internal/transport/graphql"
	"github.com/Henzer3/test-ozon/post-service/internal/transport/rest"
	"github.com/Henzer3/test-ozon/post-service/internal/transport/rest/middleware"
	"github.com/Henzer3/test-ozon/post-service/internal/usecase/post"
	"github.com/gin-gonic/gin"
)

func Run(logger *slog.Logger, cfg config.Config) error {
	// creating storage
	storage, err := repository.New(logger, cfg.DBAddress)
	if err != nil {
		logger.Error("error creating postgres storage", "error", err)
		return err
	}

	defer func() {
		if err := storage.Close(); err != nil {
			logger.Error("error closing postgres storage", "error", err)
		}
	}()

	// migrate
	if err := storage.Migrate(); err != nil {
		logger.Error("migrate error", "error", err)
		return err
	}

	// creating ssoClient
	ssoClient, err := ssoClient.NewClient(cfg.SSO.SSOAddress, logger)
	if err != nil {
		logger.Error("cannot init sso client", "error", err)
		return err
	}

	defer func() {
		if err := ssoClient.Close(); err != nil {
			logger.Error("cannot close sso client", "error", err)
		}
	}()

	// creating service

	service := post.New(logger, storage, storage)

	router := gin.New()
	router.Use(gin.Recovery())

	graphqlServer := handler.New(
		graphqlapi.NewExecutableSchema(graphqlapi.Config{
			Resolvers: &graphqlapi.Resolver{
				PostService:    service,
				CommentService: service,
			},
		}),
	)

	graphqlServer.AddTransport(transport.Options{})
	graphqlServer.AddTransport(transport.GET{})
	graphqlServer.AddTransport(transport.POST{})

	router.GET("/", gin.WrapH(
		playground.Handler("GraphQL Playground", "/query"),
	))

	router.POST("/query", middleware.Auth(ssoClient), gin.WrapH(graphqlServer))
	router.POST("/api/login", rest.NewLoginHandler(logger, ssoClient, int32(cfg.AppID)))
	router.POST("/api/register", rest.NewRegisterHandler(logger, ssoClient))

	server := http.Server{
		Addr:        cfg.HTTPConfig.Address,
		ReadTimeout: cfg.HTTPConfig.Timeout,
		Handler:     router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	chanError := make(chan error, 1)
	go func() {
		<-ctx.Done()
		logger.Debug("shutting down server")
		ctxShutdown, cancel := context.WithTimeout(context.Background(), cfg.HTTPConfig.GracefulShutdownTime)
		defer cancel()

		chanError <- server.Shutdown(ctxShutdown)
	}()

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server closed unexpectedly", "error", err)
		return err
	}

	if err = <-chanError; err != nil {
		logger.Debug("hard stop server", "err", err)
		return err
	}

	return nil
}
