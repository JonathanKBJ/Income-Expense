package router

import (
	"net/http"

	"expense-tracker/internal/handlers"
	"expense-tracker/internal/middleware"
	"expense-tracker/internal/service"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// New creates and configures the HTTP router with all middleware and routes.
func New(
	authService *service.AuthService,
	authHandler *handlers.AuthHandler,
	adminHandler *handlers.AdminHandler,
	txHandler *handlers.TransactionHandler,
	catHandler *handlers.CategoryHandler,
) http.Handler {
	r := chi.NewRouter()

	// --- Global Middleware ---
	r.Use(chimiddleware.Logger)      // Log every request
	r.Use(chimiddleware.Recoverer)   // Recover from panics
	r.Use(chimiddleware.RequestID)   // Inject request ID
	r.Use(chimiddleware.RealIP)      // Get real client IP
	r.Use(chimiddleware.Heartbeat("/health")) // Health check endpoint

	// CORS configuration for React frontend
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173", "http://127.0.0.1:5173", "*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// --- Public Routes ---
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	// --- Protected API Routes ---
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authService))

		r.Route("/api", func(r chi.Router) {
			// Transaction routes
			r.Route("/transactions", func(r chi.Router) {
				r.Get("/", txHandler.GetTransactions)
				r.Post("/", txHandler.CreateTransaction)
				r.Post("/batch", txHandler.CreateTransactionsBatch)
				r.Patch("/{id}", txHandler.UpdateTransaction)
				r.Delete("/{id}", txHandler.DeleteTransaction)
			})

			// Category routes
			r.Route("/categories", func(r chi.Router) {
				r.Get("/", catHandler.GetCategories)
				r.Post("/", catHandler.CreateCategory)
				r.Patch("/{id}", catHandler.UpdateCategory)
				r.Delete("/{id}", catHandler.DeleteCategory)
			})

			// Admin routes (Protected by Auth + AdminOnly)
			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.AdminOnly)
				r.Get("/users", adminHandler.ListUsers)
				r.Patch("/users/{id}/status", adminHandler.UpdateUserStatus)
				r.Get("/groups", adminHandler.ListGroups)
				r.Get("/groups/{id}/members", adminHandler.GetGroupMembers)
				r.Post("/groups/{id}/members", adminHandler.AddMemberToGroup)
				r.Delete("/groups/{id}/members/{userID}", adminHandler.RemoveMemberFromGroup)
			})
		})
	})

	return r
}

