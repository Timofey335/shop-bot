package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"shop-bot/config"
	"shop-bot/internal/domain"
	"shop-bot/internal/repository/postgres"
)

func main() {
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
	// trakingRepo := postgres.NewTrackingRepo(pool)

	ctx := context.Background()

	testUser := &domain.User{
		TelegramID: 1234567789,
		Username:   "test_user",
		FirstName:  "test",
		LastName:   "testttt",
	}

	err = userRepo.Create(ctx, testUser)
	if err != nil {
		log.Printf("Create user error: %v", err)
	} else {
		log.Printf("Created user with ID: %v", testUser.ID)
	}

	user, err := userRepo.GetByTelegramID(ctx, testUser.TelegramID)
	if err != nil {
		log.Fatalf("Get user error: %v", err)
	}
	log.Printf("Found user: %s, %s", user.Username, user.LastName)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// go func() {
	// 	log.Println("Bot is starting...")

	// 	<-ctx.Done()
	// }()

	sig := <-sigChan
	log.Printf("Received signal %v:", sig)

	// shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer shutdownCancel()

	// log.Println("Shutdown gracefully...")

	// select {
	// case <-shutdownCtx.Done():
	// 	log.Println("Force shutdown due to timeout")
	// default:
	// 	log.Println("Cleanup finished")
	// }

	log.Println("Bot stopped")
}
