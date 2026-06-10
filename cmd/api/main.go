package main

import (
	"context"
	"financial-tracker/internal/handlers"
	"financial-tracker/internal/repository"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	envCheck := godotenv.Load(".env")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

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

	var repo = repository.NewPostgresRepo(pool, logger)

	handler := handlers.NewTransactionHandler(repo, logger)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	http.HandleFunc("POST /transactions", handler.Create())
	http.HandleFunc("GET /transactions", handler.GetAll())
	http.HandleFunc("GET /transactions/{id}", handler.GetByID())
	http.HandleFunc("PUT /transactions/{id}", handler.Update())
	http.HandleFunc("DELETE /transactions/{id}", handler.Delete())
	http.HandleFunc("GET /balance", handler.GetBalance())
	http.HandleFunc("GET /stats/categories", handler.GetCategoryStats())

	srv := http.Server{Addr: ":8080", Handler: nil}

	go func() {
		log.Println("Listening on port 8080")

		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}

	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	<-ch
	log.Println("Shutting down gracefully...")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
