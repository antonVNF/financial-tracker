package handlers

import (
	"encoding/json"
	"errors"
	"financial-tracker/internal/models"
	"financial-tracker/internal/repository"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type TransactionHandler struct {
	repo   repository.TransactionRepository
	logger *slog.Logger
}

func NewTransactionHandler(repo repository.TransactionRepository, logger *slog.Logger) *TransactionHandler {
	return &TransactionHandler{repo: repo, logger: logger}
}

func (h *TransactionHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodPost {
			h.logger.Warn("method not allowed", "method", r.Method, "url", r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var tx models.Transaction
		if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
			h.logger.Warn("failed to decode request body", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if tx.Amount <= 0 || tx.Date.IsZero() || tx.Category == "" {
			h.logger.Warn("validation failed", "amount", tx.Amount, "date", tx.Date, "category", tx.Category)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := h.repo.Create(r.Context(), tx); err != nil {
			h.logger.Error("failed to create transaction", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(tx); err != nil {
			h.logger.Error("failed to encode response", "error", err)
		}
	}
}

func (h *TransactionHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodGet {
			h.logger.Warn("method not allowed", "method", r.Method, "url", r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		query := r.URL.Query()
		startDate := query.Get("startDate")
		var parsedStartDate *time.Time
		if startDate != "" {
			date, err := time.Parse("2006-01-02", startDate)
			if err != nil {
				h.logger.Warn("invalid startDate", "value", startDate, "error", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			parsedStartDate = &date
		}

		endDate := query.Get("endDate")
		var parsedEndDate *time.Time
		if endDate != "" {
			date, err := time.Parse("2006-01-02", endDate)
			if err != nil {
				h.logger.Warn("invalid endDate", "value", endDate, "error", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			parsedEndDate = &date
		}

		limit := 10
		if limitStr := query.Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			} else {
				h.logger.Warn("invalid limit", "value", limitStr, "error", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		offset := 0
		if offsetStr := query.Get("offset"); offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			} else {
				h.logger.Warn("invalid offset", "value", offsetStr, "error", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		filter := models.TransactionFilter{
			Category:  query.Get("category"),
			Type:      query.Get("type"),
			StartDate: parsedStartDate,
			EndDate:   parsedEndDate,
			Order:     query.Get("order"),
			Limit:     limit,
			Offset:    offset,
		}

		transactions, err := h.repo.GetAll(r.Context(), filter)
		if err != nil {
			h.logger.Error("failed to fetch transactions", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(transactions); err != nil {
			h.logger.Error("failed to encode response", "error", err)
			return
		}
	}
}

func (h *TransactionHandler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodGet {
			h.logger.Warn("method not allowed", "method", r.Method, "url", r.URL.Path)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		idStr := r.PathValue("id")
		if idStr == "" {
			h.logger.Warn("missing id")
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.logger.Warn("invalid id", "value", idStr, "error", err)
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		tx, err := h.repo.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				h.logger.Warn("transaction not found", "id", id)
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			h.logger.Error("failed to get transaction by id", "id", id, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(tx); err != nil {
			h.logger.Error("failed to encode response", "error", err)
		}
	}
}

func (h *TransactionHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodPut {
			h.logger.Warn("method not allowed", "method", r.Method, "url", r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		idStr := r.PathValue("id")
		if idStr == "" {
			h.logger.Warn("missing id")
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.logger.Warn("invalid id", "value", idStr, "error", err)
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		var tx models.Transaction
		if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
			h.logger.Warn("failed to decode request body", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if tx.Amount <= 0 || tx.Date.IsZero() || tx.Category == "" || (tx.Type != "income" && tx.Type != "expense") {
			h.logger.Warn("validation failed", "amount", tx.Amount, "date", tx.Date, "category", tx.Category, "type", tx.Type)
			http.Error(w, "invalid transaction data", http.StatusBadRequest)
			return
		}

		if err := h.repo.Update(r.Context(), id, tx); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				h.logger.Warn("transaction not found for update", "id", id)
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			h.logger.Error("failed to update transaction", "id", id, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		updatedTx, err := h.repo.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				h.logger.Warn("transaction disappeared after update", "id", id)
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			h.logger.Error("failed to fetch updated transaction", "id", id, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(updatedTx); err != nil {
			h.logger.Error("failed to encode response", "error", err)
		}
	}
}

func (h *TransactionHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodDelete {
			h.logger.Warn("method not allowed", "method", r.Method, "url", r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		idStr := r.PathValue("id")
		if idStr == "" {
			h.logger.Warn("missing id")
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.logger.Warn("invalid id", "value", idStr, "error", err)
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if err := h.repo.Delete(r.Context(), id); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				h.logger.Warn("transaction not found for delete", "id", id)
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			h.logger.Error("failed to delete transaction", "id", id, "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *TransactionHandler) GetBalance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodGet {
			h.logger.Warn("method not allowed", "method", r.Method, "url", r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		balance, err := h.repo.GetBalance(r.Context())
		if err != nil {
			h.logger.Error("failed to get balance", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]float64{"balance": balance}); err != nil {
			h.logger.Error("failed to encode response", "error", err)
		}
	}
}

func (h *TransactionHandler) GetCategoryStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			h.logger.Warn("method not allowed", "method", r.Method, "url", r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		startDateStr := r.URL.Query().Get("start_date")
		endDateStr := r.URL.Query().Get("end_date")

		var startDate, endDate time.Time
		if startDateStr != "" {
			t, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				h.logger.Warn("invalid start_date", "value", startDateStr, "error", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			startDate = t
		}
		if endDateStr != "" {
			t, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				h.logger.Warn("invalid end_date", "value", endDateStr, "error", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			endDate = t
		}

		stats, err := h.repo.GetStatsByCategory(r.Context(), startDate, endDate)
		if err != nil {
			h.logger.Error("failed to get stats by category", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			h.logger.Error("failed to encode response", "error", err)
		}
	}
}
