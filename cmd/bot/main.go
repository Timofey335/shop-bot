package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"shop-bot/config"
	"shop-bot/internal/repository/postgres"
	"shop-bot/internal/service"
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

	shopService := service.NewShopService(cfg.PythonAPIURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Testing get shops")

	shops, err := shopService.GetShops(ctx)
	if err != nil {
		log.Fatalf("Get shops failed: %v", err)
	}
	log.Printf("Found %d shops", len(shops))
	for _, s := range shops {
		log.Printf("  - %s (ID: %s)", s.Name, s.ID)
	}

	log.Println("Bot stopped")
}
