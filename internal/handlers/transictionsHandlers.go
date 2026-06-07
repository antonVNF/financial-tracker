package handlers

import (
	"encoding/json"
	"errors"
	"financial-tracker/internal/models"
	"financial-tracker/internal/repository"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

func CreateTransactionHandler(repo *repository.PostgresRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var tx models.Transaction
		if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if tx.Amount <= 0 || tx.Date.IsZero() || tx.Category == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := repo.Create(r.Context(), tx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(tx); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	}
}

func GetTransactionsHandler(repo *repository.PostgresRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		startDate := r.URL.Query().Get("startDate")
		var parsedStartDate *time.Time
		if startDate != "" {
			date, err := time.Parse("2006-01-02", startDate)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			parsedStartDate = &date
		}

		endDate := r.URL.Query().Get("endDate")
		var parsedEndDate *time.Time
		if endDate != "" {
			date, err := time.Parse("2006-01-02", endDate)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			parsedEndDate = &date
		}

		limitStr := r.URL.Query().Get("limit")
		limit := 10
		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			limit = parsedLimit
		}
		offsetStr := r.URL.Query().Get("offset")
		offset := 0
		if offsetStr != "" {
			parsedOffset, err := strconv.Atoi(offsetStr)
			if err != nil || parsedOffset < 0 {
				http.Error(w, "Offset must be a non-negative integer", http.StatusBadRequest)
				return
			}
			offset = parsedOffset
		}
		var transactionsFilter = models.TransactionFilter{
			Category:  r.URL.Query().Get("category"),
			Type:      r.URL.Query().Get("type"),
			StartDate: parsedStartDate,
			EndDate:   parsedEndDate,
			Order:     r.URL.Query().Get("order"),
			Limit:     limit,
			Offset:    offset,
		}

		transactions, err := repo.GetAll(r.Context(), transactionsFilter)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(transactions); err != nil {
			log.Printf("failed to encode response: %v", err)
			return
		}
	}
}

func GetTransactionByIdHandler(repo *repository.PostgresRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		idStr := r.PathValue("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		tx, err := repo.GetByID(r.Context(), id)
		if err != nil {
			if err == pgx.ErrNoRows {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(tx); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	}
}

func UpdateTransactionHandler(repo *repository.PostgresRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		idStr := r.PathValue("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		var tx models.Transaction
		if err := json.NewDecoder((r.Body)).Decode(&tx); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if tx.Amount <= 0 || tx.Date.IsZero() || tx.Category == "" || (tx.Type != "income" && tx.Type != "expense") {
			http.Error(w, "invalid transaction data", http.StatusBadRequest)
			return
		}

		if err := repo.Update(r.Context(), id, tx); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		updatedTx, err := repo.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(updatedTx); err != nil {
			log.Printf("failed to encode response: %v", err)
			return
		}
	}
}

func DeleteTransactionHandler(repo *repository.PostgresRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		idStr := r.PathValue("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if err := repo.Delete(r.Context(), id); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func GetBalanceHandler(repo *repository.PostgresRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		balance, err := repo.GetBalance(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]float64{"balance": balance}); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	}
}

func GetCategoryStatsHandler(repo *repository.PostgresRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
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
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			startDate = t

		}
		if endDateStr != "" {
			t, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			endDate = t
		}

		stats, err := repo.GetStatsByCategory(r.Context(), startDate, endDate)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			log.Printf("failed to encode response: %v", err)
		}

		return

	}
}
