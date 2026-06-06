package repository

import (
	"context"
	"financial-tracker/internal/models"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx models.Transaction) error

	GetAll(ctx context.Context, filter models.TransactionFilter) ([]models.Transaction, error)

	GetByID(ctx context.Context, id int64) (*models.Transaction, error)

	Update(ctx context.Context, id int64, tx models.Transaction) error

	Delete(ctx context.Context, id int64) error

	GetBalance(ctx context.Context) (float64, error)

	GetStatsByCategory(ctx context.Context, start, end time.Time) (map[string]float64, error)
}

type PostgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{
		db: pool,
	}
}

func (r *PostgresRepo) Create(ctx context.Context, tx models.Transaction) error {
	sql := `INSERT INTO transactions (amount, category, description, date, type, created_at) VALUES ($1, $2, $3, $4,$5, $6)`
	_, err := r.db.Exec(ctx, sql, tx.Amount, tx.Category, tx.Description, tx.Date, tx.Type, tx.CreatedAt)
	if err != nil {
		log.Printf("Error inserting transaction: %v", err)
	}

	return err
}

func (r *PostgresRepo) GetAll(ctx context.Context, filter models.TransactionFilter) ([]models.Transaction, error) {
	sql := `
		SELECT id, amount, type, category, description, date, created_at FROM transactions WHERE 1=1
	`
	args := []interface{}{}
	argsId := 1
	if filter.Category != "" {
		sql += fmt.Sprintf(" AND category = $%d", argsId)
		args = append(args, filter.Category)
		argsId++
	}

	if filter.Type != "" {
		sql += fmt.Sprintf(" AND type = $%d", argsId)
		args = append(args, filter.Type)
		argsId++
	}

	if filter.StartDate != nil && !filter.StartDate.IsZero() {
		sql += fmt.Sprintf(" AND date >= $%d", argsId)
		args = append(args, *filter.StartDate)
		argsId++
	}

	if filter.EndDate != nil && !filter.EndDate.IsZero() {
		sql += fmt.Sprintf(" AND date <= $%d", argsId)
		args = append(args, *filter.EndDate)
		argsId++
	}

	order := "DESC"
	if filter.Order == "asc" {
		order = "ASC"
	}
	sql += fmt.Sprintf(" ORDER BY date %s", order)

	if filter.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT $%d", argsId)
		args = append(args, filter.Limit)
		argsId++
	}
	if filter.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET $%d", argsId)
		args = append(args, filter.Offset)
		argsId++
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.Transaction{}
	for rows.Next() {
		var transaction models.Transaction
		err := rows.Scan(&transaction.ID, &transaction.Amount, &transaction.Type, &transaction.Category, &transaction.Description, &transaction.Date, &transaction.CreatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, transaction)
	}
	return result, nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id int64) (*models.Transaction, error) {
	sql := `SELECT id, amount, type, category, description, date, created_at FROM transactions WHERE id = $1`
	var transactions models.Transaction
	err := r.db.QueryRow(ctx, sql, id).Scan(
		&transactions.ID,
		&transactions.Amount,
		&transactions.Type,
		&transactions.Category,
		&transactions.Description,
		&transactions.Date,
		&transactions.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &transactions, nil
}

func (r *PostgresRepo) Update(ctx context.Context, id int64, tx models.Transaction) error {
	sql := `UPDATE  transactions
			SET amount = $2, type = $3, category = $4, description = $5, date = $6
			WHERE id = $1`

	result, err := r.db.Exec(ctx, sql, id, tx.Amount, tx.Type, tx.Category, tx.Description, tx.Date)
	if err != nil {
		log.Printf("Error updating transaction: %v", err)
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return err
}

func (r *PostgresRepo) Delete(ctx context.Context, id int64) error {
	sql := `DELETE FROM transactions WHERE id = $1`

	result, err := r.db.Exec(ctx, sql, id)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresRepo) GetBalance(ctx context.Context) (float64, error) {
	sql := `SELECT COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) FROM transactions`

	var balance float64

	err := r.db.QueryRow(ctx, sql).Scan(&balance)
	if err != nil {
		return 0, err
	}

	return balance, nil
}
