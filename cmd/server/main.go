package main

import (
	"log"

	"github.com/Code-Linx/wallet-service/internal/config"
	"github.com/Code-Linx/wallet-service/internal/handlers"
	"github.com/Code-Linx/wallet-service/internal/repositories"
	"github.com/Code-Linx/wallet-service/internal/usecases"
	"github.com/Code-Linx/wallet-service/pkg/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Test database connection
	if err := database.TestConnection(db); err != nil {
		log.Fatalf("Failed to test database connection: %v", err)
	}

	// Run database migrations
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Initialize repositories
	repos := repositories.NewRepositories(db)

	// Initialize use cases
	useCases := usecases.NewUseCases(repos)

	// Initialize handlers
	handlers := handlers.NewHandlers(useCases)

	// Setup router
	router := handlers.SetupRouter(handlers)

	// Start server
	serverAddr := cfg.GetServerAddress()
	log.Printf("Starting server on %s", serverAddr)
	log.Printf("Environment: %s", cfg.App.Env)
	log.Printf("Database: %s", cfg.Database.Name)

	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
