package main

import (
	"context"
	"financial-tracker/internal/handlers"
	"financial-tracker/internal/repository"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	envCheck := godotenv.Load(".env")

	if err := envCheck; err != nil {
		log.Fatal(err)
	}

	dbUrl := os.Getenv("DATABASE_URL")

	if dbUrl == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	cfg, err := pgxpool.ParseConfig(dbUrl)
	pool, err := repository.NewDB(ctx, cfg)
	if err != nil {
		fmt.Println("Error connecting to database:", err)
	}

	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	log.Println("Database connected and pinged successfully")

	var repo = repository.NewPostgresRepo(pool)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	http.HandleFunc("/transactions", handlers.CreateTransactionHandler(*repo))

	http.ListenAndServe(":5959", nil)
}
