package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"github.com/kuromii5/poller/config"
	"github.com/kuromii5/poller/internal/adapters/postgres"
	redisadapter "github.com/kuromii5/poller/internal/adapters/redis"
	"github.com/kuromii5/poller/internal/handlers"
	healthhandler "github.com/kuromii5/poller/internal/handlers/health"
	pollhandler "github.com/kuromii5/poller/internal/handlers/poll"
	pollservice "github.com/kuromii5/poller/internal/service/poll"
)

func main() {
	// 1. Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// 2. Load .env
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using environment variables")
	}

	// 3. Parse config
	cfg := config.MustLoad()

	// 4. Connect to PostgreSQL
	db, err := postgres.New(cfg.DB)
	if err != nil {
		slog.Error("connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 5. Run migrations
	if err := runMigrations(db); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}

	// 6. Connect to Redis
	rdb := redisadapter.NewClient(cfg.Redis)
	defer rdb.Close()

	// 7. Build adapters
	cache := redisadapter.NewCache(rdb)
	rateLimiter := redisadapter.NewRateLimiter(rdb)

	// 8. Build service
	svc := pollservice.NewService(db, cache)

	// 9. Build handlers
	pollH := pollhandler.NewHandler(svc)
	healthH := healthhandler.NewHandler(db.DB, rdb)

	// 10. Build router
	router := handlers.NewRouter(pollH, healthH, rateLimiter)

	// 11. Start server with graceful shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server started", "port", cfg.Port)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}

func runMigrations(db *postgres.DB) error {
	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db.DB.DB, "migrations")
}
