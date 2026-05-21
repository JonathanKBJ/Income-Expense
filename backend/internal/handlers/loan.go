package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"expense-tracker/internal/middleware"
	"expense-tracker/internal/models"
	"expense-tracker/internal/repository"

	"github.com/go-chi/chi/v5"
)

type LoanHandler struct {
	loanRepo  *repository.LoanRepository
	groupRepo *repository.GroupRepository
}

func NewLoanHandler(loanRepo *repository.LoanRepository, groupRepo *repository.GroupRepository) *LoanHandler {
	return &LoanHandler{loanRepo: loanRepo, groupRepo: groupRepo}
}

// ListLoans handles GET /api/loans
func (h *LoanHandler) ListLoans(w http.ResponseWriter, r *http.Request) {
	groupID := middleware.GetGroupID(r.Context())
	if groupID == "" {
		writeError(w, http.StatusForbidden, "group identification required")
		return
	}

	loans, err := h.loanRepo.ListByGroup(r.Context(), groupID)
	if err != nil {
		log.Printf("ERROR: ListLoans: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch loans")
		return
	}

	// Build summaries with a single batch entries query (avoids N+1)
	loanIDs := make([]string, len(loans))
	for i, loan := range loans {
		loanIDs[i] = loan.ID
	}
	entriesMap, err := h.loanRepo.ListEntriesByLoanIDs(r.Context(), loanIDs)
	if err != nil {
		entriesMap = make(map[string][]models.LoanEntry)
	}

	result := make([]models.LoanDetail, 0, len(loans))
	for _, loan := range loans {
		entries := entriesMap[loan.ID]
		if entries == nil {
			entries = []models.LoanEntry{}
		}
		detail := computeLoanDetail(loan, entries)
		result = append(result, detail)
	}

	writeJSON(w, http.StatusOK, result)
}

// GetLoanDetail handles GET /api/loans/{id}
func (h *LoanHandler) GetLoanDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	groupID := middleware.GetGroupID(r.Context())

	loan, err := h.loanRepo.GetByID(r.Context(), id, groupID)
	if err != nil || loan == nil {
		writeError(w, http.StatusNotFound, "loan not found")
		return
	}

	detail := h.buildLoanDetail(r.Context(), *loan)
	writeJSON(w, http.StatusOK, detail)
}

// CreateLoan handles POST /api/loans
func (h *LoanHandler) CreateLoan(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	groupID := middleware.GetGroupID(r.Context())

	var req models.CreateLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Counterparty == "" {
		writeError(w, http.StatusBadRequest, "counterparty is required")
		return
	}
	if req.Principal <= 0 {
		writeError(w, http.StatusBadRequest, "principal must be greater than 0")
		return
	}
	if req.StartDate == "" {
		writeError(w, http.StatusBadRequest, "startDate is required")
		return
	}

	loan := &models.Loan{
		Type:              req.Type,
		Name:              req.Name,
		Counterparty:      req.Counterparty,
		Principal:         req.Principal,
		TermMonths:        req.TermMonths,
		InstallmentAmount: req.InstallmentAmount,
		PaymentDay:        req.PaymentDay,
		InterestRate:      req.InterestRate,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		Notes:             req.Notes,
	}

	if err := h.loanRepo.Create(r.Context(), loan, userID, groupID); err != nil {
		log.Printf("ERROR: CreateLoan: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create loan")
		return
	}

	writeJSON(w, http.StatusCreated, loan)
}

// UpdateLoan handles PATCH /api/loans/{id}
func (h *LoanHandler) UpdateLoan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	groupID := middleware.GetGroupID(r.Context())

	var req models.UpdateLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if err := h.loanRepo.Update(r.Context(), id, groupID, req); err != nil {
		log.Printf("ERROR: UpdateLoan: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "loan updated"})
}

// DeleteLoan handles DELETE /api/loans/{id}
func (h *LoanHandler) DeleteLoan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	groupID := middleware.GetGroupID(r.Context())

	if err := h.loanRepo.Delete(r.Context(), id, groupID); err != nil {
		log.Printf("ERROR: DeleteLoan: %v", err)
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "loan deleted"})
}

// --- Loan Entries ---

// ListEntries handles GET /api/loans/{id}/entries
func (h *LoanHandler) ListEntries(w http.ResponseWriter, r *http.Request) {
	loanID := chi.URLParam(r, "id")
	groupID := middleware.GetGroupID(r.Context())

	// Verify loan belongs to this group
	_, err := h.loanRepo.GetByID(r.Context(), loanID, groupID)
	if err != nil {
		writeError(w, http.StatusNotFound, "loan not found")
		return
	}

	entries, err := h.loanRepo.ListEntries(r.Context(), loanID)
	if err != nil {
		log.Printf("ERROR: ListEntries: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch entries")
		return
	}

	writeJSON(w, http.StatusOK, entries)
}

// AddEntry handles POST /api/loans/{id}/entries
func (h *LoanHandler) AddEntry(w http.ResponseWriter, r *http.Request) {
	loanID := chi.URLParam(r, "id")
	groupID := middleware.GetGroupID(r.Context())

	// Verify loan belongs to this group
	_, err := h.loanRepo.GetByID(r.Context(), loanID, groupID)
	if err != nil {
		writeError(w, http.StatusNotFound, "loan not found")
		return
	}

	var req models.CreateLoanEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be greater than 0")
		return
	}
	if req.Date == "" {
		writeError(w, http.StatusBadRequest, "date is required")
		return
	}

	entry := &models.LoanEntry{
		LoanID:       loanID,
		EntryType:    req.EntryType,
		Amount:       req.Amount,
		Date:         req.Date,
		Description:  req.Description,
		ReceiptImage: req.ReceiptImage,
	}

	if err := h.loanRepo.AddEntry(r.Context(), entry); err != nil {
		log.Printf("ERROR: AddEntry: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to add entry")
		return
	}

	writeJSON(w, http.StatusCreated, entry)
}

// DeleteEntry handles DELETE /api/loans/{id}/entries/{eid}
func (h *LoanHandler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "eid")

	if err := h.loanRepo.DeleteEntry(r.Context(), entryID); err != nil {
		log.Printf("ERROR: DeleteEntry: %v", err)
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "entry deleted"})
}

// --- Helpers ---

// buildLoanDetail is the single-loan wrapper (used by GetLoanDetail).
func (h *LoanHandler) buildLoanDetail(ctx context.Context, loan models.Loan) models.LoanDetail {
	entries, err := h.loanRepo.ListEntries(ctx, loan.ID)
	if err != nil {
		entries = []models.LoanEntry{}
	}
	return computeLoanDetail(loan, entries)
}

// computeLoanDetail is a pure function that computes summary fields from loan + its entries.
// Used directly by ListLoans with batch-fetched entries to avoid N+1 queries.
func computeLoanDetail(loan models.Loan, entries []models.LoanEntry) models.LoanDetail {
	var totalWithdrawn, totalDeposited, totalInstallments float64
	for _, e := range entries {
		switch e.EntryType {
		case models.EntryWithdrawal:
			totalWithdrawn += e.Amount
		case models.EntryDeposit:
			totalDeposited += e.Amount
		case models.EntryInstallment:
			totalInstallments += e.Amount
		}
	}

	var outstanding float64
	if loan.Type == models.LoanBorrow {
		outstanding = loan.Principal - totalInstallments
	} else {
		outstanding = loan.Principal - totalDeposited
	}
	if outstanding < 0 {
		outstanding = 0
	}
	var progress float64
	if loan.Principal > 0 {
		progress = ((loan.Principal - outstanding) / loan.Principal) * 100
	}

	return models.LoanDetail{
		Loan:              loan,
		Entries:           entries,
		TotalWithdrawn:    totalWithdrawn,
		TotalDeposited:    totalDeposited,
		TotalInstallments: totalInstallments,
		Outstanding:       outstanding,
		ProgressPercent:   progress,
	}
}

// CalculateNextInstallmentDate computes the next installment due date.
func CalculateNextInstallmentDate(startDate string, paymentDay int, termMonths int) ([]string, bool) {
	if paymentDay < 1 || paymentDay > 31 || termMonths <= 0 {
		return nil, false
	}

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, false
	}

	var dates []string
	today := time.Now().Truncate(24 * time.Hour)
	hasOverdue := false

	for i := 0; i < termMonths; i++ {
		dueDate := start.AddDate(0, i, 0)
		// Set to payment day
		year, month, _ := dueDate.Date()
		// Handle months where payment day exceeds max days
		lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
		day := paymentDay
		if day > lastDay {
			day = lastDay
		}
		dueDate = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		dates = append(dates, dueDate.Format("2006-01-02"))
		if dueDate.Before(today) {
			hasOverdue = true
		}
	}

	return dates, hasOverdue
}
