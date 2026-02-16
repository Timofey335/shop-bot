package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	if len(shops) > 0 {
		testShopID := shops[0].ID
		log.Printf("Testing GetProducts for shop %s...", testShopID)

		products, err := shopService.GetProducts(ctx, testShopID)
		if err != nil {
			log.Fatalf("Get products failed: %v", err)
		}
		log.Printf("Found %d products", len(products))

		if len(products) > 0 {
			log.Printf("First product: %s (в наличии: %d)",
				products[0].Name, products[0].Availability)
		}

		log.Println("Testing searchProducts...")
		searchResults, err := shopService.SearchProducts(ctx, testShopID, "молоко")
		if err != nil {
			log.Fatalf("searchProducts failed %v", err)
		}

		log.Printf("Found products %d", len(searchResults))

		for _, i := range searchResults {
			fmt.Println(i)
		}
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	log.Println("Bot stopped")
}
