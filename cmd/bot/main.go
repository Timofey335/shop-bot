package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shop-bot/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Config loaded successfully")
	log.Printf("Database: %s:%s/%s", cfg.DBHost, cfg.DBPort, cfg.DBName)
	log.Printf("Python API: %s", cfg.PythonAPIURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Bot is starting...")

		<-ctx.Done()
	}()

	sig := <-sigChan
	log.Printf("Received signal %v:", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	log.Println("Shutdown gracefully...")

	select {
	case <-shutdownCtx.Done():
		log.Println("Force shutdown due to timeout")
	default:
		log.Println("Cleanup finished")
	}

	log.Println("Bot stopped")
}
