package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/pxc1984/nnkl-backend/api"
	apiv1 "github.com/pxc1984/nnkl-backend/api/v1"
	store2 "github.com/pxc1984/nnkl-backend/store"
	"github.com/pxc1984/nnkl-backend/utils"
)

func main() {
	_ = godotenv.Load()
	utils.InitSettings()
	utils.InitLogging()

	slog.Debug("loaded utils from .env")

	store, err := store2.InitStore()
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			slog.Error("close store", "error", err)
		}
	}()

	router := gin.Default()

	v1 := router.Group("/api/v1")
	v1.Use(api.AuditMiddleware())
	apiv1.RegisterRoutes(v1)

	router.GET("/api/health", api.AuditMiddleware(), api.HealthCheck)

	addr := fmt.Sprintf("%s:%d", utils.Settings.Host, utils.Settings.Port)
	slog.Info("listening", "addr", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown:", err)
	}

	slog.Info("server exited")
}
