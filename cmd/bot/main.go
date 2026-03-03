package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"shop-bot/config"
	"shop-bot/internal/repository/postgres"
	"shop-bot/internal/repository/redis"
	"shop-bot/internal/service"
	"shop-bot/internal/transport/telegram"
	"shop-bot/internal/worker"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := newLogger(cfg.LogLevel, cfg.Environment)
	logger.Info("starting application",
		"environment", cfg.Environment,
		"log_level", cfg.LogLevel)

	pool, err := postgres.New(cfg.PostgreSQLConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	logger.Info("Connected to PostgreSQL")

	stateMgr, err := redis.NewStateManager(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer stateMgr.Close()
	logger.Info("connected to redis")

	userRepo := postgres.NewUserRepo(pool)
	trackingRepo := postgres.NewTrackingRepo(pool)

	shopService := service.NewShopService(cfg.PythonAPIURL)
	trackingService := service.NewTrackingService(trackingRepo, shopService)

	bot, err := telegram.NewBot(cfg.TelegramToken, logger, userRepo, shopService, trackingService, stateMgr)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go bot.Start(ctx)

	tracker := worker.NewTracker(trackingService, userRepo, shopService, logger, bot, 15*time.Minute)
	go tracker.Start(ctx)

	logger.Info("Bot started")

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")

	logger.Info("Bot stopped")
}

func newLogger(level, env string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	if env == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		}))
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}))

}
