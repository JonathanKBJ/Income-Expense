package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"expense-tracker/internal/config"
	"expense-tracker/internal/database"
	"expense-tracker/internal/handlers"
	"expense-tracker/internal/repository"
	"expense-tracker/internal/router"
	"expense-tracker/internal/service"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Expense Tracker API server...")

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Configuration loaded (port=%s, db=%s)", cfg.ServerPort, cfg.TursoDatabaseURL)

	// Connect to Turso database
	db, err := database.New(cfg.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to Turso database")

	// Run schema migration
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run database migration: %v", err)
	}
	log.Println("Database migration completed")

	// Initialize layers: Repository → Service → Handler → Router
	userRepo := repository.NewUserRepository(db.DB) // Access underlying *sql.DB
	groupRepo := repository.NewGroupRepository(db.DB)
	authService := service.NewAuthService(userRepo, groupRepo)

	authHandler := handlers.NewAuthHandler(authService)
	adminHandler := handlers.NewAdminHandler(userRepo, groupRepo)

	txRepo := repository.NewTransactionRepository(db)
	txHandler := handlers.NewTransactionHandler(txRepo)

	catRepo := repository.NewCategoryRepository(db)
	catHandler := handlers.NewCategoryHandler(catRepo)

	httpRouter := router.New(authService, authHandler, adminHandler, txHandler, catHandler)

	// Configure HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      httpRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on http://localhost:%s", cfg.ServerPort)
		log.Printf("API endpoints:")
		log.Printf("  POST   /auth/login")
		log.Printf("  POST   /auth/register")
		log.Printf("  GET    /api/transactions")
		log.Printf("  GET    /api/categories")
		log.Printf("  GET    /api/admin/users")
		log.Printf("  GET    /health")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal %s, shutting down gracefully...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
