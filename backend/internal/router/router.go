package router

import (
	"net/http"

	"expense-tracker/internal/handlers"
	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
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
	groupHandler *handlers.GroupHandler,
	loanHandler *handlers.LoanHandler,
) http.Handler {
	r := chi.NewRouter()

	// --- Global Middleware ---
	r.Use(chimiddleware.Logger)       // Log every request
	r.Use(chimiddleware.Recoverer)    // Recover from panics
	r.Use(chimiddleware.RequestID)    // Inject request ID
	r.Use(chimiddleware.RealIP)       // Get real client IP
	r.Use(chimiddleware.Heartbeat("/health")) // Health check endpoint

	// CORS configuration for React frontend
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"https://expensetracker-ashen-nu.vercel.app",
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		},
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
				// Read: all group members
				r.Get("/", txHandler.GetTransactions)
				r.Get("/annual", txHandler.GetAnnualSummary)
				r.Get("/wallet-summary", txHandler.GetWalletSummary)
				// Write: EDITOR+
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Post("/", txHandler.CreateTransaction)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Post("/batch", txHandler.CreateTransactionsBatch)
					r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Post("/batch-to-group", txHandler.CreateTransactionsBatchToGroup)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Patch("/{id}", txHandler.UpdateTransaction)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Delete("/{id}", txHandler.DeleteTransaction)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Delete("/batch", txHandler.DeleteTransactionsBatch)
			})

			// Category routes
			r.Route("/categories", func(r chi.Router) {
				// Read: all group members
				r.Get("/", catHandler.GetCategories)
				// Write: EDITOR+
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Post("/", catHandler.CreateCategory)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Patch("/{id}", catHandler.UpdateCategory)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Delete("/{id}", catHandler.DeleteCategory)
			})

			// Group & Activity routes (authenticated, all users)
			r.Route("/me", func(r chi.Router) {
				r.Get("/group", groupHandler.GetMyGroup)
				r.Patch("/group", groupHandler.UpdateGroupName)
				r.Get("/groups", groupHandler.ListMyGroups)
				r.Post("/groups", groupHandler.CreateMyGroup)
				r.Delete("/groups/{id}", groupHandler.DeleteMyGroup)
				r.Post("/switch-group", groupHandler.SwitchGroup)
				r.Get("/activity", groupHandler.GetActivityFeed)
				r.Post("/group/invite", groupHandler.CreateInvite)
				r.Post("/group/join", groupHandler.JoinGroup)
				r.Post("/group/leave", groupHandler.LeaveGroup)
			})

			// Loan routes (EDITOR+)
			r.Route("/loans", func(r chi.Router) {
				r.Get("/", loanHandler.ListLoans)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Post("/", loanHandler.CreateLoan)
				r.Get("/{id}", loanHandler.GetLoanDetail)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Patch("/{id}", loanHandler.UpdateLoan)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Delete("/{id}", loanHandler.DeleteLoan)
				r.Get("/{id}/entries", loanHandler.ListEntries)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Post("/{id}/entries", loanHandler.AddEntry)
				r.With(middleware.GroupRoleMiddleware(models.RoleEditor)).Delete("/{id}/entries/{eid}", loanHandler.DeleteEntry)
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
