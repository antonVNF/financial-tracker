package models

import "time"

type Transaction struct {
	ID          int64     `db:"id" json:"id"`
	Amount      float64   `db:"amount" json:"amount"`
	Type        string    `db:"type" json:"type"`
	Category    string    `db:"category" json:"category"`
	Description string    `db:"description" json:"description"`
	Date        time.Time `db:"date" json:"date"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type TransactionFilter struct {
	Category  string
	Type      string
	StartDate *time.Time
	EndDate   *time.Time
	Order     string
	Limit     int
	Offset    int
}

type CategoryStat struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}
