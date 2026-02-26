package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"webhook-ingestion-service/internal/httpapi"
	"webhook-ingestion-service/internal/store/postgres"
	"webhook-ingestion-service/internal/task"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dbURL := os.Getenv("DB_URL")
	secret := os.Getenv("WEBHOOK_SECRET")

	if dbURL == "" {
		log.Fatal("DB_URL is required")
	}
	if secret == "" {
		log.Fatal("WEBHOOK_SECRET is required")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Fast fail if DB unreachable
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	mux := http.NewServeMux()

	eventsRepo := postgres.NewEventRepo(db)
	svc := task.NewService(eventsRepo)
	// Readyz (DB check)
	mux.HandleFunc("GET /readyz", httpapi.ReadyzHandler(db))

	// Debug: process one due event
	deps := task.WorkerDeps{
		Repo:      eventsRepo,           // postgres.EventRepo implements task.WorkerRepository
		Processor: task.NoopProcessor{}, // dodamy poniżej
		Backoff:   task.DefaultBackoff(),
		Now:       func() time.Time { return time.Now().UTC() },
		// RNG optional; if nil, NextRetryAt will create one
	}

	mux.HandleFunc("POST /process/once", httpapi.ProcessOnceHandler(deps))
	mux.HandleFunc("POST /webhooks/provider", httpapi.WebhookProviderHandler(secret, nil, svc))
	mux.HandleFunc("GET /events/", httpapi.GetEventHandler(svc))

	// Placeholder: later we’ll add /webhooks/provider here
	// mux.HandleFunc("POST /webhooks/provider", ...)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Printf("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Printf("bye")
}

type DBPinger interface {
	PingContext(ctx context.Context) error
}

func healthHandler(db DBPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
