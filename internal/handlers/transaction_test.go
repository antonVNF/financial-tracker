package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"financial-tracker/internal/models"
	"financial-tracker/internal/repository"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	balance       float64
	getBalanceErr error
	createErr     error
	getAllResult  []models.Transaction
	getAllErr     error
	getByIdErr    error
	getByIdResult *models.Transaction
	updateErr     error
	deleteErr     error
	lastDeletedID int64
}

func (m *mockRepo) GetBalance(ctx context.Context) (float64, error) {
	return m.balance, m.getBalanceErr
}

func (m *mockRepo) Create(ctx context.Context, tx models.Transaction) error { return m.createErr }
func (m *mockRepo) GetAll(ctx context.Context, filter models.TransactionFilter) ([]models.Transaction, error) {
	return m.getAllResult, m.getAllErr
}
func (m *mockRepo) GetByID(ctx context.Context, id int64) (*models.Transaction, error) {
	return m.getByIdResult, m.getByIdErr
}
func (m *mockRepo) Update(ctx context.Context, id int64, tx models.Transaction) error {
	return m.updateErr
}
func (m *mockRepo) Delete(ctx context.Context, id int64) error {
	m.lastDeletedID = id
	return m.deleteErr
}
func (m *mockRepo) GetStatsByCategory(ctx context.Context, start, end time.Time) ([]models.CategoryStat, error) {
	return nil, nil
}

func TestGetBalanceHandler(t *testing.T) {
	mock := &mockRepo{balance: 123.45, getBalanceErr: nil}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewTransactionHandler(mock, logger).GetBalance()

	req := httptest.NewRequest("GET", "/balance", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]float64
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, 123.45, response["balance"])
}

func TestCreate(t *testing.T) {
	mock := &mockRepo{createErr: nil}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewTransactionHandler(mock, logger).Create()

	tx := models.Transaction{Amount: 100, Category: "Food", Date: time.Now()}
	body, _ := json.Marshal(tx)
	req := httptest.NewRequest("POST", "/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp models.Transaction
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, tx.Amount, resp.Amount)
}

func TestGetAll(t *testing.T) {
	expectedTransactions := []models.Transaction{
		{ID: 1, Amount: 500, Category: "Food"},
		{ID: 2, Amount: 100, Category: "Transport"},
	}
	mock := &mockRepo{getAllErr: nil, getAllResult: expectedTransactions}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewTransactionHandler(mock, logger).GetAll()

	req := httptest.NewRequest("GET", "/transactions?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []models.Transaction

	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedTransactions, resp)
}

func TestGetById(t *testing.T) {
	tx := &models.Transaction{ID: 1, Amount: 500, Category: "Food"}
	mock := &mockRepo{getByIdErr: nil, getByIdResult: tx}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewTransactionHandler(mock, logger).GetByID()

	req := httptest.NewRequest("GET", "/transactions/{id}", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.Transaction
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, *tx, resp)
}

func TestUpdate(t *testing.T) {
	updatedTx := &models.Transaction{
		ID:       1,
		Amount:   600,
		Category: "Food",
		Type:     "expense",
		Date:     time.Now(),
	}
	mock := &mockRepo{updateErr: nil, getByIdResult: updatedTx}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewTransactionHandler(mock, logger).Update()
	tx := &models.Transaction{
		Amount:   600,
		Category: "Food",
		Type:     "expense",
		Date:     time.Now(),
	}
	body, _ := json.Marshal(tx)
	req := httptest.NewRequest("PUT", "/transactions/{id}", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.Transaction
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, updatedTx.ID, resp.ID)
	assert.Equal(t, updatedTx.Amount, resp.Amount)
}

func TestDelete_Success(t *testing.T) {
	mock := &mockRepo{deleteErr: nil}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewTransactionHandler(mock, logger).Delete()

	req := httptest.NewRequest("DELETE", "/transactions/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, int64(1), mock.lastDeletedID)
}

func TestDelete_NotFound(t *testing.T) {
	mock := &mockRepo{deleteErr: repository.ErrNotFound}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewTransactionHandler(mock, logger).Delete()

	req := httptest.NewRequest("DELETE", "/transactions/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_InternalError(t *testing.T) {
	mock := &mockRepo{deleteErr: errors.New("db error")}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewTransactionHandler(mock, logger).Delete()

	req := httptest.NewRequest("DELETE", "/transactions/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
