package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"

	"github.com/go-chi/chi/v5"
)

// CategoryHandler handles HTTP requests for category operations.
type CategoryHandler struct {
	repo *repository.CategoryRepository
}

// NewCategoryHandler creates a new handler with the given repository.
func NewCategoryHandler(repo *repository.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{repo: repo}
}

// GetCategories handles GET /api/categories?type={INCOME|EXPENSE}
// Returns all categories, optionally filtered by type.
func (h *CategoryHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	catType := strings.ToUpper(r.URL.Query().Get("type"))

	// Validate type parameter if provided
	if catType != "" && catType != "INCOME" && catType != "EXPENSE" {
		writeError(w, http.StatusBadRequest, "type must be INCOME or EXPENSE")
		return
	}

	// Injected by AuthMiddleware
	groupID := middleware.GetGroupID(r.Context())

	categories, err := h.repo.GetAll(r.Context(), groupID, catType)
	if err != nil {
		log.Printf("ERROR: GetCategories: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch categories")
		return
	}

	writeJSON(w, http.StatusOK, categories)
}

// CreateCategory handles POST /api/categories
// Creates a new category.
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	// Validate required fields
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.Type != models.TypeIncome && req.Type != models.TypeExpense {
		writeError(w, http.StatusBadRequest, "type must be INCOME or EXPENSE")
		return
	}

	// Injected by AuthMiddleware
	groupID := middleware.GetGroupID(r.Context())

	category, err := h.repo.Create(r.Context(), req, groupID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeError(w, http.StatusConflict, "a category with this name already exists for this type")
			return
		}
		log.Printf("ERROR: CreateCategory: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create category")
		return
	}

	writeJSON(w, http.StatusCreated, category)
}

// UpdateCategory handles PATCH /api/categories/{id}
// Updates a category's name.
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "category id is required")
		return
	}

	// Injected by AuthMiddleware
	groupID := middleware.GetGroupID(r.Context())

	// Fetch existing category
	existing, err := h.repo.GetByID(r.Context(), id, groupID)
	if err != nil {
		log.Printf("ERROR: UpdateCategory (fetch): %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch category")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}

	var req models.UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	updated, err := h.repo.Update(r.Context(), id, groupID, req)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeError(w, http.StatusConflict, "a category with this name already exists for this type")
			return
		}
		log.Printf("ERROR: UpdateCategory: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update category")
		return
	}
	if updated == nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// DeleteCategory handles DELETE /api/categories/{id}
// Prevents deletion if the category is in use by any transactions.
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "category id is required")
		return
	}

	// Injected by AuthMiddleware
	groupID := middleware.GetGroupID(r.Context())

	// Fetch existing category to get name and type for usage check
	existing, err := h.repo.GetByID(r.Context(), id, groupID)
	if err != nil {
		log.Printf("ERROR: DeleteCategory (fetch): %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch category")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}

	// Check if the category is used by any transactions
	inUse, err := h.repo.IsCategoryInUse(r.Context(), existing.Name, string(existing.Type), groupID)
	if err != nil {
		log.Printf("ERROR: DeleteCategory (usage check): %v", err)
		writeError(w, http.StatusInternalServerError, "failed to check category usage")
		return
	}
	if inUse {
		writeError(w, http.StatusConflict, "cannot delete category: it is used by existing transactions")
		return
	}

	if err := h.repo.Delete(r.Context(), id, groupID); err != nil {
		log.Printf("ERROR: DeleteCategory: %v", err)
		writeError(w, http.StatusNotFound, "category not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "category deleted"})
}
