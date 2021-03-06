package app

import (
	"context"
	"fmt"
	"github.com/v-lozhkin/deployProject/cmd/api/app/config"
	"github.com/v-lozhkin/deployProject/internal/pkg/api/middlewares"
	imageStore "github.com/v-lozhkin/deployProject/internal/pkg/image/storage/fs"
	echoDelivery "github.com/v-lozhkin/deployProject/internal/pkg/item/delivery/echo"
	itemRepo "github.com/v-lozhkin/deployProject/internal/pkg/item/repository/inmemory"
	itemUsecase "github.com/v-lozhkin/deployProject/internal/pkg/item/usecase"
	userDelivery "github.com/v-lozhkin/deployProject/internal/pkg/user/delivery"
	userRepo "github.com/v-lozhkin/deployProject/internal/pkg/user/repository/inmemory"
	user "github.com/v-lozhkin/deployProject/internal/pkg/user/usecase"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	// postgres driver

	"github.com/labstack/echo/v4"
	echoMiddlewares "github.com/labstack/echo/v4/middleware"
	echolog "github.com/labstack/gommon/log"

	_ "github.com/lib/pq"

	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func App() {
	server := echo.New()
	server.Logger.SetLevel(echolog.INFO)

	cfg := config.Config{}
	if err := cfg.ReadFromFile(server.Logger); err != nil {
		cfg.ReadFromEnv(server.Logger)
	}

	server.Use(echoMiddlewares.Recover())
	server.Use(echoMiddlewares.Logger())
	server.Use(middlewares.RequestIDMiddleware())
	server.Use(echoMiddlewares.TimeoutWithConfig(echoMiddlewares.TimeoutConfig{
		Timeout: time.Second * 10,
	}))

	loglevel, ok := loglevelMap[cfg.Loglevel]
	if !ok {
		loglevel = echolog.INFO
	}
	server.Logger.SetLevel(loglevel)

	stat := promauto.Factory{}

	authMiddleware := middlewares.JWTAuthMiddleware(cfg.AuthConfig.JWTSecret)

	//db, err := sqlx.Open(
	//	"postgres",
	//	cfg.DBConfig.DBUrl,
	//)
	//if err != nil {
	//	server.Logger.Fatalf("failed to open db connection %v", err)
	//}
	//
	itemsRepository := itemRepo.New()

	userRepository := userRepo.New()

	itemsUsecase := itemUsecase.New(itemsRepository, stat)
	images := imageStore.New(cfg.StoragePath, stat)
	usersUsecase := user.New(userRepository)

	itemsDelivery := echoDelivery.New(itemsUsecase, images, stat)
	usersDelivery := userDelivery.New(
		usersUsecase,
		time.Now().Add(cfg.AuthConfig.JWTTTL).Unix(),
		cfg.AuthConfig.JWTSecret,
	)

	v1Group := server.Group("/v1")
	itemsGroup := v1Group.Group("/items")
	itemsGroup.GET("", itemsDelivery.List)
	itemsGroup.POST("", itemsDelivery.Create, authMiddleware)
	itemsGroup.POST("/:id/upload", itemsDelivery.Upload, authMiddleware)
	itemsGroup.GET("/:id", itemsDelivery.List)
	itemsGroup.PUT("/:id", itemsDelivery.Update, authMiddleware)
	itemsGroup.DELETE("/:id", itemsDelivery.Delete, authMiddleware)

	v1Group.POST("/user/login", usersDelivery.Login)

	v1Group.Static("/static", "storage")

	server.Any("/metrics", func(ectx echo.Context) error {
		promhttp.Handler().ServeHTTP(ectx.Response().Writer, ectx.Request())
		return nil
	})

	go func() {
		if err := server.Start(fmt.Sprintf(":%d", cfg.Port)); err != nil && err != http.ErrServerClosed {
			server.Logger.Fatal(err)
		}
	}()

	quite := make(chan os.Signal, 1)
	signal.Notify(quite, syscall.SIGINT, syscall.SIGTERM)
	<-quite
	server.Logger.Info("shutdown inited")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		server.Logger.Fatal(err)
	}
}

var loglevelMap = map[string]echolog.Lvl{
	"debug": echolog.DEBUG,
	"info":  echolog.INFO,
	"error": echolog.ERROR,
	"warn":  echolog.WARN,
	"off":   echolog.OFF,
}
