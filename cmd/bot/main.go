package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"shop-bot/config"
	"shop-bot/internal/repository/postgres"
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

	pool, err := postgres.New(cfg.PostgreSQLConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Printf("Connected to PostgreSQL")

	userRepo := postgres.NewUserRepo(pool)
	trackingRepo := postgres.NewTrackingRepo(pool)

	shopService := service.NewShopService(cfg.PythonAPIURL)
	trackingService := service.NewTrackingService(trackingRepo, shopService)

	bot, err := telegram.NewBot(cfg.TelegramToken, userRepo, shopService, trackingService)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go bot.Start(ctx)

	tracker := worker.NewTracker(trackingService, userRepo, shopService, bot, 1*time.Minute)
	go tracker.Start(ctx)

	log.Println("Bot started. Press Ctrl+C to stop.")

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	log.Println("Bot stopped")
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
